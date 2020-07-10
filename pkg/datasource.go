package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/athena"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/patrickmn/go-cache"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/resource/httpadapter"
	"github.com/grafana/grafana-plugin-sdk-go/data"
)

type AwsAthenaDatasource struct {
	cache   *cache.Cache
	metrics *AwsAthenaMetrics
}

type AwsAthenaMetrics struct {
	queriesTotal          *prometheus.CounterVec
	dataScannedBytesTotal *prometheus.CounterVec
}

var (
	legendFormatPattern *regexp.Regexp
	clientCache         = make(map[string]*athena.Athena)
)

const (
	DEFAULT_MAX_ROWS          = 1000
	AWS_API_RESULT_MAX_LENGTH = 50
	QUERY_WAIT_COUNT          = 30
)

func init() {
	legendFormatPattern = regexp.MustCompile(`\{\{\s*(.+?)\s*\}\}`)
}

const metricNamespace = "aws_athena_datasource"

func NewDataSource(mux *http.ServeMux) *AwsAthenaDatasource {
	ds := &AwsAthenaDatasource{
		cache: cache.New(300*time.Second, 5*time.Second),
	}

	metrics := &AwsAthenaMetrics{
		queriesTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name:      "data_query_total",
				Help:      "data query counter",
				Namespace: metricNamespace,
			},
			[]string{"region"},
		),
		dataScannedBytesTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name:      "data_scanned_bytes_total",
				Help:      "scanned data size counter",
				Namespace: metricNamespace,
			},
			[]string{"region"},
		),
	}
	prometheus.MustRegister(metrics.queriesTotal)
	prometheus.MustRegister(metrics.dataScannedBytesTotal)
	ds.metrics = metrics

	mux.HandleFunc("/regions", ds.handleResourceRegions)
	mux.HandleFunc("/workgroup_names", ds.handleResourceWorkgroupNames)
	mux.HandleFunc("/named_query_names", ds.handleResourceNamedQueryNames)
	mux.HandleFunc("/named_query_queries", ds.handleResourceNamedQueryQueries)
	mux.HandleFunc("/query_executions", ds.handleResourceQueryExecutions)
	mux.HandleFunc("/query_executions_by_name", ds.handleResourceQueryExecutionsByName)

	return ds
}

func (ds *AwsAthenaDatasource) CheckHealth(ctx context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	res := &backend.CheckHealthResult{}

	if req.PluginContext.DataSourceInstanceSettings == nil {
		res.Status = backend.HealthStatusOk
		res.Message = "Plugin is Running"
		return res, nil
	}

	svc, err := ds.getClient(req.PluginContext.DataSourceInstanceSettings, "us-east-1")
	if err != nil {
		res.Status = backend.HealthStatusError
		res.Message = "Unable to create client"
		return res, nil
	}

	_, err = svc.ListNamedQueriesWithContext(ctx, &athena.ListNamedQueriesInput{})
	if err != nil {
		res.Status = backend.HealthStatusError
		res.Message = "Unable to call Athena API"
		return res, nil
	}

	res.Status = backend.HealthStatusOk
	res.Message = "Success"
	return res, nil
}

func (ds *AwsAthenaDatasource) QueryData(ctx context.Context, tsdbReq *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	responses := &backend.QueryDataResponse{
		Responses: map[string]backend.DataResponse{},
	}

	targets := make([]AwsAthenaQuery, 0)
	for _, query := range tsdbReq.Queries {
		target := AwsAthenaQuery{}
		if err := json.Unmarshal([]byte(query.JSON), &target); err != nil {
			return nil, err
		}
		target.From = query.TimeRange.From
		target.To = query.TimeRange.To

		svc, err := ds.getClient(tsdbReq.PluginContext.DataSourceInstanceSettings, target.Region)
		if err != nil {
			return nil, err
		}
		dsInfo, err := ds.getDsInfo(tsdbReq.PluginContext.DataSourceInstanceSettings, target.Region)
		if err != nil {
			return nil, err
		}
		if target.Region == "default" || target.Region == "" {
			target.Region = dsInfo.DefaultRegion
		}
		target.client = svc
		target.cache = ds.cache
		target.metrics = ds.metrics
		target.datasourceID = tsdbReq.PluginContext.DataSourceInstanceSettings.ID

		targets = append(targets, target)
	}

	for _, target := range targets {
		result, err := target.getQueryResults(ctx, tsdbReq.PluginContext)
		if err != nil {
			responses.Responses[target.RefId] = backend.DataResponse{
				Error: err,
			}
			continue
		}

		timeFormat := target.TimeFormat
		if timeFormat == "" {
			timeFormat = time.RFC3339Nano
		}

		if frames, err := parseResponse(result, target.RefId, target.From, target.To, target.TimestampColumn, target.ValueColumn, target.LegendFormat, timeFormat); err != nil {
			responses.Responses[target.RefId] = backend.DataResponse{
				Error: err,
			}
		} else {
			responses.Responses[target.RefId] = backend.DataResponse{
				Frames: append(responses.Responses[target.RefId].Frames, frames...),
			}
		}
	}

	return responses, nil
}

func parseResponse(resp *athena.GetQueryResultsOutput, refId string, from time.Time, to time.Time, timestampColumn string, valueColumn string, legendFormat string, timeFormat string) ([]*data.Frame, error) {
	warnings := []string{}

	timestampIndex := -1
	converters := make([]data.FieldConverter, len(resp.ResultSet.ResultSetMetadata.ColumnInfo))
	for i, c := range resp.ResultSet.ResultSetMetadata.ColumnInfo {
		fc, ok := converterMap[*c.Type]
		if !ok {
			warning := fmt.Sprintf("unknown column type: %s", *c.Type)
			warnings = append(warnings, warning)
			fc = stringFieldConverter
		}
		if *c.Name == timestampColumn {
			timestampIndex = i
			if *c.Type == "varchar" {
				fc = genTimeFieldConverter(timeFormat)
			}
		}
		if *c.Name == valueColumn {
			fc = floatFieldConverter
		}
		converters[i] = fc
	}

	if timestampIndex != -1 {
		n := 0
		// filter row without timestamp
		for _, row := range resp.ResultSet.Rows {
			if row.Data[timestampIndex].VarCharValue == nil {
				continue
			}
			resp.ResultSet.Rows[n] = row
			n++
		}
		resp.ResultSet.Rows = resp.ResultSet.Rows[:n]

		// sort by timestamp
		sort.Slice(resp.ResultSet.Rows, func(i, j int) bool {
			return *resp.ResultSet.Rows[i].Data[timestampIndex].VarCharValue < *resp.ResultSet.Rows[j].Data[timestampIndex].VarCharValue
		})
	}

	fieldNames := make([]string, 0)
	for _, column := range resp.ResultSet.ResultSetMetadata.ColumnInfo {
		fieldNames = append(fieldNames, *column.Name)
	}

	fm := make(map[string]*data.Frame)
	for _, row := range resp.ResultSet.Rows {
		kv := make(map[string]string)
		for columnIdx, cell := range row.Data {
			if cell == nil || cell.VarCharValue == nil {
				continue
			}
			columnName := *resp.ResultSet.ResultSetMetadata.ColumnInfo[columnIdx].Name
			if columnName == timestampColumn || columnName == valueColumn {
				continue
			}
			kv[columnName] = *cell.VarCharValue
		}
		name := formatLegend(kv, legendFormat)
		frame, ok := fm[name]
		if !ok {
			fTypes := make([]data.FieldType, len(converters))
			for i, fc := range converters {
				fTypes[i] = fc.OutputFieldType
			}
			frame = data.NewFrameOfFieldTypes("", 0, fTypes...)

			frame.RefID = refId
			frame.Name = name
			meta := make(map[string]interface{})
			meta["warnings"] = warnings
			frame.Meta = &data.FrameMeta{Custom: meta}
			if err := frame.SetFieldNames(fieldNames...); err != nil {
				return nil, err
			}
			fm[name] = frame
		}
		newRow := make([]interface{}, 0, len(row.Data))
		for columnIdx, cell := range row.Data {
			if cell == nil || cell.VarCharValue == nil {
				newRow = append(newRow, nil)
			} else if converters[columnIdx].Converter == nil {
				return nil, fmt.Errorf("converter should set")
			} else {
				convertedCell, err := converters[columnIdx].Converter(*cell.VarCharValue)
				if err != nil {
					return nil, err
				} else {
					newRow = append(newRow, convertedCell)
				}
			}
		}
		if len(newRow) == len(row.Data) {
			frame.AppendRow(newRow...)
		}
	}

	frames := make([]*data.Frame, 0)
	for _, frame := range fm {
		frames = append(frames, frame)
	}

	return frames, nil
}

var converterMap = map[string]data.FieldConverter{
	"varchar":   stringFieldConverter,
	"integer":   intFieldConverter,
	"tinyint":   intFieldConverter,
	"smallint":  intFieldConverter,
	"bigint":    intFieldConverter,
	"float":     floatFieldConverter,
	"double":    floatFieldConverter,
	"boolean":   boolFieldConverter,
	"date":      genTimeFieldConverter("2006-01-02"),
	"timestamp": genTimeFieldConverter("2006-01-02 15:04:05.000"),
}

func genTimeFieldConverter(timeFormat string) data.FieldConverter {
	return data.FieldConverter{
		OutputFieldType: data.FieldTypeNullableTime,
		Converter: func(v interface{}) (interface{}, error) {
			val, ok := v.(string)
			if !ok {
				return nil, fmt.Errorf("expected string input but got type %T", v)
			}
			if t, err := time.Parse(timeFormat, val); err != nil {
				return nil, err
			} else {
				return aws.Time(t), nil
			}
		},
	}
}

var stringFieldConverter = data.FieldConverter{
	OutputFieldType: data.FieldTypeNullableString,
	Converter: func(v interface{}) (interface{}, error) {
		val, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("expected string input but got type %T", v)
		}
		return aws.String(val), nil
	},
}

var intFieldConverter = data.FieldConverter{
	OutputFieldType: data.FieldTypeNullableInt64,
	Converter: func(v interface{}) (interface{}, error) {
		val, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("expected string input but got type %T", v)
		}
		if cval, err := strconv.ParseInt(val, 10, 64); err != nil {
			return nil, err
		} else {
			return aws.Int64(cval), nil
		}
	},
}

var floatFieldConverter = data.FieldConverter{
	OutputFieldType: data.FieldTypeNullableFloat64,
	Converter: func(v interface{}) (interface{}, error) {
		val, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("expected string input but got type %T", v)
		}
		if cval, err := strconv.ParseFloat(val, 64); err != nil {
			return nil, err
		} else {
			return aws.Float64(cval), nil
		}
	},
}

var boolFieldConverter = data.FieldConverter{
	OutputFieldType: data.FieldTypeNullableBool,
	Converter: func(v interface{}) (interface{}, error) {
		val, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("expected string input but got type %T", v)
		}
		return aws.Bool(val == "true"), nil
	},
}

func formatLegend(kv map[string]string, legendFormat string) string {
	if legendFormat == "" {
		l := make([]string, 0)
		for k, v := range kv {
			l = append(l, fmt.Sprintf("%s=\"%s\"", k, v))
		}
		return "{" + strings.Join(l, ",") + "}"
	}

	result := legendFormatPattern.ReplaceAllFunc([]byte(legendFormat), func(in []byte) []byte {
		columnName := strings.Replace(string(in), "{{", "", 1)
		columnName = strings.Replace(columnName, "}}", "", 1)
		columnName = strings.TrimSpace(columnName)
		if val, exists := kv[columnName]; exists {
			return []byte(val)
		}

		return in
	})

	return string(result)
}

func writeResult(rw http.ResponseWriter, path string, val interface{}, err error) {
	response := make(map[string]interface{})
	code := http.StatusOK
	if err != nil {
		response["error"] = err.Error()
		code = http.StatusBadRequest
	} else {
		response[path] = val
	}

	body, err := json.Marshal(response)
	if err != nil {
		body = []byte(err.Error())
		code = http.StatusInternalServerError
	}
	_, err = rw.Write(body)
	if err != nil {
		code = http.StatusInternalServerError
	}
	rw.WriteHeader(code)
}

func (ds *AwsAthenaDatasource) handleResourceRegions(rw http.ResponseWriter, req *http.Request) {
	backend.Logger.Debug("Received resource call", "url", req.URL.String(), "method", req.Method)
	if req.Method != http.MethodGet {
		return
	}

	ctx := req.Context()
	pluginContext := httpadapter.PluginConfigFromContext(ctx)

	svc, err := ds.getEC2Client(pluginContext.DataSourceInstanceSettings, "us-east-1")
	if err != nil {
		writeResult(rw, "?", nil, err)
		return
	}

	regions := []string{"default"}
	ro, err := svc.DescribeRegions(&ec2.DescribeRegionsInput{})
	if err != nil {
		// ignore error
		regions = append(regions, []string{
			"ap-east-1",
			"ap-northeast-1",
			"ap-northeast-2",
			"ap-northeast-3",
			"ap-south-1",
			"ap-southeast-1",
			"ap-southeast-2",
			"ca-central-1",
			"cn-north-1",
			"cn-northwest-1",
			"eu-central-1",
			"eu-north-1",
			"eu-west-1",
			"eu-west-2",
			"eu-west-3",
			"me-south-1",
			"sa-east-1",
			"us-east-1",
			"us-east-2",
			"us-gov-east-1",
			"us-gov-west-1",
			"us-iso-east-1",
			"us-isob-east-1",
			"us-west-1",
			"us-west-2",
		}...)
	} else {
		for _, r := range ro.Regions {
			regions = append(regions, *r.RegionName)
		}
	}
	sort.Strings(regions)

	writeResult(rw, "regions", regions, err)
}

func (ds *AwsAthenaDatasource) handleResourceWorkgroupNames(rw http.ResponseWriter, req *http.Request) {
	backend.Logger.Debug("Received resource call", "url", req.URL.String(), "method", req.Method)
	if req.Method != http.MethodGet {
		return
	}

	ctx := req.Context()
	pluginContext := httpadapter.PluginConfigFromContext(ctx)
	urlQuery := req.URL.Query()
	region := urlQuery.Get("region")

	svc, err := ds.getClient(pluginContext.DataSourceInstanceSettings, region)
	if err != nil {
		writeResult(rw, "?", nil, err)
		return
	}

	workgroupNames := make([]string, 0)
	li := &athena.ListWorkGroupsInput{}
	lo := &athena.ListWorkGroupsOutput{}
	err = svc.ListWorkGroupsPagesWithContext(ctx, li,
		func(page *athena.ListWorkGroupsOutput, lastPage bool) bool {
			lo.WorkGroups = append(lo.WorkGroups, page.WorkGroups...)
			return !lastPage
		})
	if err != nil {
		writeResult(rw, "?", nil, err)
		return
	}
	for _, w := range lo.WorkGroups {
		workgroupNames = append(workgroupNames, *w.Name)
	}

	writeResult(rw, "workgroup_names", workgroupNames, err)
}

func (ds *AwsAthenaDatasource) handleResourceNamedQueryNames(rw http.ResponseWriter, req *http.Request) {
	backend.Logger.Debug("Received resource call", "url", req.URL.String(), "method", req.Method)
	if req.Method != http.MethodGet {
		return
	}

	ctx := req.Context()
	pluginContext := httpadapter.PluginConfigFromContext(ctx)
	urlQuery := req.URL.Query()
	region := urlQuery.Get("region")
	workGroup := urlQuery.Get("workGroup")

	svc, err := ds.getClient(pluginContext.DataSourceInstanceSettings, region)
	if err != nil {
		writeResult(rw, "?", nil, err)
		return
	}

	data := make([]string, 0)
	var workGroupParam *string
	workGroupParam = nil
	if workGroup != "" {
		workGroupParam = &workGroup
	}
	li := &athena.ListNamedQueriesInput{
		WorkGroup: workGroupParam,
	}
	lo := &athena.ListNamedQueriesOutput{}
	if err := svc.ListNamedQueriesPagesWithContext(ctx, li,
		func(page *athena.ListNamedQueriesOutput, lastPage bool) bool {
			lo.NamedQueryIds = append(lo.NamedQueryIds, page.NamedQueryIds...)
			return !lastPage
		}); err != nil {
		writeResult(rw, "?", nil, err)
		return
	}
	for i := 0; i < len(lo.NamedQueryIds); i += AWS_API_RESULT_MAX_LENGTH {
		e := int64(math.Min(float64(i+AWS_API_RESULT_MAX_LENGTH), float64(len(lo.NamedQueryIds))))
		bi := &athena.BatchGetNamedQueryInput{NamedQueryIds: lo.NamedQueryIds[i:e]}
		bo, err := svc.BatchGetNamedQueryWithContext(ctx, bi)
		if err != nil {
			writeResult(rw, "?", nil, err)
			return
		}
		for _, q := range bo.NamedQueries {
			data = append(data, *q.Name)
		}
	}
	writeResult(rw, "named_query_names", data, err)
}

func (ds *AwsAthenaDatasource) getNamedQueryQueries(ctx context.Context, pluginContext backend.PluginContext, region string, workGroup string, pattern string) ([]string, error) {
	svc, err := ds.getClient(pluginContext.DataSourceInstanceSettings, region)
	if err != nil {
		return nil, err
	}

	data := make([]string, 0)
	var workGroupParam *string
	workGroupParam = nil
	if workGroup != "" {
		workGroupParam = &workGroup
	}
	r := regexp.MustCompile(pattern)
	li := &athena.ListNamedQueriesInput{
		WorkGroup: workGroupParam,
	}
	lo := &athena.ListNamedQueriesOutput{}
	err = svc.ListNamedQueriesPagesWithContext(ctx, li,
		func(page *athena.ListNamedQueriesOutput, lastPage bool) bool {
			lo.NamedQueryIds = append(lo.NamedQueryIds, page.NamedQueryIds...)
			return !lastPage
		})
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(lo.NamedQueryIds); i += AWS_API_RESULT_MAX_LENGTH {
		e := int64(math.Min(float64(i+AWS_API_RESULT_MAX_LENGTH), float64(len(lo.NamedQueryIds))))
		bi := &athena.BatchGetNamedQueryInput{NamedQueryIds: lo.NamedQueryIds[i:e]}
		bo, err := svc.BatchGetNamedQueryWithContext(ctx, bi)
		if err != nil {
			return nil, err
		}
		for _, q := range bo.NamedQueries {
			if r.MatchString(*q.Name) {
				data = append(data, *q.QueryString)
			}
		}
	}

	return data, nil
}

func (ds *AwsAthenaDatasource) handleResourceNamedQueryQueries(rw http.ResponseWriter, req *http.Request) {
	backend.Logger.Debug("Received resource call", "url", req.URL.String(), "method", req.Method)
	if req.Method != http.MethodGet {
		return
	}

	ctx := req.Context()
	pluginContext := httpadapter.PluginConfigFromContext(ctx)
	urlQuery := req.URL.Query()
	region := urlQuery.Get("region")
	pattern := urlQuery.Get("pattern")
	workGroup := urlQuery.Get("workGroup")

	data, err := ds.getNamedQueryQueries(ctx, pluginContext, region, workGroup, pattern)
	if err != nil {
		writeResult(rw, "?", nil, err)
		return
	}

	writeResult(rw, "named_query_queries", data, err)
}

func (ds *AwsAthenaDatasource) getQueryExecutions(ctx context.Context, pluginContext backend.PluginContext, region string, workGroup string, pattern string, to time.Time) ([]*athena.QueryExecution, error) {
	svc, err := ds.getClient(pluginContext.DataSourceInstanceSettings, region)
	if err != nil {
		return nil, err
	}

	var workGroupParam *string
	workGroupParam = nil
	if workGroup != "" {
		workGroupParam = &workGroup
	}
	r := regexp.MustCompile(pattern)

	var lastQueryExecutionID string
	lastQueryExecutionIDCacheKey := "LastQueryExecutionId/" + strconv.FormatInt(pluginContext.DataSourceInstanceSettings.ID, 10) + "/" + region + "/" + workGroup
	if item, _, found := ds.cache.GetWithExpiration(lastQueryExecutionIDCacheKey); found {
		if id, ok := item.(string); ok {
			lastQueryExecutionID = id
		}
	}

	li := &athena.ListQueryExecutionsInput{
		WorkGroup: workGroupParam,
	}
	lo := &athena.ListQueryExecutionsOutput{}
	err = svc.ListQueryExecutionsPagesWithContext(ctx, li,
		func(page *athena.ListQueryExecutionsOutput, lastPage bool) bool {
			lo.QueryExecutionIds = append(lo.QueryExecutionIds, page.QueryExecutionIds...)
			if *lo.QueryExecutionIds[0] == lastQueryExecutionID {
				return false // valid cache exists, get query executions from cache
			}
			return !lastPage
		})
	if err != nil {
		return nil, err
	}

	allQueryExecution := make([]*athena.QueryExecution, 0)
	QueryExecutionsCacheKey := "QueryExecutions/" + strconv.FormatInt(pluginContext.DataSourceInstanceSettings.ID, 10) + "/" + region + "/" + workGroup
	if *lo.QueryExecutionIds[0] == lastQueryExecutionID {
		if item, _, found := ds.cache.GetWithExpiration(QueryExecutionsCacheKey); found {
			if aqe, ok := item.([]*athena.QueryExecution); ok {
				allQueryExecution = aqe
			}
		}
	} else {
		for i := 0; i < len(lo.QueryExecutionIds); i += AWS_API_RESULT_MAX_LENGTH {
			e := int64(math.Min(float64(i+AWS_API_RESULT_MAX_LENGTH), float64(len(lo.QueryExecutionIds))))
			bi := &athena.BatchGetQueryExecutionInput{QueryExecutionIds: lo.QueryExecutionIds[i:e]}
			bo, err := svc.BatchGetQueryExecutionWithContext(ctx, bi)
			if err != nil {
				return nil, err
			}
			allQueryExecution = append(allQueryExecution, bo.QueryExecutions...)
		}

		ds.cache.Set(lastQueryExecutionIDCacheKey, *lo.QueryExecutionIds[0], time.Duration(24)*time.Hour)
		ds.cache.Set(QueryExecutionsCacheKey, allQueryExecution, time.Duration(24)*time.Hour)
	}

	fbo := make([]*athena.QueryExecution, 0)
	for _, q := range allQueryExecution {
		if *q.Status.State != "SUCCEEDED" {
			continue
		}
		if (*q.Status.CompletionDateTime).After(to) {
			continue
		}
		if r.MatchString(*q.Query) {
			fbo = append(fbo, q)
		}
	}
	sort.Slice(fbo, func(i, j int) bool {
		return fbo[i].Status.CompletionDateTime.After(*fbo[j].Status.CompletionDateTime)
	})
	return fbo, nil
}

func (ds *AwsAthenaDatasource) handleResourceQueryExecutions(rw http.ResponseWriter, req *http.Request) {
	backend.Logger.Debug("Received resource call", "url", req.URL.String(), "method", req.Method)
	if req.Method != http.MethodGet {
		return
	}

	ctx := req.Context()
	pluginContext := httpadapter.PluginConfigFromContext(ctx)
	urlQuery := req.URL.Query()
	region := urlQuery.Get("region")
	pattern := urlQuery.Get("pattern")
	limit, err := strconv.ParseInt(urlQuery.Get("limit"), 10, 64)
	if err != nil {
		writeResult(rw, "?", nil, err)
		return
	}
	workGroup := urlQuery.Get("workGroup")
	to, err := time.Parse(time.RFC3339, urlQuery.Get("to"))
	if err != nil {
		writeResult(rw, "?", nil, err)
		return
	}

	queryExecutions, err := ds.getQueryExecutions(ctx, pluginContext, region, workGroup, pattern, to)
	if err != nil {
		writeResult(rw, "?", nil, err)
		return
	}

	if limit != -1 {
		limit = int64(math.Min(float64(limit), float64(len(queryExecutions))))
		queryExecutions = queryExecutions[0:limit]
	}

	writeResult(rw, "query_executions", queryExecutions, err)
}

func (ds *AwsAthenaDatasource) handleResourceQueryExecutionsByName(rw http.ResponseWriter, req *http.Request) {
	backend.Logger.Info("handleResourceQueryExecutionsByName Received resource call", "url", req.URL.String(), "method", req.Method)
	if req.Method != http.MethodGet {
		return
	}

	ctx := req.Context()
	pluginContext := httpadapter.PluginConfigFromContext(ctx)
	urlQuery := req.URL.Query()
	region := urlQuery.Get("region")
	pattern := urlQuery.Get("pattern")
	limit, err := strconv.ParseInt(urlQuery.Get("limit"), 10, 64)
	if err != nil {
		writeResult(rw, "?", nil, err)
		return
	}
	workGroup := urlQuery.Get("workGroup")

	to, err := time.Parse(time.RFC3339, urlQuery.Get("to"))
	if err != nil {
		writeResult(rw, "?", nil, err)
		return
	}

	namedQueryQueries, err := ds.getNamedQueryQueries(ctx, pluginContext, region, workGroup, pattern)
	if err != nil {
		writeResult(rw, "?", nil, err)
		return
	}
	//if we did not find the named query based on the string, we return nil
	if len(namedQueryQueries) == 0 {
		writeResult(rw, "?", nil, errors.New("No query with that name found"))
		return
	}
	sql := namedQueryQueries[0]
	sql = strings.TrimRight(sql, " ")
	sql = strings.TrimRight(sql, ";")

	queryExecutions, err := ds.getQueryExecutions(ctx, pluginContext, region, workGroup, "^"+sql+"$", to)
	if err != nil {
		writeResult(rw, "?", nil, err)
		return
	}

	if limit != -1 {
		limit = int64(math.Min(float64(limit), float64(len(queryExecutions))))
		queryExecutions = queryExecutions[0:limit]
	}

	writeResult(rw, "query_executions_by_name", queryExecutions, err)
}

type Duration time.Duration

func (d *Duration) UnmarshalJSON(b []byte) error {
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	switch value := v.(type) {
	case string:
		if value == "" {
			value = "0s"
		}
		tmp, err := time.ParseDuration(value)
		if err != nil {
			return err
		}
		*d = Duration(tmp)
		return nil
	default:
		return errors.New("invalid duration")
	}
}
