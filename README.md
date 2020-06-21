## AWS Athena Datasource Plugin for Grafana

### Features:
 * Graph/Table view
 * Cache query result
 * Post query to AWS Athena (experimental)

### Setup
#### Install Plugin
Follow [Installing Plugins Manually](https://grafana.com/docs/plugins/installation/) steps, and install plugin from released zip file.

#### IAM Policy
Allow following API for IAM User/Role.

- athena:GetQueryResults
- athena:BatchGetNamedQuery
- athena:BatchGetQueryExecution
- athena:ListNamedQueries
- athena:ListQueryExecutions
- athena:ListWorkGroups

If use experimental query posting feature, allow following.
- athena:StartQueryExecution
- athena:GetWorkGroup

### Adding the DataSource to Grafana
See also CloudWatch DataSource Authentication doc to setup datasource.
https://grafana.com/docs/grafana/latest/features/datasources/cloudwatch/#authentication

| Name                       | Description                                                                                             |
| -------------------------- | ------------------------------------------------------------------------------------------------------- |
| _Name_                     | The data source name. This is how you refer to the data source in panels and queries.                   |
| _Default Region_           | Used in query editor to set region (can be changed on per query basis)                                  |
| _Auth Provider_            | Specify the provider to get credentials.                                                                |
| _Credentials_ profile name | Specify the name of the profile to use (if you use `~/.aws/credentials` file), leave blank for default. |
| _Assume Role Arn_          | Specify the ARN of the role to assume                                                                   |
| _Output Location_          | Specify the S3 Output Location for Athena query result. (experimental feature)                          |

### Query
#### Query Editor

| Name                       | Description                                                                                             |
| -------------------------- | ------------------------------------------------------------------------------------------------------- |
| _Region_                   | Specify the Region. (To use default region, specify "default")                                          |
| _Work Group_               | Specify the Work Group. (Work as filter for query execution id, or posting target workgroup)            |
| _Query Execution Id_       | Specify the comma separated Query Execution Ids to get result. (result format should be same)           |
| _Query String_             | Specify the AWS Athena Query. (experimental)                                                            |
| _Legend Format_            | Specify the Legend Format.                                                                              |
| _Max Rows_                 | Specify the Max Rows to get result. (default is 1000, -1 is unlimited)                                  |
| _Cache Duration_           | Specify the Cache Duration for caching query result. (cache key is query execution id and max rows)     |
| _Timestamp Column_         | Specify the Timestamp Column for time series.                                                           |
| _Value Column_             | Specify the Value Column for time series.                                                               |
| _Time Format_              | Specify the Time Format of Timestamp column. (default format is RFC3339)                                |

#### Query variable

| Name                                                                        | Description                                                                |
| --------------------------------------------------------------------------- | -------------------------------------------------------------------------- |
| *regions()*                                                                 | Returns a list of regions.                                                 |
| *workgroup_names(region)*                                                   | Returns a list of workgroup names.                                         |
| *named_query_names(region, work_group?)*                                    | Returns a list of named query names.                                       |
| *named_query_queries(region, pattern, work_group?)*                         | Returns a list of named query expressions which name match `pattern`.      |
| *query_execution_ids(region, limit, pattern, work_group?)*                  | Returns a list of query execution ids which query match `pattern`.         |
| *query_execution_ids_by_name(region, limit, named query name, work_group?)* | Returns a list of query execution ids which query match named query query. |

If a `work_group` is specified, result is filtered by that work_group.
The `query_execution_ids()` and `query_execution_ids_by_name()` results are always sorted by `CompletionDateTime` in descending order.

### Caution
This plugin experimentally support posting query.
To use the feature, set S3 output location in datasource settings.

And, limit data usage in workgroup settings.
https://docs.aws.amazon.com/athena/latest/ug/workgroups-setting-control-limits-cloudwatch.html

Every time when opening dashboard, Grafana post query without user acknowledgement, so it may cause too much AWS cost.
Please use carefully posting feature.
