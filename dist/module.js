'use strict';

System.register(['./datasource', './query_ctrl'], function (_export, _context) {
  "use strict";

  var AwsAthenaDatasource, AwsAthenaDatasourceQueryCtrl, AwsAthenaDatasourceConfigCtrl;

  function _classCallCheck(instance, Constructor) {
    if (!(instance instanceof Constructor)) {
      throw new TypeError("Cannot call a class as a function");
    }
  }

  return {
    setters: [function (_datasource) {
      AwsAthenaDatasource = _datasource.AwsAthenaDatasource;
    }, function (_query_ctrl) {
      AwsAthenaDatasourceQueryCtrl = _query_ctrl.AwsAthenaDatasourceQueryCtrl;
    }],
    execute: function () {
      _export('ConfigCtrl', AwsAthenaDatasourceConfigCtrl = function AwsAthenaDatasourceConfigCtrl() {
        _classCallCheck(this, AwsAthenaDatasourceConfigCtrl);
      });

      AwsAthenaDatasourceConfigCtrl.templateUrl = 'partials/config.html';

      _export('Datasource', AwsAthenaDatasource);

      _export('QueryCtrl', AwsAthenaDatasourceQueryCtrl);

      _export('ConfigCtrl', AwsAthenaDatasourceConfigCtrl);
    }
  };
});
//# sourceMappingURL=module.js.map
