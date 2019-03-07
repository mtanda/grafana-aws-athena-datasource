'use strict';

System.register(['./datasource', './query_ctrl', './config_ctrl'], function (_export, _context) {
  "use strict";

  var AwsAthenaDatasource, AwsAthenaDatasourceQueryCtrl, AwsAthenaDatasourceConfigCtrl;
  return {
    setters: [function (_datasource) {
      AwsAthenaDatasource = _datasource.AwsAthenaDatasource;
    }, function (_query_ctrl) {
      AwsAthenaDatasourceQueryCtrl = _query_ctrl.AwsAthenaDatasourceQueryCtrl;
    }, function (_config_ctrl) {
      AwsAthenaDatasourceConfigCtrl = _config_ctrl.AwsAthenaDatasourceConfigCtrl;
    }],
    execute: function () {
      _export('Datasource', AwsAthenaDatasource);

      _export('QueryCtrl', AwsAthenaDatasourceQueryCtrl);

      _export('ConfigCtrl', AwsAthenaDatasourceConfigCtrl);
    }
  };
});
//# sourceMappingURL=module.js.map
