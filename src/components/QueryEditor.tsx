import React, { ChangeEvent } from 'react';
import { InlineField, Input, InlineFieldRow, TextArea } from '@grafana/ui';
import { QueryEditorProps } from '@grafana/data';
import { DataSource } from '../datasource';
import { MyDataSourceOptions, MyQuery } from '../types';

type Props = QueryEditorProps<DataSource, MyQuery, MyDataSourceOptions>;

export function QueryEditor({ query, onChange }: Props) {
  const onResultCodeChange = (event: ChangeEvent<HTMLInputElement>) => {
    onChange({ ...query, resultCode: event.target.value });
  };
  const onPythonChange = (event: ChangeEvent<HTMLTextAreaElement>) => {
    onChange({ ...query, code: event.target.value });
  };
  const onTimeNamesCodeChange = (event: ChangeEvent<HTMLInputElement>) => {
    onChange({ ...query, timeNamesCode: event.target.value });
  };

  const { code, resultCode, timeNamesCode: timeNameListCode } = query;

  return (
    <>
    <InlineFieldRow>
        <InlineField label="Python Code" tooltip="Not used yet" grow>
          <TextArea
            id="query-editor-python-code"

            onChange={onPythonChange}
            value={code || ''}
            required
            // placeholder="Enter a query"
          />
        </InlineField>
      </InlineFieldRow>
      <InlineFieldRow>
        <InlineField label="Result Variable" labelWidth={24} tooltip="Not used yet">
          <Input
            id="query-editor-result-variable"
            onChange={onResultCodeChange}
            value={resultCode || ''}
            required
            placeholder="Enter a query"
          />
        </InlineField>
      </InlineFieldRow>
      <InlineFieldRow>
        <InlineField label="Time Name List Variable" labelWidth={24} tooltip="Not used yet">
          <Input
            id="query-editor-time-names-variable"
            onChange={onTimeNamesCodeChange}
            value={timeNameListCode || ''}
            required
            placeholder="Enter a query"
          />
        </InlineField>
      </InlineFieldRow>
    </>
  );
}
