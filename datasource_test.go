package main

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/athena"
	"github.com/bmizerany/assert"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

func TestAwsAthenaDatasource(t *testing.T) {
	t.Run("QueryData", func(t *testing.T) {
		t.Run("simple query", func(t *testing.T) {
			ctx := context.Background()
			q, _ := json.Marshal(Target{
				RefId:     "A",
				QueryType: "timeSeriesQuery",
				Format:    "timeserie",
				Region:    "us-east-1",
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
}
