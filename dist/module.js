define(["app/core/table_model","app/plugins/sdk","lodash"], function(__WEBPACK_EXTERNAL_MODULE_grafana_app_core_table_model__, __WEBPACK_EXTERNAL_MODULE_grafana_app_plugins_sdk__, __WEBPACK_EXTERNAL_MODULE_lodash__) { return /******/ (function(modules) { // webpackBootstrap
/******/ 	// The module cache
/******/ 	var installedModules = {};
/******/
/******/ 	// The require function
/******/ 	function __webpack_require__(moduleId) {
/******/
/******/ 		// Check if module is in cache
/******/ 		if(installedModules[moduleId]) {
/******/ 			return installedModules[moduleId].exports;
/******/ 		}
/******/ 		// Create a new module (and put it into the cache)
/******/ 		var module = installedModules[moduleId] = {
/******/ 			i: moduleId,
/******/ 			l: false,
/******/ 			exports: {}
/******/ 		};
/******/
/******/ 		// Execute the module function
/******/ 		modules[moduleId].call(module.exports, module, module.exports, __webpack_require__);
/******/
/******/ 		// Flag the module as loaded
/******/ 		module.l = true;
/******/
/******/ 		// Return the exports of the module
/******/ 		return module.exports;
/******/ 	}
/******/
/******/
/******/ 	// expose the modules object (__webpack_modules__)
/******/ 	__webpack_require__.m = modules;
/******/
/******/ 	// expose the module cache
/******/ 	__webpack_require__.c = installedModules;
/******/
/******/ 	// define getter function for harmony exports
/******/ 	__webpack_require__.d = function(exports, name, getter) {
/******/ 		if(!__webpack_require__.o(exports, name)) {
/******/ 			Object.defineProperty(exports, name, { enumerable: true, get: getter });
/******/ 		}
/******/ 	};
/******/
/******/ 	// define __esModule on exports
/******/ 	__webpack_require__.r = function(exports) {
/******/ 		if(typeof Symbol !== 'undefined' && Symbol.toStringTag) {
/******/ 			Object.defineProperty(exports, Symbol.toStringTag, { value: 'Module' });
/******/ 		}
/******/ 		Object.defineProperty(exports, '__esModule', { value: true });
/******/ 	};
/******/
/******/ 	// create a fake namespace object
/******/ 	// mode & 1: value is a module id, require it
/******/ 	// mode & 2: merge all properties of value into the ns
/******/ 	// mode & 4: return value when already ns object
/******/ 	// mode & 8|1: behave like require
/******/ 	__webpack_require__.t = function(value, mode) {
/******/ 		if(mode & 1) value = __webpack_require__(value);
/******/ 		if(mode & 8) return value;
/******/ 		if((mode & 4) && typeof value === 'object' && value && value.__esModule) return value;
/******/ 		var ns = Object.create(null);
/******/ 		__webpack_require__.r(ns);
/******/ 		Object.defineProperty(ns, 'default', { enumerable: true, value: value });
/******/ 		if(mode & 2 && typeof value != 'string') for(var key in value) __webpack_require__.d(ns, key, function(key) { return value[key]; }.bind(null, key));
/******/ 		return ns;
/******/ 	};
/******/
/******/ 	// getDefaultExport function for compatibility with non-harmony modules
/******/ 	__webpack_require__.n = function(module) {
/******/ 		var getter = module && module.__esModule ?
/******/ 			function getDefault() { return module['default']; } :
/******/ 			function getModuleExports() { return module; };
/******/ 		__webpack_require__.d(getter, 'a', getter);
/******/ 		return getter;
/******/ 	};
/******/
/******/ 	// Object.prototype.hasOwnProperty.call
/******/ 	__webpack_require__.o = function(object, property) { return Object.prototype.hasOwnProperty.call(object, property); };
/******/
/******/ 	// __webpack_public_path__
/******/ 	__webpack_require__.p = "";
/******/
/******/
/******/ 	// Load entry module and return exports
/******/ 	return __webpack_require__(__webpack_require__.s = "./module.ts");
/******/ })
/************************************************************************/
/******/ ({

/***/ "./config_ctrl.ts":
/*!************************!*\
  !*** ./config_ctrl.ts ***!
  \************************/
/*! no static exports found */
/***/ (function(module, exports, __webpack_require__) {

"use strict";


Object.defineProperty(exports, "__esModule", {
  value: true
});

var AwsAthenaDatasourceConfigCtrl =
/** @class */
function () {
  /** @ngInject */
  AwsAthenaDatasourceConfigCtrl.$inject = ["$scope", "datasourceSrv"];
  function AwsAthenaDatasourceConfigCtrl($scope, datasourceSrv) {
    this.current.jsonData.authType = this.current.jsonData.authType || 'credentials';
    this.accessKeyExist = this.current.secureJsonFields.accessKey;
    this.secretKeyExist = this.current.secureJsonFields.secretKey;
    this.datasourceSrv = datasourceSrv;
    this.authTypes = [{
      name: 'Access & secret key',
      value: 'keys'
    }, {
      name: 'Credentials file',
      value: 'credentials'
    }, {
      name: 'ARN',
      value: 'arn'
    }];
  }

  AwsAthenaDatasourceConfigCtrl.prototype.resetAccessKey = function () {
    this.accessKeyExist = false;
  };

  AwsAthenaDatasourceConfigCtrl.prototype.resetSecretKey = function () {
    this.secretKeyExist = false;
  };

  AwsAthenaDatasourceConfigCtrl.templateUrl = 'partials/config.html';
  return AwsAthenaDatasourceConfigCtrl;
}();

exports.AwsAthenaDatasourceConfigCtrl = AwsAthenaDatasourceConfigCtrl;

/***/ }),

/***/ "./datasource.ts":
/*!***********************!*\
  !*** ./datasource.ts ***!
  \***********************/
/*! no static exports found */
/***/ (function(module, exports, __webpack_require__) {

"use strict";


Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.AwsAthenaDatasource = undefined;

var _lodash = __webpack_require__(/*! lodash */ "lodash");

var _lodash2 = _interopRequireDefault(_lodash);

var _table_model = __webpack_require__(/*! grafana/app/core/table_model */ "grafana/app/core/table_model");

var _table_model2 = _interopRequireDefault(_table_model);

function _interopRequireDefault(obj) { return obj && obj.__esModule ? obj : { default: obj }; }

var AwsAthenaDatasource =
/** @class */
function () {
  function AwsAthenaDatasource(instanceSettings, $q, backendSrv, templateSrv, timeSrv) {
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

  AwsAthenaDatasource.prototype.query = function (options) {
    var query = this.buildQueryParameters(options);
    query.targets = query.targets.filter(function (t) {
      return !t.hide;
    });

    if (query.targets.length <= 0) {
      return this.q.when({
        data: []
      });
    }

    return this.doRequest({
      data: query
    });
  };

  AwsAthenaDatasource.prototype.testDatasource = function () {
    var _this = this;

    return this.doMetricQueryRequest('named_query_names', {
      region: this.defaultRegion
    }).then(function (res) {
      return _this.q.when({
        status: "success",
        message: "Data source is working",
        title: "Success"
      });
    }).catch(function (err) {
      return {
        status: "error",
        message: err.message,
        title: "Error"
      };
    });
  };

  AwsAthenaDatasource.prototype.doRequest = function (options) {
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

      _lodash2.default.forEach(result.data.results, function (r) {
        if (!_lodash2.default.isEmpty(r.series)) {
          _lodash2.default.forEach(r.series, function (s) {
            res.push({
              target: s.name,
              datapoints: s.points
            });
          });
        }

        if (!_lodash2.default.isEmpty(r.tables)) {
          _lodash2.default.forEach(r.tables, function (t) {
            var table = new _table_model2.default();
            table.columns = t.columns;
            table.rows = t.rows;
            res.push(table);
          });
        }
      });

      result.data = res;
      return result;
    });
  };

  AwsAthenaDatasource.prototype.buildQueryParameters = function (options) {
    var _this = this;

    var targets = _lodash2.default.map(options.targets, function (target) {
      return {
        refId: target.refId,
        hide: target.hide,
        datasourceId: _this.id,
        queryType: 'timeSeriesQuery',
        format: target.format || 'timeserie',
        region: _this.templateSrv.replace(target.region, options.scopedVars) || _this.defaultRegion,
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
  };

  AwsAthenaDatasource.prototype.metricFindQuery = function (query) {
    var region;
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

    var queryExecutionIdsQuery = query.match(/^query_execution_ids\(([^,]+?),\s?([^,]+?),\s?(.+)\)/);

    if (queryExecutionIdsQuery) {
      region = queryExecutionIdsQuery[1];
      var limit = queryExecutionIdsQuery[2];
      var pattern = queryExecutionIdsQuery[3];
      return this.doMetricQueryRequest('query_execution_ids', {
        region: this.templateSrv.replace(region),
        limit: parseInt(this.templateSrv.replace(limit), 10),
        pattern: this.templateSrv.replace(pattern, {}, 'regex')
      });
    }

    return this.q.when([]);
  };

  AwsAthenaDatasource.prototype.doMetricQueryRequest = function (subtype, parameters) {
    var _this = this;

    var range = this.timeSrv.timeRange();
    return this.backendSrv.datasourceRequest({
      url: '/api/tsdb/query',
      method: 'POST',
      data: {
        from: range.from.valueOf().toString(),
        to: range.to.valueOf().toString(),
        queries: [_lodash2.default.extend({
          refId: 'metricFindQuery',
          datasourceId: this.id,
          queryType: 'metricFindQuery',
          subtype: subtype
        }, parameters)]
      }
    }).then(function (r) {
      return _this.transformSuggestDataFromTable(r.data);
    });
  };

  AwsAthenaDatasource.prototype.transformSuggestDataFromTable = function (suggestData) {
    return _lodash2.default.map(suggestData.results['metricFindQuery'].tables[0].rows, function (v) {
      return {
        text: v[0],
        value: v[1]
      };
    });
  };

  return AwsAthenaDatasource;
}();

exports.AwsAthenaDatasource = AwsAthenaDatasource;

/***/ }),

/***/ "./module.ts":
/*!*******************!*\
  !*** ./module.ts ***!
  \*******************/
/*! no static exports found */
/***/ (function(module, exports, __webpack_require__) {

"use strict";


Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.ConfigCtrl = exports.QueryCtrl = exports.Datasource = undefined;

var _datasource = __webpack_require__(/*! ./datasource */ "./datasource.ts");

var _query_ctrl = __webpack_require__(/*! ./query_ctrl */ "./query_ctrl.ts");

var _config_ctrl = __webpack_require__(/*! ./config_ctrl */ "./config_ctrl.ts");

exports.Datasource = _datasource.AwsAthenaDatasource;
exports.QueryCtrl = _query_ctrl.AwsAthenaDatasourceQueryCtrl;
exports.ConfigCtrl = _config_ctrl.AwsAthenaDatasourceConfigCtrl;

/***/ }),

/***/ "./query_ctrl.ts":
/*!***********************!*\
  !*** ./query_ctrl.ts ***!
  \***********************/
/*! no static exports found */
/***/ (function(module, exports, __webpack_require__) {

"use strict";


Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.AwsAthenaDatasourceQueryCtrl = undefined;

var _sdk = __webpack_require__(/*! grafana/app/plugins/sdk */ "grafana/app/plugins/sdk");

var __extends = undefined && undefined.__extends || function () {
  var extendStatics = Object.setPrototypeOf || {
    __proto__: []
  } instanceof Array && function (d, b) {
    d.__proto__ = b;
  } || function (d, b) {
    for (var p in b) {
      if (b.hasOwnProperty(p)) d[p] = b[p];
    }
  };

  return function (d, b) {
    extendStatics(d, b);

    function __() {
      this.constructor = d;
    }

    d.prototype = b === null ? Object.create(b) : (__.prototype = b.prototype, new __());
  };
}();

var AwsAthenaDatasourceQueryCtrl =
/** @class */
function (_super) {
  __extends(AwsAthenaDatasourceQueryCtrl, _super);

  function AwsAthenaDatasourceQueryCtrl($scope, $injector) {
    var _this = _super.call(this, $scope, $injector) || this;

    _this.scope = $scope;
    _this.target.format = _this.target.format || _this.target.type || 'timeserie';
    _this.target.region = _this.target.region || '';
    _this.target.timestampColumn = _this.target.timestampColumn || '';
    _this.target.valueColumn = _this.target.valueColumn || '';
    _this.target.legendFormat = _this.target.legendFormat || '';
    _this.target.queryExecutionId = _this.target.queryExecutionId || '';
    return _this;
  }

  AwsAthenaDatasourceQueryCtrl.prototype.onChangeInternal = function () {
    this.panelCtrl.refresh();
  };

  AwsAthenaDatasourceQueryCtrl.templateUrl = 'partials/query.editor.html';
  return AwsAthenaDatasourceQueryCtrl;
}(_sdk.QueryCtrl);

exports.AwsAthenaDatasourceQueryCtrl = AwsAthenaDatasourceQueryCtrl;

/***/ }),

/***/ "grafana/app/core/table_model":
/*!***************************************!*\
  !*** external "app/core/table_model" ***!
  \***************************************/
/*! no static exports found */
/***/ (function(module, exports) {

module.exports = __WEBPACK_EXTERNAL_MODULE_grafana_app_core_table_model__;

/***/ }),

/***/ "grafana/app/plugins/sdk":
/*!**********************************!*\
  !*** external "app/plugins/sdk" ***!
  \**********************************/
/*! no static exports found */
/***/ (function(module, exports) {

module.exports = __WEBPACK_EXTERNAL_MODULE_grafana_app_plugins_sdk__;

/***/ }),

/***/ "lodash":
/*!*************************!*\
  !*** external "lodash" ***!
  \*************************/
/*! no static exports found */
/***/ (function(module, exports) {

module.exports = __WEBPACK_EXTERNAL_MODULE_lodash__;

/***/ })

/******/ })});;
//# sourceMappingURL=module.js.map