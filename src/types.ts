import { DataSourceJsonData } from '@grafana/data';
import { DataQuery } from '@grafana/schema';

export interface MyQuery extends DataQuery {
  code?: string;
  resultCode?: string;
  timeNamesCode?: string;
}

export const DEFAULT_QUERY: Partial<MyQuery> = {
  code: "",
  resultCode: "result",
  timeNamesCode: "time_names"
};

export interface DataPoint {
  Time: number;
  Value: number;
}

export interface DataSourceResponse {
  datapoints: DataPoint[];
}

/**
 * These are options configured for each DataSource instance
 */
export interface MyDataSourceOptions extends DataSourceJsonData {
  apiBaseUrl?: string;
}

/**
 * Value that is used in the backend, but never sent over HTTP to the frontend
 */
export interface MySecureJsonData {
  apiToken?: string;
}
