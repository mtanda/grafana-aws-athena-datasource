package main

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/athena"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"golang.org/x/net/context"
	"gotest.tools/assert"
)

func TestAwsAthenaDatasource(t *testing.T) {
	t.Run("QueryData", func(t *testing.T) {
		t.Run("simple query", func(t *testing.T) {
			ctx := context.Background()
			q, _ := json.Marshal(Target{
				RefId:  "A",
				Format: "timeserie",
				Region: "us-east-1",
				Inputs: []athena.GetQueryResultsInput{
					athena.GetQueryResultsInput{
						QueryExecutionId: aws.String("43bcaae3-22f0-4dcf-a861-bbab3084d6a2"),
					},
				},
				TimestampColumn: "ts",
				ValueColumn:     "_col2",
				LegendFormat:    "",
				timeFormat:      "",
				From:            time.Now().Add(time.Duration(-24) * time.Hour),
				To:              time.Now(),
			})
			query := &backend.QueryDataRequest{
				PluginContext: backend.PluginContext{
					DataSourceInstanceSettings: &backend.DataSourceInstanceSettings{
						JSONData: []byte("{}"),
					},
				},
				Queries: []backend.DataQuery{
					backend.DataQuery{
						JSON: q,
					},
				},
			}
			ds := &AwsAthenaDatasource{}
			result, err := ds.QueryData(ctx, query)
			assert.Equal(t, nil, err)
			r := result.Responses["A"].Frames[0].Fields[0].CopyAt(0)
			s, ok := r.(string)
			assert.Equal(t, true, ok)
			assert.Equal(t, "2020-06-08T17:00:00.000000000Z", s)
		})
	})

	t.Run("parseResponse", func(t *testing.T) {
		t.Run("simple response", func(t *testing.T) {
			response := &athena.GetQueryResultsOutput{
				ResultSet: &athena.ResultSet{
					ResultSetMetadata: &athena.ResultSetMetadata{
						ColumnInfo: []*athena.ColumnInfo{
							&athena.ColumnInfo{
								Name:  aws.String("timestamp"),
								Label: aws.String("timestamp"),
								Type:  aws.String("timestamp"),
							},
							&athena.ColumnInfo{
								Name:  aws.String("value"),
								Label: aws.String("value"),
								Type:  aws.String("bigint"),
							},
						},
					},
					Rows: []*athena.Row{
						&athena.Row{
							Data: []*athena.Datum{
								&athena.Datum{
									VarCharValue: aws.String("2006-01-02 01:04:05.000"),
								},
								&athena.Datum{
									VarCharValue: aws.String("100"),
								},
							},
						},
						&athena.Row{
							Data: []*athena.Datum{
								&athena.Datum{
									VarCharValue: aws.String("2006-01-02 02:04:05.000"),
								},
								&athena.Datum{
									VarCharValue: aws.String("200"),
								},
							},
						},
					},
				},
			}

			from, _ := time.Parse("2006-01-02 15:04:05.000", "2006-01-02 00:04:05.000")
			to, _ := time.Parse("2006-01-02 15:04:05.000", "2006-01-02 23:04:05.000")
			frames, err := parseResponse(response, "A", from, to, "timestamp", "value", "", "2006-01-02 15:04:05.000")
			assert.Equal(t, nil, err)
			assert.Equal(t, "A", frames[0].RefID)
			assert.Equal(t, "timestamp", frames[0].Fields[0].Name)
			assert.Equal(t, "value", frames[0].Fields[1].Name)
			assert.Equal(t, float64(100), frames[0].Fields[1].At(0))
			assert.Equal(t, float64(200), frames[0].Fields[1].At(1))
			et1, _ := time.Parse("2006-01-02 15:04:05.000", "2006-01-02 01:04:05.000")
			t1, ok := frames[0].Fields[0].At(0).(*time.Time)
			assert.Equal(t, true, ok)
			assert.Equal(t, et1, *t1)
			et2, _ := time.Parse("2006-01-02 15:04:05.000", "2006-01-02 02:04:05.000")
			t2, ok := frames[0].Fields[0].At(1).(*time.Time)
			assert.Equal(t, true, ok)
			assert.Equal(t, et2, *t2)
		})
	})
}
