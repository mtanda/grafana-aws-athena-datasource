import AwsAthenaDatasource from './datasource';
import { AwsAthenaDatasourceQueryCtrl } from './query_ctrl';
import { AwsAthenaDatasourceConfigCtrl } from './config_ctrl';
import { DataSourcePlugin } from '@grafana/data';
import { AwsAthenaQuery, AwsAthenaOptions } from './types';

export { AwsAthenaDatasource as Datasource, AwsAthenaDatasourceQueryCtrl as QueryCtrl, AwsAthenaDatasourceConfigCtrl as ConfigCtrl };
export const plugin = new DataSourcePlugin<AwsAthenaDatasource, AwsAthenaQuery, AwsAthenaOptions>(AwsAthenaDatasource)
  .setConfigCtrl(AwsAthenaDatasourceConfigCtrl)
  .setQueryCtrl(AwsAthenaDatasourceQueryCtrl);
