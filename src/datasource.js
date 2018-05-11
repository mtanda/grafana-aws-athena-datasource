import _ from "lodash";
import TableModel from 'app/core/table_model';

export class AwsAthenaDatasource {
  constructor(instanceSettings, $q, backendSrv, templateSrv) {
    this.type = instanceSettings.type;
    this.url = instanceSettings.url;
    this.name = instanceSettings.name;
    this.id = instanceSettings.id;
    this.defaultRegion = instanceSettings.jsonData.defaultRegion;
    this.q = $q;
    this.backendSrv = backendSrv;
    this.templateSrv = templateSrv;
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
    return this.q.when({ status: "success", message: "Data source is working", title: "Success" });
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
      let res = [];
      _.forEach(result.data.results, r => {
        if (!_.isEmpty(r.series)) {
          _.forEach(r.series, s => {
            res.push({ target: s.name, datapoints: s.points });
          })
        }
        if (!_.isEmpty(r.tables)) {
          _.forEach(r.tables, t => {
            let table = new TableModel()
            table.columns = t.columns
            table.rows = t.rows
            res.push(table);
          })
        }
      })

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
        format: target.type || 'timeserie',
        region: target.region || this.defaultRegion,
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
}
