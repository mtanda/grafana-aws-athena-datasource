import _ from "lodash";
import TableModel from 'grafana/app/core/table_model';

export class AwsAthenaDatasource {
  type: string;
  url: string;
  name: string;
  id: string;
  defaultRegion: string;
  q: any;
  $q: any;
  backendSrv: any;
  templateSrv: any;
  timeSrv: any;

  constructor(instanceSettings, $q, backendSrv, templateSrv, timeSrv) {
    this.type = instanceSettings.type;
    this.url = instanceSettings.url;
    this.name = instanceSettings.name;
    this.id = instanceSettings.id;
    this.defaultRegion = instanceSettings.jsonData.defaultRegion;
    this.q = $q;
    this.backendSrv = backendSrv;
    this.templateSrv = templateSrv;
    this.timeSrv = timeSrv;
  }

  query(options) {
    let query = this.buildQueryParameters(options);
    query.targets = query.targets.filter(t => !t.hide);

    if (query.targets.length <= 0) {
      return this.q.when({ data: [] });
    }

    return this.doRequest({
      data: query
    });
  }

  testDatasource() {
    return this.doMetricQueryRequest('named_query_names', {
      region: this.defaultRegion,
    }).then(res => {
      return this.q.when({ status: "success", message: "Data source is working", title: "Success" });
    }).catch(err => {
      return { status: "error", message: err.message, title: "Error" };
    });
  }

  doRequest(options) {
    return this.backendSrv.datasourceRequest({
      url: '/api/tsdb/query',
      method: 'POST',
      data: {
        from: options.data.range.from.valueOf().toString(),
        to: options.data.range.to.valueOf().toString(),
        queries: options.data.targets,
      }
    }).then(result => {
      let res: any = [];
      for (const query of options.data.targets) {
        const r = result.data.results[query.refId];
        if (!_.isEmpty(r.series)) {
          _.forEach(r.series, s => {
            res.push({ target: s.name, datapoints: s.points });
          });
        }
        if (!_.isEmpty(r.tables)) {
          _.forEach(r.tables, t => {
            let table = new TableModel();
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
    let targets = _.map(options.targets, target => {
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
        input: {
          queryExecutionId: this.templateSrv.replace(target.queryExecutionId, options.scopedVars)
        }
      };
    });

    options.targets = targets;
    return options;
  }

  metricFindQuery(query) {
    let region;

    let namedQueryNamesQuery = query.match(/^named_query_names\(([^\)]+?)\)/);
    if (namedQueryNamesQuery) {
      region = namedQueryNamesQuery[1];
      return this.doMetricQueryRequest('named_query_names', {
        region: this.templateSrv.replace(region),
      });
    }

    let namedQueryQueryQuery = query.match(/^named_query_queries\(([^,]+?),\s?(.+)\)/);
    if (namedQueryQueryQuery) {
      region = namedQueryQueryQuery[1];
      let pattern = namedQueryQueryQuery[2];
      return this.doMetricQueryRequest('named_query_queries', {
        region: this.templateSrv.replace(region),
        pattern: this.templateSrv.replace(pattern, {}, 'regex'),
      });
    }

    let queryExecutionIdsQuery = query.match(/^query_execution_ids\(([^,]+?),\s?([^,]+?),\s?(.+)\)/);
    if (queryExecutionIdsQuery) {
      region = queryExecutionIdsQuery[1];
      let limit = queryExecutionIdsQuery[2];
      let pattern = queryExecutionIdsQuery[3];
      return this.doMetricQueryRequest('query_execution_ids', {
        region: this.templateSrv.replace(region),
        limit: parseInt(this.templateSrv.replace(limit), 10),
        pattern: this.templateSrv.replace(pattern, {}, 'regex'),
      });
    }

    return this.q.when([]);
  }

  doMetricQueryRequest(subtype, parameters) {
    var range = this.timeSrv.timeRange();
    return this.backendSrv.datasourceRequest({
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
      }
    }).then(r => {
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
