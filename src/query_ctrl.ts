import { QueryCtrl } from 'grafana/app/plugins/sdk';

export class AwsAthenaDatasourceQueryCtrl extends QueryCtrl {
  scope: any;
  target: any;
  panelCtrl: any;
  static templateUrl = 'partials/query.editor.html';

  constructor($scope, $injector) {
    super($scope, $injector);

    this.scope = $scope;
    this.target.type = this.target.type || 'timeserie';
    this.target.region = this.target.region || '';
    this.target.timestampColumn = this.target.timestampColumn || '';
    this.target.valueColumn = this.target.valueColumn || '';
    this.target.legendFormat = this.target.legendFormat || '';
    this.target.queryExecutionId = this.target.queryExecutionId || '';
  }

  onChangeInternal() {
    this.panelCtrl.refresh();
  }
}

