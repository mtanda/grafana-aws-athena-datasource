package main

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/athena"

	"github.com/grafana/grafana_plugin_model/go/datasource"
	plugin "github.com/hashicorp/go-plugin"
)

type AwsAthenaDatasource struct {
	plugin.NetRPCUnsupportedPlugin
}

type Target struct {
	RefId           string
	Format          string
	Region          string
	Input           athena.GetQueryResultsInput
	TimestampColumn string
	ValueColumn     string
	LegendFormat    string
}

var (
	legendFormatPattern *regexp.Regexp
)

func init() {
	legendFormatPattern = regexp.MustCompile(`\{\{\s*(.+?)\s*\}\}`)
}

func (t *AwsAthenaDatasource) Query(ctx context.Context, tsdbReq *datasource.DatasourceRequest) (*datasource.DatasourceResponse, error) {
	response := &datasource.DatasourceResponse{}

	targets := make([]Target, 0)
	for _, query := range tsdbReq.Queries {
		target := Target{}
		if err := json.Unmarshal([]byte(query.ModelJson), &target); err != nil {
			return nil, err
		}
		targets = append(targets, target)
	}

	for _, target := range targets {
		awsCfg := &aws.Config{Region: aws.String(target.Region)}
		sess, err := session.NewSession(awsCfg)
		if err != nil {
			return nil, err
		}
		svc := athena.New(sess, awsCfg)

		resp, err := svc.GetQueryResults(&target.Input)
		if err != nil {
			return nil, err
		}

		switch target.Format {
		case "timeserie":
			r, err := parseTimeSeriesResponse(resp, target.RefId, target.TimestampColumn, target.ValueColumn, target.LegendFormat)
			if err != nil {
				return nil, err
			}
			response.Results = append(response.Results, r)
		case "table":
			r, err := parseTableResponse(resp, target.RefId)
			if err != nil {
				return nil, err
			}
			response.Results = append(response.Results, r)
		}
	}

	return response, nil
}

func parseTimeSeriesResponse(resp *athena.GetQueryResultsOutput, refId string, timestampColumn string, valueColumn string, legendFormat string) (*datasource.QueryResult, error) {
	series := make(map[string]*datasource.TimeSeries)

	var t time.Time
	var timestamp int64
	var value float64
	var err error
	for i, r := range resp.ResultSet.Rows {
		if i == 0 {
			continue // skip header
		}

		kv := make(map[string]string)
		for j, d := range r.Data {
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
				kv[columnName] = *d.VarCharValue
			}
		}

		name := formatLegend(kv, legendFormat)
		if (series[name]) == nil {
			series[name] = &datasource.TimeSeries{Name: name}
		}

		series[name].Points = append(series[name].Points, &datasource.Point{
			Timestamp: timestamp,
			Value:     value,
		})
	}

	s := make([]*datasource.TimeSeries, 0)
	for _, ss := range series {
		s = append(s, ss)
	}

	return &datasource.QueryResult{
		RefId:  refId,
		Series: s,
	}, nil
}

func parseTableResponse(resp *athena.GetQueryResultsOutput, refId string) (*datasource.QueryResult, error) {
	table := &datasource.Table{}

	for _, c := range resp.ResultSet.ResultSetMetadata.ColumnInfo {
		table.Columns = append(table.Columns, &datasource.TableColumn{Name: *c.Name})
	}
	for i, r := range resp.ResultSet.Rows {
		if i == 0 {
			continue // skip header
		}

		row := &datasource.TableRow{}
		for j, d := range r.Data {
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
