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

	"github.com/aws/aws-sdk-go/service/athena"
	"github.com/patrickmn/go-cache"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/resource/httpadapter"
	"github.com/grafana/grafana-plugin-sdk-go/data"
)

type AwsAthenaDatasource struct {
	cache        *cache.Cache
	queriesTotal *prometheus.CounterVec
}

type Target struct {
	RefId           string
	Region          string
	Inputs          []athena.GetQueryResultsInput
	TimestampColumn string
	ValueColumn     string
	LegendFormat    string
	TimeFormat      string
	MaxRows         string
	CacheDuration   Duration
	From            time.Time
	To              time.Time
}

var (
	legendFormatPattern *regexp.Regexp
	clientCache         = make(map[string]*athena.Athena)
)

func init() {
	legendFormatPattern = regexp.MustCompile(`\{\{\s*(.+?)\s*\}\}`)
}

const metricNamespace = "aws_athena_datasource"

func NewDataSource(mux *http.ServeMux) *AwsAthenaDatasource {
	ds := &AwsAthenaDatasource{
		cache: cache.New(300*time.Second, 5*time.Second),
	}

	ds.queriesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name:      "data_query_total",
			Help:      "data query counter",
			Namespace: metricNamespace,
		},
		[]string{"region"},
	)
	prometheus.MustRegister(ds.queriesTotal)

	mux.HandleFunc("/named_query_names", ds.handleResourceNamedQueryNames)
	mux.HandleFunc("/named_query_queries", ds.handleResourceNamedQueryQueries)
	mux.HandleFunc("/query_execution_ids", ds.handleResourceQueryExecutionIds)

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

	_, err = svc.ListNamedQueries(&athena.ListNamedQueriesInput{})
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

	targets := make([]Target, 0)
	for _, query := range tsdbReq.Queries {
		target := Target{}
		if err := json.Unmarshal([]byte(query.JSON), &target); err != nil {
			return nil, err
		}
		target.From = query.TimeRange.From
		target.To = query.TimeRange.To
		targets = append(targets, target)
	}

	for _, target := range targets {
		svc, err := ds.getClient(tsdbReq.PluginContext.DataSourceInstanceSettings, target.Region)
		if err != nil {
			return nil, err
		}

		dedupe := true // TODO: add query option?
		if dedupe {
			bi := &athena.BatchGetQueryExecutionInput{}
			for _, input := range target.Inputs {
				bi.QueryExecutionIds = append(bi.QueryExecutionIds, input.QueryExecutionId)
			}
			bo, err := svc.BatchGetQueryExecution(bi)
			if err != nil {
				return nil, err
			}
			dupCheck := make(map[string]bool)
			target.Inputs = make([]athena.GetQueryResultsInput, 0)
			for _, q := range bo.QueryExecutions {
				if _, dup := dupCheck[*q.Query]; dup {
					continue
				}
				dupCheck[*q.Query] = true
				target.Inputs = append(target.Inputs, athena.GetQueryResultsInput{
					QueryExecutionId: q.QueryExecutionId,
				})
			}
		}

		maxRows, err := strconv.ParseInt(target.MaxRows, 10, 64)
		if err != nil {
			return nil, err
		}
		result := athena.GetQueryResultsOutput{
			ResultSet: &athena.ResultSet{
				Rows: make([]*athena.Row, 0),
			},
		}
		for _, input := range target.Inputs {
			var resp *athena.GetQueryResultsOutput

			cacheKey := target.Region + "/" + *input.QueryExecutionId + "/" + target.MaxRows
			if item, _, found := ds.cache.GetWithExpiration(cacheKey); found && target.CacheDuration > 0 {
				resp = item.(*athena.GetQueryResultsOutput)
			} else {
				err := svc.GetQueryResultsPagesWithContext(ctx, &input,
					func(page *athena.GetQueryResultsOutput, lastPage bool) bool {
						ds.queriesTotal.With(prometheus.Labels{"region": target.Region}).Inc()
						if resp == nil {
							resp = page
						} else {
							resp.ResultSet.Rows = append(resp.ResultSet.Rows, page.ResultSet.Rows...)
						}
						// result include extra header row, +1 here
						if maxRows != -1 && int64(len(resp.ResultSet.Rows)) > maxRows+1 {
							resp.ResultSet.Rows = resp.ResultSet.Rows[0 : maxRows+1]
							return false
						}
						return !lastPage
					})
				if err != nil {
					return nil, err
				}

				if target.CacheDuration > 0 {
					ds.cache.Set(cacheKey, resp, time.Duration(target.CacheDuration)*time.Second)
				}
			}

			result.ResultSet.ResultSetMetadata = resp.ResultSet.ResultSetMetadata
			result.ResultSet.Rows = append(result.ResultSet.Rows, resp.ResultSet.Rows[1:]...) // trim header row
		}

		timeFormat := target.TimeFormat
		if timeFormat == "" {
			timeFormat = time.RFC3339Nano
		}

		if frames, err := parseResponse(&result, target.RefId, target.From, target.To, target.TimestampColumn, target.ValueColumn, target.LegendFormat, timeFormat); err != nil {
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
			fc = data.AsStringFieldConverter
		}
		if *c.Name == timestampColumn && *c.Type == "varchar" {
			fc = genTimeFieldConverter(timeFormat)
			timestampIndex = i
		}
		if *c.Name == valueColumn {
			fc = floatFieldConverter
		}
		converters[i] = fc
	}

	if timestampIndex != -1 {
		sort.Slice(resp.ResultSet.Rows, func(i, j int) bool {
			return resp.ResultSet.Rows[i].Data[timestampIndex].VarCharValue != nil && resp.ResultSet.Rows[j].Data[timestampIndex].VarCharValue != nil && *resp.ResultSet.Rows[i].Data[timestampIndex].VarCharValue < *resp.ResultSet.Rows[j].Data[timestampIndex].VarCharValue
		})
	}

	fieldNames := make([]string, 0)
	for _, column := range resp.ResultSet.ResultSetMetadata.ColumnInfo {
		fieldNames = append(fieldNames, *column.Name)
	}

	ficm := make(map[string]*data.FrameInputConverter)
	for rowIdx, row := range resp.ResultSet.Rows {
		kv := make(map[string]string)
		var timestamp time.Time
		for columnIdx, cell := range row.Data {
			if cell == nil || cell.VarCharValue == nil {
				continue
			}
			columnName := *resp.ResultSet.ResultSetMetadata.ColumnInfo[columnIdx].Name
			if columnName == timestampColumn {
				var err error
				timestamp, err = time.Parse(time.RFC3339, *cell.VarCharValue)
				if err != nil {
					return nil, err
				}
			}
			if columnName == timestampColumn || columnName == valueColumn {
				continue
			}
			kv[columnName] = *cell.VarCharValue
		}
		if timestampColumn != "" && (timestamp.IsZero() || (timestamp.Before(from) || timestamp.After(to))) {
			continue // out of range data
		}
		name := formatLegend(kv, legendFormat)
		inputConverter, ok := ficm[name]
		if !ok {
			var err error
			inputConverter, err = data.NewFrameInputConverter(converters, len(resp.ResultSet.Rows))
			if err != nil {
				return nil, err
			}
			frame := inputConverter.Frame
			frame.RefID = refId
			frame.Name = name
			meta := make(map[string]interface{})
			meta["warnings"] = warnings
			frame.Meta = &data.FrameMeta{Custom: meta}
			err = inputConverter.Frame.SetFieldNames(fieldNames...)
			if err != nil {
				return nil, err
			}
			ficm[name] = inputConverter
		}
		for columnIdx, cell := range row.Data {
			if cell == nil || cell.VarCharValue == nil {
				continue
			}
			if err := inputConverter.Set(columnIdx, rowIdx, *cell.VarCharValue); err != nil {
				return nil, err
			}
		}
	}

	frames := make([]*data.Frame, 0)
	for _, fic := range ficm {
		frames = append(frames, fic.Frame)
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
				return &t, nil
			}
		},
	}
}

var stringFieldConverter = data.FieldConverter{
	OutputFieldType: data.FieldTypeString,
}

var intFieldConverter = data.FieldConverter{
	OutputFieldType: data.FieldTypeInt64,
	Converter: func(v interface{}) (interface{}, error) {
		val, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("expected string input but got type %T", v)
		}
		return strconv.ParseInt(val, 10, 64)
	},
}

var floatFieldConverter = data.FieldConverter{
	OutputFieldType: data.FieldTypeFloat64,
	Converter: func(v interface{}) (interface{}, error) {
		val, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("expected string input but got type %T", v)
		}
		return strconv.ParseFloat(val, 64)
	},
}

var boolFieldConverter = data.FieldConverter{
	OutputFieldType: data.FieldTypeBool,
	Converter: func(v interface{}) (interface{}, error) {
		val, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("expected string input but got type %T", v)
		}
		return val == "true", nil
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

func (ds *AwsAthenaDatasource) handleResourceNamedQueryNames(rw http.ResponseWriter, req *http.Request) {
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

	data := make([]string, 0)
	li := &athena.ListNamedQueriesInput{}
	lo := &athena.ListNamedQueriesOutput{}
	if err := svc.ListNamedQueriesPages(li,
		func(page *athena.ListNamedQueriesOutput, lastPage bool) bool {
			lo.NamedQueryIds = append(lo.NamedQueryIds, page.NamedQueryIds...)
			return !lastPage
		}); err != nil {
		writeResult(rw, "?", nil, err)
		return
	}
	for i := 0; i < len(lo.NamedQueryIds); i += 50 {
		e := int64(math.Min(float64(i+50), float64(len(lo.NamedQueryIds))))
		bi := &athena.BatchGetNamedQueryInput{NamedQueryIds: lo.NamedQueryIds[i:e]}
		bo, err := svc.BatchGetNamedQuery(bi)
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

	svc, err := ds.getClient(pluginContext.DataSourceInstanceSettings, region)
	if err != nil {
		writeResult(rw, "?", nil, err)
		return
	}

	data := make([]string, 0)
	r := regexp.MustCompile(pattern)
	li := &athena.ListNamedQueriesInput{}
	lo := &athena.ListNamedQueriesOutput{}
	err = svc.ListNamedQueriesPages(li,
		func(page *athena.ListNamedQueriesOutput, lastPage bool) bool {
			lo.NamedQueryIds = append(lo.NamedQueryIds, page.NamedQueryIds...)
			return !lastPage
		})
	if err != nil {
		writeResult(rw, "?", nil, err)
		return
	}
	for i := 0; i < len(lo.NamedQueryIds); i += 50 {
		e := int64(math.Min(float64(i+50), float64(len(lo.NamedQueryIds))))
		bi := &athena.BatchGetNamedQueryInput{NamedQueryIds: lo.NamedQueryIds[i:e]}
		bo, err := svc.BatchGetNamedQuery(bi)
		if err != nil {
			writeResult(rw, "?", nil, err)
			return
		}
		for _, q := range bo.NamedQueries {
			if r.MatchString(*q.Name) {
				data = append(data, *q.QueryString)
			}
		}
	}
	writeResult(rw, "named_query_queries", data, err)
}

func (ds *AwsAthenaDatasource) handleResourceQueryExecutionIds(rw http.ResponseWriter, req *http.Request) {
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
	r := regexp.MustCompile(pattern)
	li := &athena.ListQueryExecutionsInput{
		WorkGroup: workGroupParam,
	}
	lo := &athena.ListQueryExecutionsOutput{}
	err = svc.ListQueryExecutionsPagesWithContext(ctx, li,
		func(page *athena.ListQueryExecutionsOutput, lastPage bool) bool {
			lo.QueryExecutionIds = append(lo.QueryExecutionIds, page.QueryExecutionIds...)
			return !lastPage
		})
	if err != nil {
		writeResult(rw, "?", nil, err)
		return
	}
	fbo := make([]*athena.QueryExecution, 0)
	for i := 0; i < len(lo.QueryExecutionIds); i += 50 {
		e := int64(math.Min(float64(i+50), float64(len(lo.QueryExecutionIds))))
		bi := &athena.BatchGetQueryExecutionInput{QueryExecutionIds: lo.QueryExecutionIds[i:e]}
		bo, err := svc.BatchGetQueryExecution(bi)
		if err != nil {
			writeResult(rw, "?", nil, err)
			return
		}
		for _, q := range bo.QueryExecutions {
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
	}
	sort.Slice(fbo, func(i, j int) bool {
		return fbo[i].Status.CompletionDateTime.After(*fbo[j].Status.CompletionDateTime)
	})
	limit = int64(math.Min(float64(limit), float64(len(fbo))))
	for _, q := range fbo[0:limit] {
		data = append(data, *q.QueryExecutionId)
	}
	writeResult(rw, "query_execution_ids", data, err)
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
