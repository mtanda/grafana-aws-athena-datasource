import React, { PureComponent } from 'react';
import { QueryEditorProps, SelectableValue } from '@grafana/data';
import { Select, InlineFormLabel } from '@grafana/ui';
import { DataSource } from '../datasource';
import { AwsAthenaQuery, AwsAthenaOptions } from '../types';

type Props = QueryEditorProps<DataSource, AwsAthenaQuery, AwsAthenaOptions>;

const FORMAT_OPTIONS: Array<SelectableValue<string>> = [
  { label: 'Time series', value: 'timeserie' },
  { label: 'Table', value: 'table' },
];

interface State {
  formatOption: SelectableValue<string>;
  region: string;
  queryExecutionId: string;
  timestampColumn: string;
  valueColumn: string;
  legendFormat: string;
  timeFormat: string;
}

export class QueryEditor extends PureComponent<Props, State> {
  query: AwsAthenaQuery;

  constructor(props: Props) {
    super(props);
    const defaultQuery: Partial<AwsAthenaQuery> = {
      format: 'timeserie',
      region: '',
      queryExecutionId: '',
      timestampColumn: '',
      valueColumn: '',
      legendFormat: '',
      timeFormat: '',
    };
    const query = Object.assign({}, defaultQuery, props.query);
    this.query = query;
    this.state = {
      formatOption: FORMAT_OPTIONS.find(option => option.value === query.format) || FORMAT_OPTIONS[0],
      region: query.region,
      queryExecutionId: query.queryExecutionId,
      timestampColumn: query.timestampColumn,
      valueColumn: query.valueColumn,
      legendFormat: query.legendFormat,
      timeFormat: query.timeFormat,
    };
  }

  onFormatChange = (option: SelectableValue<string>) => {
    this.query.format = option.value || 'timeserie';
    this.setState({ formatOption: option }, this.onRunQuery);
  };

  onRegionChange = (e: React.SyntheticEvent<HTMLInputElement>) => {
    const region = e.currentTarget.value;
    this.query.region = region;
    this.setState({ region });
  };

  onQueryExecutionIdChange = (e: React.SyntheticEvent<HTMLInputElement>) => {
    const queryExecutionId = e.currentTarget.value;
    this.query.queryExecutionId = queryExecutionId;
    this.setState({ queryExecutionId });
  };

  onTimestampColumnChange = (e: React.SyntheticEvent<HTMLInputElement>) => {
    const timestampColumn = e.currentTarget.value;
    this.query.timestampColumn = timestampColumn;
    this.setState({ timestampColumn });
  };

  onValueColumnChange = (e: React.SyntheticEvent<HTMLInputElement>) => {
    const valueColumn = e.currentTarget.value;
    this.query.valueColumn = valueColumn;
    this.setState({ valueColumn });
  };

  onLegendFormatChange = (e: React.SyntheticEvent<HTMLInputElement>) => {
    const legendFormat = e.currentTarget.value;
    this.query.legendFormat = legendFormat;
    this.setState({ legendFormat });
  };

  onTimeFormatChange = (e: React.SyntheticEvent<HTMLInputElement>) => {
    const timeFormat = e.currentTarget.value;
    this.query.timeFormat = timeFormat;
    this.setState({ timeFormat });
  };

  onRunQuery = () => {
    const { query } = this;
    this.props.onChange(query);
    this.props.onRunQuery();
  };

  render() {
    const {
      formatOption,
      region,
      queryExecutionId,
      timestampColumn,
      valueColumn,
      legendFormat,
      timeFormat,
    } = this.state;
    return (
      <>
        <div className="gf-form-inline">
          <div className="gf-form">
            <InlineFormLabel width={8}>Format</InlineFormLabel>
            <Select
              width={16}
              isSearchable={false}
              options={FORMAT_OPTIONS}
              onChange={this.onFormatChange}
              value={formatOption}
            />
          </div>

          <div className="gf-form">
            <InlineFormLabel width={8}>Query Execution Id</InlineFormLabel>
            <input
              type="text"
              className="gf-form-input"
              placeholder="query execution id"
              value={queryExecutionId}
              onChange={this.onQueryExecutionIdChange}
              onBlur={this.onRunQuery}
            />
          </div>
        </div>

        <div className="gf-form-inline">
          <div className="gf-form">
            <InlineFormLabel width={8}>Region</InlineFormLabel>
            <input
              type="text"
              className="gf-form-input"
              placeholder="us-east-1"
              value={region}
              onChange={this.onRegionChange}
              onBlur={this.onRunQuery}
            />
          </div>

          <div className="gf-form">
            <InlineFormLabel width={8}>Timestamp Column</InlineFormLabel>
            <input
              type="text"
              className="gf-form-input"
              placeholder=""
              value={timestampColumn}
              onChange={this.onTimestampColumnChange}
              onBlur={this.onRunQuery}
            />
          </div>

          <div className="gf-form">
            <InlineFormLabel width={8}>Value Column</InlineFormLabel>
            <input
              type="text"
              className="gf-form-input"
              placeholder=""
              value={valueColumn}
              onChange={this.onValueColumnChange}
              onBlur={this.onRunQuery}
            />
          </div>

          <div className="gf-form">
            <InlineFormLabel width={8}>Legend Format</InlineFormLabel>
            <input
              type="text"
              className="gf-form-input"
              placeholder=""
              value={legendFormat}
              onChange={this.onLegendFormatChange}
              onBlur={this.onRunQuery}
            />
          </div>

          <div className="gf-form">
            <InlineFormLabel width={8}>Time Format</InlineFormLabel>
            <input
              type="text"
              className="gf-form-input"
              placeholder=""
              value={timeFormat}
              onChange={this.onTimestampColumnChange}
              onBlur={this.onRunQuery}
            />
          </div>
        </div>
      </>
    );
  }
}
