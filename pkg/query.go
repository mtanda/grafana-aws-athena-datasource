package main

import (
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/athena"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/patrickmn/go-cache"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/net/context"
)

type AwsAthenaQuery struct {
	client          *athena.Athena
	cache           *cache.Cache
	metrics         *AwsAthenaMetrics
	RefId           string
	Region          string
	Inputs          []athena.GetQueryResultsInput
	TimestampColumn string
	ValueColumn     string
	LegendFormat    string
	TimeFormat      string
	MaxRows         string
	CacheDuration   Duration
	WorkGroup       string
	QueryString     string
	OutputLocation  string
	From            time.Time
	To              time.Time
}

func (query *AwsAthenaQuery) getQueryResults(ctx context.Context, pluginContext backend.PluginContext, target AwsAthenaQuery) (*athena.GetQueryResultsOutput, error) {
	var err error

	waitQueryExecutionIds := make([]*string, 0)
	if target.QueryString == "" {
		dedupe := true // TODO: add query option?
		if dedupe {
			bi := &athena.BatchGetQueryExecutionInput{}
			for _, input := range target.Inputs {
				bi.QueryExecutionIds = append(bi.QueryExecutionIds, input.QueryExecutionId)
			}
			bo, err := query.client.BatchGetQueryExecutionWithContext(ctx, bi)
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
	} else {
		workgroup, err := query.getWorkgroup(ctx, pluginContext, target.Region, target.WorkGroup)
		if err != nil {
			return nil, err
		}
		if workgroup.WorkGroup.Configuration.BytesScannedCutoffPerQuery == nil {
			return nil, fmt.Errorf("should set scan data limit")
		}

		// cache instant query result by query string
		var queryExecutionID string
		cacheKey := "StartQueryExecution/" + strconv.FormatInt(pluginContext.DataSourceInstanceSettings.ID, 10) + "/" + target.Region + "/" + target.QueryString + "/" + target.MaxRows
		if item, _, found := query.cache.GetWithExpiration(cacheKey); found && target.CacheDuration > 0 {
			if id, ok := item.(string); ok {
				queryExecutionID = id
			}
		} else {
			si := &athena.StartQueryExecutionInput{
				QueryString: aws.String(target.QueryString),
				WorkGroup:   aws.String(target.WorkGroup),
				ResultConfiguration: &athena.ResultConfiguration{
					OutputLocation: aws.String(target.OutputLocation),
				},
			}
			so, err := query.client.StartQueryExecutionWithContext(ctx, si)
			if err != nil {
				return nil, err
			}
			queryExecutionID = *so.QueryExecutionId
			if target.CacheDuration > 0 {
				query.cache.Set(cacheKey, queryExecutionID, time.Duration(target.CacheDuration)*time.Second)
			}
			waitQueryExecutionIds = append(waitQueryExecutionIds, &queryExecutionID)
		}
		target.Inputs = append(target.Inputs, athena.GetQueryResultsInput{
			QueryExecutionId: aws.String(queryExecutionID),
		})
	}

	// wait until query completed
	if len(waitQueryExecutionIds) > 0 {
		for i := 0; i < QUERY_WAIT_COUNT; i++ {
			completeCount := 0
			bi := &athena.BatchGetQueryExecutionInput{QueryExecutionIds: waitQueryExecutionIds}
			bo, err := query.client.BatchGetQueryExecutionWithContext(ctx, bi)
			if err != nil {
				return nil, err
			}
			for _, e := range bo.QueryExecutions {
				// TODO: add warning for FAILED or CANCELLED
				if !(*e.Status.State == "QUEUED" || *e.Status.State == "RUNNING") {
					completeCount++
				}
			}
			if len(waitQueryExecutionIds) == completeCount {
				for _, e := range bo.QueryExecutions {
					query.metrics.dataScannedBytesTotal.With(prometheus.Labels{"region": target.Region}).Add(float64(*e.Statistics.DataScannedInBytes))
				}
				break
			} else {
				time.Sleep(1 * time.Second)
			}
		}
	}

	maxRows := int64(DEFAULT_MAX_ROWS)
	if target.MaxRows != "" {
		maxRows, err = strconv.ParseInt(target.MaxRows, 10, 64)
		if err != nil {
			return nil, err
		}
	}
	result := athena.GetQueryResultsOutput{
		ResultSet: &athena.ResultSet{
			Rows: make([]*athena.Row, 0),
		},
	}
	for _, input := range target.Inputs {
		var resp *athena.GetQueryResultsOutput

		cacheKey := "QueryResults/" + strconv.FormatInt(pluginContext.DataSourceInstanceSettings.ID, 10) + "/" + target.Region + "/" + *input.QueryExecutionId + "/" + target.MaxRows
		if item, _, found := query.cache.GetWithExpiration(cacheKey); found && target.CacheDuration > 0 {
			if r, ok := item.(*athena.GetQueryResultsOutput); ok {
				resp = r
			}
		} else {
			err := query.client.GetQueryResultsPagesWithContext(ctx, &input,
				func(page *athena.GetQueryResultsOutput, lastPage bool) bool {
					query.metrics.queriesTotal.With(prometheus.Labels{"region": target.Region}).Inc()
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
				query.cache.Set(cacheKey, resp, time.Duration(target.CacheDuration)*time.Second)
			}
		}

		result.ResultSet.ResultSetMetadata = resp.ResultSet.ResultSetMetadata
		result.ResultSet.Rows = append(result.ResultSet.Rows, resp.ResultSet.Rows[1:]...) // trim header row
	}

	return &result, nil
}

func (query *AwsAthenaQuery) getWorkgroup(ctx context.Context, pluginContext backend.PluginContext, region string, workGroup string) (*athena.GetWorkGroupOutput, error) {
	WorkgroupCacheKey := "Workgroup/" + strconv.FormatInt(pluginContext.DataSourceInstanceSettings.ID, 10) + "/" + region + "/" + workGroup
	if item, _, found := query.cache.GetWithExpiration(WorkgroupCacheKey); found {
		if workgroup, ok := item.(*athena.GetWorkGroupOutput); ok {
			return workgroup, nil
		}
	}
	workgroup, err := query.client.GetWorkGroupWithContext(ctx, &athena.GetWorkGroupInput{WorkGroup: aws.String(workGroup)})
	if err != nil {
		return nil, err
	}
	query.cache.Set(WorkgroupCacheKey, workgroup, time.Duration(5)*time.Minute)

	return workgroup, nil
}
