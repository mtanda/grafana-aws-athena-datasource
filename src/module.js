import { AwsAthenaDatasource } from './datasource';
import { AwsAthenaDatasourceQueryCtrl } from './query_ctrl';

class AwsAthenaDatasourceConfigCtrl { }
AwsAthenaDatasourceConfigCtrl.templateUrl = 'partials/config.html';

export {
  AwsAthenaDatasource as Datasource,
  AwsAthenaDatasourceQueryCtrl as QueryCtrl,
  AwsAthenaDatasourceConfigCtrl as ConfigCtrl
};
