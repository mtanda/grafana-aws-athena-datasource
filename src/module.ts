import { DataSource } from './datasource';
import { DataSourcePlugin } from '@grafana/data';
import { ConfigEditor, QueryEditor } from './components';
import { AwsAthenaQuery, AwsAthenaOptions } from './types';

export const plugin = new DataSourcePlugin<DataSource, AwsAthenaQuery, AwsAthenaOptions>(DataSource)
  .setConfigEditor(ConfigEditor)
  .setQueryEditor(QueryEditor);
