import _ from 'lodash';
import TableModel from 'grafana/app/core/table_model';
import { DataSourceApi, DataSourceInstanceSettings, DataQueryRequest, DataQueryResponse } from '@grafana/ui';
import { AwsAthenaQuery, AwsAthenaOptions } from './types';
import { Observable } from 'rxjs';

export default class AwsAthenaDatasource extends DataSourceApi<AwsAthenaQuery, AwsAthenaOptions> {
  type: string;
  url: string;
  name: string;
  id: number;
  defaultRegion: string;
  q: any;
  $q: any;
  backendSrv: any;
  templateSrv: any;
  timeSrv: any;

  /** @ngInject */
  constructor(instanceSettings: DataSourceInstanceSettings<AwsAthenaOptions>, $q, backendSrv, templateSrv, timeSrv) {
    super(instanceSettings);
    this.type = instanceSettings.type;
    this.url = instanceSettings.url || '';
    this.name = instanceSettings.name;
    this.id = instanceSettings.id;
    this.defaultRegion = instanceSettings.jsonData.defaultRegion;
    this.q = $q;
    this.backendSrv = backendSrv;
    this.templateSrv = templateSrv;
    this.timeSrv = timeSrv;
  }

  query(options: DataQueryRequest<AwsAthenaQuery>): Observable<DataQueryResponse> {
    const query = this.buildQueryParameters(options);

    if (query.targets.length <= 0) {
      return this.q.when({ data: [] });
    }

    return this.doRequest({
      data: query,
    });
  }

  testDatasource() {
    return this.doMetricQueryRequest('named_query_names', {
      region: this.defaultRegion,
    })
      .then(res => {
        return this.q.when({ status: 'success', message: 'Data source is working', title: 'Success' });
      })
      .catch(err => {
        return { status: 'error', message: err.message, title: 'Error' };
      });
  }

  doRequest(options) {
    return this.backendSrv
      .datasourceRequest({
        url: '/api/tsdb/query',
        method: 'POST',
        data: {
          from: options.data.range.from.valueOf().toString(),
          to: options.data.range.to.valueOf().toString(),
          queries: options.data.targets,
        },
      })
      .then(result => {
        const res: any = [];
        for (const query of options.data.targets) {
          const r = result.data.results[query.refId];
          if (!_.isEmpty(r.series)) {
            _.forEach(r.series, s => {
              res.push({ target: s.name, datapoints: s.points });
            });
          }
          if (!_.isEmpty(r.tables)) {
            _.forEach(r.tables, t => {
              const table = new TableModel();
              table.columns = t.columns;
              table.rows = t.rows;
              res.push(table);
            });
          }
        }

        result.data = res;
        return result;
      });
  }

  buildQueryParameters(options) {
    const targets = options.targets
      .filter(target => !target.hide && target.queryExecutionId)
      .map(target => {
        return {
          refId: target.refId,
          hide: target.hide,
          datasourceId: this.id,
          queryType: 'timeSeriesQuery',
          format: target.format || 'timeserie',
          region: this.templateSrv.replace(target.region, options.scopedVars) || this.defaultRegion,
          timestampColumn: target.timestampColumn,
          valueColumn: target.valueColumn,
          legendFormat: target.legendFormat || '',
          timeFormat: target.timeFormat || '',
          inputs: this.templateSrv
            .replace(target.queryExecutionId, options.scopedVars)
            .split(/,/)
            .map(id => {
              return {
                queryExecutionId: id,
              };
            }),
        };
      });

    options.targets = targets;
    return options;
  }

  metricFindQuery(query) {
    let region;

    const namedQueryNamesQuery = query.match(/^named_query_names\(([^\)]+?)\)/);
    if (namedQueryNamesQuery) {
      region = namedQueryNamesQuery[1];
      return this.doMetricQueryRequest('named_query_names', {
        region: this.templateSrv.replace(region),
      });
    }

    const namedQueryQueryQuery = query.match(/^named_query_queries\(([^,]+?),\s?(.+)\)/);
    if (namedQueryQueryQuery) {
      region = namedQueryQueryQuery[1];
      const pattern = namedQueryQueryQuery[2];
      return this.doMetricQueryRequest('named_query_queries', {
        region: this.templateSrv.replace(region),
        pattern: this.templateSrv.replace(pattern, {}, 'regex'),
      });
    }

    const queryExecutionIdsQuery = query.match(/^query_execution_ids\(([^,]+?),\s?([^,]+?),\s?([^,]+)(,\s?.+)?\)/);
    if (queryExecutionIdsQuery) {
      region = queryExecutionIdsQuery[1];
      const limit = queryExecutionIdsQuery[2];
      const pattern = queryExecutionIdsQuery[3];
      let workGroup = queryExecutionIdsQuery[4];
      if (workGroup) {
        workGroup = workGroup.substr(1); //remove the comma
        workGroup = workGroup.trim();
      } else {
        workGroup = null;
      }

      return this.doMetricQueryRequest('query_execution_ids', {
        region: this.templateSrv.replace(region),
        limit: parseInt(this.templateSrv.replace(limit), 10),
        pattern: this.templateSrv.replace(pattern, {}, 'regex'),
        work_group: this.templateSrv.replace(workGroup),
      });
    }

    return this.q.when([]);
  }

  doMetricQueryRequest(subtype, parameters) {
    const range = this.timeSrv.timeRange();
    return this.backendSrv
      .datasourceRequest({
        url: '/api/tsdb/query',
        method: 'POST',
        data: {
          from: range.from.valueOf().toString(),
          to: range.to.valueOf().toString(),
          queries: [
            _.extend(
              {
                refId: 'metricFindQuery',
                datasourceId: this.id,
                queryType: 'metricFindQuery',
                subtype: subtype,
              },
              parameters
            ),
          ],
        },
      })
      .then(r => {
        return this.transformSuggestDataFromTable(r.data);
      });
  }

  transformSuggestDataFromTable(suggestData) {
    return _.map(suggestData.results['metricFindQuery'].tables[0].rows, v => {
      return {
        text: v[0],
        value: v[1],
      };
    });
  }
}
