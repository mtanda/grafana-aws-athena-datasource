"use strict";

System.register(["lodash", "app/core/table_model"], function (_export, _context) {
  "use strict";

  var _, TableModel, _createClass, AwsAthenaDatasource;

  function _classCallCheck(instance, Constructor) {
    if (!(instance instanceof Constructor)) {
      throw new TypeError("Cannot call a class as a function");
    }
  }

  return {
    setters: [function (_lodash) {
      _ = _lodash.default;
    }, function (_appCoreTable_model) {
      TableModel = _appCoreTable_model.default;
    }],
    execute: function () {
      _createClass = function () {
        function defineProperties(target, props) {
          for (var i = 0; i < props.length; i++) {
            var descriptor = props[i];
            descriptor.enumerable = descriptor.enumerable || false;
            descriptor.configurable = true;
            if ("value" in descriptor) descriptor.writable = true;
            Object.defineProperty(target, descriptor.key, descriptor);
          }
        }

        return function (Constructor, protoProps, staticProps) {
          if (protoProps) defineProperties(Constructor.prototype, protoProps);
          if (staticProps) defineProperties(Constructor, staticProps);
          return Constructor;
        };
      }();

      _export("AwsAthenaDatasource", AwsAthenaDatasource = function () {
        function AwsAthenaDatasource(instanceSettings, $q, backendSrv, templateSrv, timeSrv) {
          _classCallCheck(this, AwsAthenaDatasource);

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

        _createClass(AwsAthenaDatasource, [{
          key: "query",
          value: function query(options) {
            var query = this.buildQueryParameters(options);
            query.targets = query.targets.filter(function (t) {
              return !t.hide;
            });

            if (query.targets.length <= 0) {
              return this.q.when({ data: [] });
            }

            return this.doRequest({
              data: query
            });
          }
        }, {
          key: "testDatasource",
          value: function testDatasource() {
            return this.q.when({ status: "success", message: "Data source is working", title: "Success" });
          }
        }, {
          key: "doRequest",
          value: function doRequest(options) {
            return this.backendSrv.datasourceRequest({
              url: '/api/tsdb/query',
              method: 'POST',
              data: {
                from: options.data.range.from.valueOf().toString(),
                to: options.data.range.to.valueOf().toString(),
                queries: options.data.targets
              }
            }).then(function (result) {
              var res = [];
              _.forEach(result.data.results, function (r) {
                if (!_.isEmpty(r.series)) {
                  _.forEach(r.series, function (s) {
                    res.push({ target: s.name, datapoints: s.points });
                  });
                }
                if (!_.isEmpty(r.tables)) {
                  _.forEach(r.tables, function (t) {
                    var table = new TableModel();
                    table.columns = t.columns;
                    table.rows = t.rows;
                    res.push(table);
                  });
                }
              });

              result.data = res;
              return result;
            });
          }
        }, {
          key: "buildQueryParameters",
          value: function buildQueryParameters(options) {
            var _this = this;

            var targets = _.map(options.targets, function (target) {
              return {
                refId: target.refId,
                hide: target.hide,
                datasourceId: _this.id,
                queryType: 'timeSeriesQuery',
                format: target.type || 'timeserie',
                region: target.region || _this.defaultRegion,
                timestampColumn: target.timestampColumn,
                valueColumn: target.valueColumn,
                legendFormat: target.legendFormat || '',
                input: {
                  queryExecutionId: _this.templateSrv.replace(target.queryExecutionId, options.scopedVars)
                }
              };
            });

            options.targets = targets;
            return options;
          }
        }, {
          key: "metricFindQuery",
          value: function metricFindQuery(query) {
            var region = void 0;

            var namedQueryNamesQuery = query.match(/^named_query_names\(([^\)]+?)\)/);
            if (namedQueryNamesQuery) {
              region = namedQueryNamesQuery[1];
              return this.doMetricQueryRequest('named_query_names', {
                region: this.templateSrv.replace(region)
              });
            }

            var namedQueryQueryQuery = query.match(/^named_query_queries\(([^,]+?),\s?(.+)\)/);
            if (namedQueryQueryQuery) {
              region = namedQueryQueryQuery[1];
              var pattern = namedQueryQueryQuery[2];
              return this.doMetricQueryRequest('named_query_queries', {
                region: this.templateSrv.replace(region),
                pattern: this.templateSrv.replace(pattern, {}, 'regex')
              });
            }

            var queryExecutionIdsQuery = query.match(/^query_execution_ids\(([^,]+?),\s?(.+)\)/);
            if (queryExecutionIdsQuery) {
              region = queryExecutionIdsQuery[1];
              var _pattern = queryExecutionIdsQuery[2];
              return this.doMetricQueryRequest('query_execution_ids', {
                region: this.templateSrv.replace(region),
                pattern: this.templateSrv.replace(_pattern, {}, 'regex')
              });
            }

            return this.$q.when([]);
          }
        }, {
          key: "doMetricQueryRequest",
          value: function doMetricQueryRequest(subtype, parameters) {
            var _this2 = this;

            var range = this.timeSrv.timeRange();
            return this.backendSrv.datasourceRequest({
              url: '/api/tsdb/query',
              method: 'POST',
              data: {
                from: range.from.valueOf().toString(),
                to: range.to.valueOf().toString(),
                queries: [_.extend({
                  refId: 'metricFindQuery',
                  datasourceId: this.id,
                  queryType: 'metricFindQuery',
                  subtype: subtype
                }, parameters)]
              }
            }).then(function (r) {
              return _this2.transformSuggestDataFromTable(r.data);
            });
          }
        }, {
          key: "transformSuggestDataFromTable",
          value: function transformSuggestDataFromTable(suggestData) {
            return _.map(suggestData.results['metricFindQuery'].tables[0].rows, function (v) {
              return {
                text: v[0],
                value: v[1]
              };
            });
          }
        }]);

        return AwsAthenaDatasource;
      }());

      _export("AwsAthenaDatasource", AwsAthenaDatasource);
    }
  };
});
//# sourceMappingURL=datasource.js.map
