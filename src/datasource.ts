import { DataSourceInstanceSettings, MetricFindValue, SelectableValue } from '@grafana/data';
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

  async getWorkgroupNameOptions(region: string): Promise<Array<SelectableValue<string>>> {
    const workgroupNames = await this.getWorkgroupNames(region);
    return workgroupNames.map(name => ({ label: name, value: name } as SelectableValue<string>));
  }

  async getWorkgroupNames(region: string): Promise<string[]> {
    return (await this.getResource('workgroup_names', { region: region }))['workgroup_names'];
  }

  async getNamedQueryNames(region: string, workGroup: string): Promise<string[]> {
    return (await this.getResource('named_query_names', { region: region, workGroup: workGroup }))['named_query_names'];
  }

  async getNamedQueryQueries(region: string, pattern: string, workGroup: string): Promise<string[]> {
    return (await this.getResource('named_query_queries', { region: region, pattern: pattern, workGroup: workGroup }))[
      'named_query_queries'
    ];
  }

  async getQueryExecutionIdOptions(region: string, workgroup: string): Promise<Array<SelectableValue<string>>> {
    const to = new Date().toISOString(); // TODO
    const queryExecutionIds = await this.getQueryExecutionIds(region, -1, '.*', workgroup, to);
    return queryExecutionIds.map(name => ({ label: name, value: name } as SelectableValue<string>));
  }

  async getQueryExecutionIds(
    region: string,
    limit: number,
    pattern: string,
    workGroup: string,
    to: string
  ): Promise<string[]> {
    return (
      await this.getResource('query_execution_ids', {
        region: region,
        limit: limit,
        pattern: pattern,
        workGroup: workGroup,
        to: to,
      })
    )['query_execution_ids'];
  }

  async getQueryExecutionIdsByName(
    region: string,
    limit: number,
    pattern: string,
    workGroup: string,
    to: string
  ): Promise<string[]> {
    return (
      await this.getResource('query_execution_ids_by_name', {
        region: region,
        limit: limit,
        pattern: pattern,
        workGroup: workGroup,
        to: to,
      })
    )['query_execution_ids_by_name'];
  }

  async metricFindQuery?(query: any, options?: any): Promise<MetricFindValue[]> {
    const templateSrv = getTemplateSrv();

    const workgroupNamesQuery = query.match(/^workgroup_names\(([^\)]+?)\)/);
    if (workgroupNamesQuery) {
      const region = templateSrv.replace(workgroupNamesQuery[1]);
      const workgroupNames = await this.getWorkgroupNames(region);
      return workgroupNames.map(n => {
        return { text: n, value: n };
      });
    }

    const namedQueryNamesQuery = query.match(/^named_query_names\(([^\)]+?)(,\s?.+)?\)/);
    if (namedQueryNamesQuery) {
      const region = templateSrv.replace(namedQueryNamesQuery[1]);
      let workGroup = namedQueryNamesQuery[2];
      if (workGroup) {
        workGroup = workGroup.substr(1); //remove the comma
        workGroup = workGroup.trim();
      } else {
        workGroup = 'primary';
      }
      workGroup = templateSrv.replace(workGroup);
      const namedQueryNames = await this.getNamedQueryNames(region, workGroup);
      return namedQueryNames.map(n => {
        return { text: n, value: n };
      });
    }

    const namedQueryQueryQuery = query.match(/^named_query_queries\(([^,]+?),\s?([^,]+)(,\s?.+)?\)/);
    if (namedQueryQueryQuery) {
      const region = templateSrv.replace(namedQueryQueryQuery[1]);
      const pattern = templateSrv.replace(namedQueryQueryQuery[2], {}, 'regex');
      let workGroup = namedQueryQueryQuery[3];
      if (workGroup) {
        workGroup = workGroup.substr(1); //remove the comma
        workGroup = workGroup.trim();
      } else {
        workGroup = 'primary';
      }
      workGroup = templateSrv.replace(workGroup);
      const namedQueryQueries = await this.getNamedQueryQueries(region, pattern, workGroup);
      return namedQueryQueries.map(n => {
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
        workGroup = 'primary';
      }
      workGroup = templateSrv.replace(workGroup);
      const to = new Date().toISOString(); // TODO

      const queryExecutionIds = await this.getQueryExecutionIds(region, limit, pattern, workGroup, to);
      return queryExecutionIds.map(n => {
        return { text: n, value: n };
      });
    }

    const queryExecutionIdsByNameQuery = query.match(
      /^query_execution_ids_by_name\(([^,]+?),\s?([^,]+?),\s?([^,]+)(,\s?.+)?\)/
    );
    if (queryExecutionIdsByNameQuery) {
      const region = templateSrv.replace(queryExecutionIdsByNameQuery[1]);
      const limit = parseInt(templateSrv.replace(queryExecutionIdsByNameQuery[2]), 10);
      const pattern = templateSrv.replace(queryExecutionIdsByNameQuery[3], {}, 'regex');
      let workGroup = queryExecutionIdsByNameQuery[4];
      if (workGroup) {
        workGroup = workGroup.substr(1); //remove the comma
        workGroup = workGroup.trim();
      } else {
        workGroup = 'primary';
      }
      workGroup = templateSrv.replace(workGroup);
      const to = new Date().toISOString(); // TODO

      const queryExecutionIdsByName = await this.getQueryExecutionIdsByName(region, limit, pattern, workGroup, to);
      return queryExecutionIdsByName.map(n => {
        return { text: n, value: n };
      });
    }

    return [];
  }
}
