import { DataQuery, DataSourceJsonData } from '@grafana/data';

export interface AwsAthenaOptions extends DataSourceJsonData {
  defaultRegion: string;
}

export interface AwsAthenaQuery extends DataQuery {
  refId: string;
  format: 'timeseries' | 'table';
  region: string;
  queryExecutionId: string;
  timestampColumn: string;
  valueColumn: string;
  legendFormat: string;
  timeFormat: string;
}
