## AWS Athena Datasource Plugin for Grafana

### Note
This plugin only support viewing Athena result. Doesn't support post query to Athena.

Grafana doesn't support cache the result now, post query when dashboard is open will cause too much AWS cost.
So, please post query outside of this plugin.

### Setup
Follow [Installing Plugins Manually](https://grafana.com/docs/plugins/installation/) steps, and install plugin from released zip file.

Allow following API for EC2 instance, and run Grafana on the EC2 instance.

- athena:GetQueryResults
- athena:BatchGetNamedQuery
- athena:BatchGetQueryExecution
- athena:ListNamedQueries
- athena:ListQueryExecutions

### Templating

#### Query variable

Name | Description
---- | --------
*named_query_names(region)* | Returns a list of named query names.
*named_query_queries(region, pattern)* | Returns a list of named query expressions which name match `pattern`.
*query_execution_ids(region, limit, pattern, work_group?)* | Returns a list of query execution ids which query match `pattern`. If a `work_group` is specified, only execution_ids within that work_group will be returned.

The `query_execution_ids()` result is always sorted by `CompletionDateTime` in descending order.

#### Changelog

##### v1.1.0
- Added support for [athena work groups](https://docs.aws.amazon.com/athena/latest/ug/user-created-workgroups.html) to work around the long api call for execution ids
- Updated the Makefile to use webpack, as well as package.json to use the latest version of babel
- Properly handle null values in the results of the query by ignoring them

##### v1.0.0
- Initial release
