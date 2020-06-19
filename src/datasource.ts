import { DataSourceInstanceSettings, MetricFindValue } from '@grafana/data';
import { DataSourceWithBackend, getTemplateSrv } from '@grafana/runtime';
import { AwsAthenaQuery, AwsAthenaOptions } from './types';

export class DataSource extends DataSourceWithBackend<AwsAthenaQuery, AwsAthenaOptions> {
  defaultRegion: string;

  constructor(instanceSettings: DataSourceInstanceSettings<AwsAthenaOptions>) {
    super(instanceSettings);
    this.defaultRegion = instanceSettings.jsonData.defaultRegion || 'us-east-1';
  }

  applyTemplateVariables(query: AwsAthenaQuery) {
    // TODO: pass scopedVars to templateSrv.replace()
    const templateSrv = getTemplateSrv();
    query.region = templateSrv.replace(query.region) || this.defaultRegion;
    query.maxRows = query.maxRows || '1000';
    query.queryExecutionId = templateSrv.replace(query.queryExecutionId);
    query.inputs = query.queryExecutionId.split(/,/).map(id => {
      return {
        queryExecutionId: id,
      };
    });
    return query;
  }

  async getNamedQueryNames(region: string): Promise<string[]> {
    return this.getResource('named_query_names', { region: region });
  }

  async getNamedQueryQueries(region: string, pattern: string): Promise<string[]> {
    return this.getResource('named_query_queries', { region: region, pattern: pattern });
  }

  async getQueryExecutionIds(
    region: string,
    limit: number,
    pattern: string,
    workGroup: string,
    to: string
  ): Promise<string[]> {
    return this.getResource('query_execution_ids', {
      region: region,
      limit: limit,
      pattern: pattern,
      workGroup: workGroup,
      to: to,
    });
  }

  async metricFindQuery?(query: any, options?: any): Promise<MetricFindValue[]> {
    const templateSrv = getTemplateSrv();

    const namedQueryNamesQuery = query.match(/^named_query_names\(([^\)]+?)\)/);
    if (namedQueryNamesQuery) {
      const region = templateSrv.replace(namedQueryNamesQuery[1]);
      const namedQueryNames = await this.getNamedQueryNames(templateSrv.replace(region));
      return namedQueryNames['named_query_names'].map(n => {
        return { text: n, value: n };
      });
    }

    const namedQueryQueryQuery = query.match(/^named_query_queries\(([^,]+?),\s?(.+)\)/);
    if (namedQueryQueryQuery) {
      const region = templateSrv.replace(namedQueryQueryQuery[1]);
      const pattern = templateSrv.replace(namedQueryQueryQuery[2], {}, 'regex');
      const namedQueryQueries = await this.getNamedQueryQueries(region, pattern);
      return namedQueryQueries['named_query_queries'].map(n => {
        return { text: n, value: n };
      });
    }

    const queryExecutionIdsQuery = query.match(/^query_execution_ids\(([^,]+?),\s?([^,]+?),\s?([^,]+)(,\s?.+)?\)/);
    if (queryExecutionIdsQuery) {
      const region = templateSrv.replace(queryExecutionIdsQuery[1]);
      const limit = parseInt(templateSrv.replace(queryExecutionIdsQuery[2]), 10);
      const pattern = templateSrv.replace(queryExecutionIdsQuery[3], {}, 'regex');
      let workGroup = queryExecutionIdsQuery[4];
      if (workGroup) {
        workGroup = workGroup.substr(1); //remove the comma
        workGroup = workGroup.trim();
      } else {
        workGroup = null;
      }
      workGroup = templateSrv.replace(workGroup);
      const to = new Date().toISOString(); // TODO

      const queryExecutionIds = await this.getQueryExecutionIds(region, limit, pattern, workGroup, to);
      return queryExecutionIds['query_execution_ids'].map(n => {
        return { text: n, value: n };
      });
    }

    return [];
  }
}
