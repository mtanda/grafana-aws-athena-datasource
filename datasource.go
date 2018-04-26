package main

import (
	"encoding/json"
	"strconv"

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
	RefId     string
	QueryType string
	Region    string
	Input     athena.GetQueryResultsInput
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
		if target.QueryType != "table" {
			continue // only support table
		}

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

		r, err := parseResponse(resp, target.RefId)
		if err != nil {
			return nil, err
		}

		response.Results = append(response.Results, r)
	}

	return response, nil
}

func parseResponse(resp *athena.GetQueryResultsOutput, refId string) (*datasource.QueryResult, error) {
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
