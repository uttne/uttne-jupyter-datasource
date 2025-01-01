import React, { ChangeEvent } from 'react';
import { InlineField, Input, SecretInput } from '@grafana/ui';
import { DataSourcePluginOptionsEditorProps } from '@grafana/data';
import { MyDataSourceOptions, MySecureJsonData } from '../types';

interface Props extends DataSourcePluginOptionsEditorProps<MyDataSourceOptions, MySecureJsonData> {}

export function ConfigEditor(props: Props) {
  const { onOptionsChange, options } = props;
  const { jsonData, secureJsonFields, secureJsonData } = options;

  const onApiUrlChange = (event: ChangeEvent<HTMLInputElement>) => {
    onOptionsChange({
      ...options,
      jsonData: {
        ...jsonData,
        apiBaseUrl: event.target.value,
      },
    });
  };

  const onAPITokenChange = (event: ChangeEvent<HTMLInputElement>) => {
    onOptionsChange({
      ...options,
      secureJsonData: {
        apiToken: event.target.value,
      },
    });
  };

  const onResetAPIToken = () => {
    onOptionsChange({
      ...options,
      secureJsonFields: {
        ...options.secureJsonFields,
        apiToken: false,
      },
      secureJsonData: {
        ...options.secureJsonData,
        apiToken: '',
      },
    });
  };

  return (
    <>
      <InlineField label="API Base URL" labelWidth={20} interactive tooltip={'Json field returned to frontend'}>
        <Input
          id="config-editor-jupyter-api-base-url"
          onChange={onApiUrlChange}
          value={jsonData.apiBaseUrl}
          placeholder="Enter the URL, e.g. http://localhost:8888/api"
          width={60}
        />
      </InlineField>
      <InlineField label="API Token" labelWidth={20} interactive tooltip={'Secure json field (backend only)'}>
        <SecretInput
          required
          id="config-editor-jupyter-api-token"
          isConfigured={secureJsonFields.apiToken}
          value={secureJsonData?.apiToken}
          placeholder="Enter your API Token"
          width={60}
          onReset={onResetAPIToken}
          onChange={onAPITokenChange}
        />
      </InlineField>
    </>
  );
}
