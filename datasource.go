package main

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/context"

	"github.com/aws/aws-sdk-go/service/athena"

	"github.com/grafana/grafana/pkg/components/simplejson"
	"github.com/grafana/grafana_plugin_model/go/datasource"
	plugin "github.com/hashicorp/go-plugin"
)

type AwsAthenaDatasource struct {
	plugin.NetRPCUnsupportedPlugin
}

type Target struct {
	RefId           string
	QueryType       string
	Format          string
	Region          string
	Input           athena.GetQueryResultsInput
	TimestampColumn string
	ValueColumn     string
	LegendFormat    string
}

var (
	legendFormatPattern *regexp.Regexp
	clientCache         = make(map[string]*athena.Athena)
)

func init() {
	legendFormatPattern = regexp.MustCompile(`\{\{\s*(.+?)\s*\}\}`)
}

func (t *AwsAthenaDatasource) Query(ctx context.Context, tsdbReq *datasource.DatasourceRequest) (*datasource.DatasourceResponse, error) {
	modelJson, err := simplejson.NewJson([]byte(tsdbReq.Queries[0].ModelJson))
	if err != nil {
		return nil, err
	}

	if modelJson.Get("queryType").MustString() == "metricFindQuery" {
		response, err := t.metricFindQuery(ctx, tsdbReq, modelJson, tsdbReq.TimeRange)
		if err != nil {
			return &datasource.DatasourceResponse{
				Results: []*datasource.QueryResult{
					{
						RefId: "metricFindQuery",
						Error: err.Error(),
					},
				},
			}, nil
		}
		return response, nil
	}

	response, err := t.handleQuery(tsdbReq)
	if err != nil {
		return &datasource.DatasourceResponse{
			Results: []*datasource.QueryResult{
				{
					Error: err.Error(),
				},
			},
		}, nil
	}

	return response, nil
}

func (t *AwsAthenaDatasource) handleQuery(tsdbReq *datasource.DatasourceRequest) (*datasource.DatasourceResponse, error) {
	response := &datasource.DatasourceResponse{}

	targets := make([]Target, 0)
	for _, query := range tsdbReq.Queries {
		target := Target{}
		if err := json.Unmarshal([]byte(query.ModelJson), &target); err != nil {
			return nil, err
		}
		targets = append(targets, target)
	}

	fromRaw, err := strconv.ParseInt(tsdbReq.TimeRange.FromRaw, 10, 64)
	if err != nil {
		return nil, err
	}
	from := time.Unix(fromRaw/1000, fromRaw%1000*1000*1000)
	toRaw, err := strconv.ParseInt(tsdbReq.TimeRange.ToRaw, 10, 64)
	if err != nil {
		return nil, err
	}
	to := time.Unix(toRaw/1000, toRaw%1000*1000*1000)
	for _, target := range targets {
		svc, err := t.getClient(tsdbReq.Datasource, target.Region)
		if err != nil {
			return nil, err
		}

		resp, err := svc.GetQueryResults(&target.Input)
		if err != nil {
			return nil, err
		}

		switch target.Format {
		case "timeserie":
			r, err := parseTimeSeriesResponse(resp, target.RefId, from, to, target.TimestampColumn, target.ValueColumn, target.LegendFormat)
			if err != nil {
				return nil, err
			}
			response.Results = append(response.Results, r)
		case "table":
			r, err := parseTableResponse(resp, target.RefId, from, to, target.TimestampColumn)
			if err != nil {
				return nil, err
			}
			response.Results = append(response.Results, r)
		}
	}

	return response, nil
}

func parseTimeSeriesResponse(resp *athena.GetQueryResultsOutput, refId string, from time.Time, to time.Time, timestampColumn string, valueColumn string, legendFormat string) (*datasource.QueryResult, error) {
	series := make(map[string]*datasource.TimeSeries)

	for i, r := range resp.ResultSet.Rows {
		var t time.Time
		var timestamp int64
		var value float64
		var err error

		if i == 0 {
			continue // skip header
		}

		kv := make(map[string]string)
		for j, d := range r.Data {
			if d == nil || d.VarCharValue == nil {
				continue
			}

			columnName := *resp.ResultSet.ResultSetMetadata.ColumnInfo[j].Name
			switch columnName {
			case timestampColumn:
				t, err = time.Parse(time.RFC3339Nano, *d.VarCharValue)
				if err != nil {
					return nil, err
				}
				timestamp = t.Unix() * 1000
			case valueColumn:
				value, err = strconv.ParseFloat(*d.VarCharValue, 64)
				if err != nil {
					return nil, err
				}
			default:
				if d != nil {
					kv[columnName] = *d.VarCharValue
				}
			}
		}

		if !t.IsZero() && (t.Before(from) || t.After(to)) {
			continue // out of range data
		}

		name := formatLegend(kv, legendFormat)
		if (series[name]) == nil {
			series[name] = &datasource.TimeSeries{Name: name, Tags: kv}
		}

		series[name].Points = append(series[name].Points, &datasource.Point{
			Timestamp: timestamp,
			Value:     value,
		})
	}

	s := make([]*datasource.TimeSeries, 0)
	for _, ss := range series {
		sort.Slice(ss.Points, func(i, j int) bool {
			return ss.Points[i].Timestamp < ss.Points[j].Timestamp
		})
		s = append(s, ss)
	}

	return &datasource.QueryResult{
		RefId:  refId,
		Series: s,
	}, nil
}

func parseTableResponse(resp *athena.GetQueryResultsOutput, refId string, from time.Time, to time.Time, timestampColumn string) (*datasource.QueryResult, error) {
	table := &datasource.Table{}

	for _, c := range resp.ResultSet.ResultSetMetadata.ColumnInfo {
		table.Columns = append(table.Columns, &datasource.TableColumn{Name: *c.Name})
	}
	for i, r := range resp.ResultSet.Rows {
		if i == 0 {
			continue // skip header
		}

		var timestamp time.Time
		var err error
		row := &datasource.TableRow{}
		for j, d := range r.Data {
			columnName := *resp.ResultSet.ResultSetMetadata.ColumnInfo[j].Name
			if columnName == timestampColumn {
				timestamp, err = time.Parse(time.RFC3339Nano, *d.VarCharValue)
				if err != nil {
					return nil, err
				}
			}

			if d == nil || d.VarCharValue == nil {
				row.Values = append(row.Values, &datasource.RowValue{Kind: datasource.RowValue_TYPE_NULL})
				continue
			}

			switch *resp.ResultSet.ResultSetMetadata.ColumnInfo[j].Type {
			case "integer":
				v, err := strconv.ParseInt(*d.VarCharValue, 10, 64)
				if err != nil {
					return nil, err
				}
				row.Values = append(row.Values, &datasource.RowValue{Kind: datasource.RowValue_TYPE_INT64, Int64Value: v})
			case "double":
				v, err := strconv.ParseFloat(*d.VarCharValue, 64)
				if err != nil {
					return nil, err
				}
				row.Values = append(row.Values, &datasource.RowValue{Kind: datasource.RowValue_TYPE_DOUBLE, DoubleValue: v})
			case "boolean":
				row.Values = append(row.Values, &datasource.RowValue{Kind: datasource.RowValue_TYPE_BOOL, BoolValue: *d.VarCharValue == "true"})
			case "varchar":
				fallthrough
			default:
				row.Values = append(row.Values, &datasource.RowValue{Kind: datasource.RowValue_TYPE_STRING, StringValue: *d.VarCharValue})
			}
		}

		if !timestamp.IsZero() && (timestamp.Before(from) || timestamp.After(to)) {
			continue // out of range data
		}

		table.Rows = append(table.Rows, row)
	}

	return &datasource.QueryResult{
		RefId:  refId,
		Tables: []*datasource.Table{table},
	}, nil
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

type suggestData struct {
	Text  string
	Value string
}

func (t *AwsAthenaDatasource) metricFindQuery(ctx context.Context, tsdbReq *datasource.DatasourceRequest, parameters *simplejson.Json, timeRange *datasource.TimeRange) (*datasource.DatasourceResponse, error) {
	region := parameters.Get("region").MustString()
	svc, err := t.getClient(tsdbReq.Datasource, region)
	if err != nil {
		return nil, err
	}

	subtype := parameters.Get("subtype").MustString()

	data := make([]suggestData, 0)
	switch subtype {
	case "named_query_names":
		li := &athena.ListNamedQueriesInput{}
		lo := &athena.ListNamedQueriesOutput{}
		err = svc.ListNamedQueriesPages(li,
			func(page *athena.ListNamedQueriesOutput, lastPage bool) bool {
				lo.NamedQueryIds = append(lo.NamedQueryIds, page.NamedQueryIds...)
				return !lastPage
			})
		if err != nil {
			return nil, err
		}
		for i := 0; i < len(lo.NamedQueryIds); i += 50 {
			e := int64(math.Min(float64(i+50), float64(len(lo.NamedQueryIds))))
			bi := &athena.BatchGetNamedQueryInput{NamedQueryIds: lo.NamedQueryIds[i:e]}
			bo, err := svc.BatchGetNamedQuery(bi)
			if err != nil {
				return nil, err
			}
			for _, q := range bo.NamedQueries {
				data = append(data, suggestData{Text: *q.Name, Value: *q.Name})
			}
		}
	case "named_query_queries":
		pattern := parameters.Get("pattern").MustString()
		r := regexp.MustCompile(pattern)
		li := &athena.ListNamedQueriesInput{}
		lo := &athena.ListNamedQueriesOutput{}
		err = svc.ListNamedQueriesPages(li,
			func(page *athena.ListNamedQueriesOutput, lastPage bool) bool {
				lo.NamedQueryIds = append(lo.NamedQueryIds, page.NamedQueryIds...)
				return !lastPage
			})
		if err != nil {
			return nil, err
		}
		for i := 0; i < len(lo.NamedQueryIds); i += 50 {
			e := int64(math.Min(float64(i+50), float64(len(lo.NamedQueryIds))))
			bi := &athena.BatchGetNamedQueryInput{NamedQueryIds: lo.NamedQueryIds[i:e]}
			bo, err := svc.BatchGetNamedQuery(bi)
			if err != nil {
				return nil, err
			}
			for _, q := range bo.NamedQueries {
				if r.MatchString(*q.Name) {
					data = append(data, suggestData{Text: *q.QueryString, Value: *q.QueryString})
				}
			}
		}
	case "query_execution_ids":
		toRaw, err := strconv.ParseInt(timeRange.ToRaw, 10, 64)
		if err != nil {
			return nil, err
		}
		to := time.Unix(toRaw/1000, toRaw%1000*1000*1000)

		pattern := parameters.Get("pattern").MustString()
		r := regexp.MustCompile(pattern)
		limit := parameters.Get("limit").MustInt()
		li := &athena.ListQueryExecutionsInput{}
		lo := &athena.ListQueryExecutionsOutput{}
		err = svc.ListQueryExecutionsPagesWithContext(ctx, li,
			func(page *athena.ListQueryExecutionsOutput, lastPage bool) bool {
				lo.QueryExecutionIds = append(lo.QueryExecutionIds, page.QueryExecutionIds...)
				return !lastPage
			})
		fbo := make([]*athena.QueryExecution, 0)
		for i := 0; i < len(lo.QueryExecutionIds); i += 50 {
			e := int64(math.Min(float64(i+50), float64(len(lo.QueryExecutionIds))))
			bi := &athena.BatchGetQueryExecutionInput{QueryExecutionIds: lo.QueryExecutionIds[i:e]}
			bo, err := svc.BatchGetQueryExecution(bi)
			if err != nil {
				return nil, err
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
		limit = int(math.Min(float64(limit), float64(len(fbo))))
		for _, q := range fbo[0:limit] {
			data = append(data, suggestData{Text: *q.QueryExecutionId, Value: *q.QueryExecutionId})
		}
	}

	table := t.transformToTable(data)

	return &datasource.DatasourceResponse{
		Results: []*datasource.QueryResult{
			{
				RefId:  "metricFindQuery",
				Tables: []*datasource.Table{table},
			},
		},
	}, nil
}

func (t *AwsAthenaDatasource) transformToTable(data []suggestData) *datasource.Table {
	table := &datasource.Table{
		Columns: make([]*datasource.TableColumn, 0),
		Rows:    make([]*datasource.TableRow, 0),
	}
	table.Columns = append(table.Columns, &datasource.TableColumn{Name: "text"})
	table.Columns = append(table.Columns, &datasource.TableColumn{Name: "value"})

	for _, r := range data {
		row := &datasource.TableRow{}
		row.Values = append(row.Values, &datasource.RowValue{Kind: datasource.RowValue_TYPE_STRING, StringValue: r.Text})
		row.Values = append(row.Values, &datasource.RowValue{Kind: datasource.RowValue_TYPE_STRING, StringValue: r.Value})
		table.Rows = append(table.Rows, row)
	}
	return table
}
