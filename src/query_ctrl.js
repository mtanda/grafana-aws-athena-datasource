import { QueryCtrl } from 'app/plugins/sdk';

export class AwsAthenaDatasourceQueryCtrl extends QueryCtrl {
  constructor($scope, $injector) {
    super($scope, $injector);

    this.scope = $scope;
    this.target.type = this.target.type || 'timeserie';
    this.target.region = this.target.region || '';
    this.target.queryExecutionId = this.target.queryExecutionId || '';
  }

  onChangeInternal() {
    this.panelCtrl.refresh();
  }
}

AwsAthenaDatasourceQueryCtrl.templateUrl = 'partials/query.editor.html';
