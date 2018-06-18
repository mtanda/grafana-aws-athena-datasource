## AWS Athena Datasource Plugin for Grafana

### Note
This plugin only support viewing Athena result. Doesn't support post query to Athena.

Grafana doesn't support cache the result now, post query when dashboard is open will cause too much AWS cost.
So, please post query outside of this plugin.

### Setup
Allow following API for EC2 instance, and run Grafana on the EC2 instance.

- GetQueryResults
- BatchGetNamedQuery
- BatchGetQueryExecution
- ListNamedQueries
- ListQueryExecutions

### Templating

#### Query variable

Name | Description
---- | --------
*named_query_names(region)* | Returns a list of named query names.
*named_query_queries(region, pattern)* | Returns a list of named query expressions which match name `pattern`.
*query_execution_ids(region, limit, pattern)* | Returns a list of query execution ids which query match `pattern`.

The `query_execution_ids()` result is always sorted by `CompletionDateTime` in descending order.

#### Changelog

##### v1.0.0
- Initial release
