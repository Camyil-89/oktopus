export type AccessLogReportSpec = {
  version: 1;
  time: { from: string; to: string };
  filters: AccessLogReportFilters;
  widgets: AccessLogReportWidgetSpec[];
};

export type AccessLogReportFilters = {
  instance_id?: string;
  user?: string;
  source?: string;
  destination?: string;
  url?: string;
  search_only?: boolean;
  action?: 0 | 1;
  decision_rule_ref?: string;
  inspect_rule_id?: string;
  /** Подстрока по значению ctx:log (любое inspect-правило в extra). */
  inspect_log?: Record<string, string>;
  /** Только записи, где поле не пустая строка (колонка или inspect_log.*). */
  field_nonempty?: AccessLogReportGroupByField[];
};

export type AccessLogReportWidgetType = "timeseries" | "bar" | "table" | "stat";

export type AccessLogReportWidgetSpec = {
  id: string;
  type: AccessLogReportWidgetType;
  title?: string;
  query: AccessLogReportWidgetQuery;
};

export type AccessLogReportWidgetQuery = {
  metric?: "count" | "avg_decide_duration_us";
  group_by_time?: "10m" | "1h" | "1d";
  split_by?: "action" | "denied_by";
  group_by?: AccessLogReportGroupByField;
  group_by_cols?: AccessLogReportGroupByField[];
  /** Подмножество group_by_cols: поиск по подстроке (ILIKE) на бэкенде. */
  search_columns?: AccessLogReportGroupByField[];
  limit?: number;
  order?: "asc" | "desc";
  /** Дополнительно к filters.field_nonempty — для одного виджета. */
  field_nonempty?: AccessLogReportGroupByField[];
};

export type AccessLogReportColumnField =
  | "instance_id"
  | "destination_address"
  | "source_address"
  | "user"
  | "action"
  | "denied_by"
  | "decision_rule_ref"
  | "search_engine";

/** Колонка таблицы / bar: колонка журнала или inspect_log.<ключ> */
export type AccessLogReportGroupByField =
  | AccessLogReportColumnField
  | `inspect_log.${string}`;

export type AccessLogReportTimeseriesPoint = { t: string; value: number };

export type AccessLogReportTimeseriesSeries = {
  key: string;
  label: string;
  points: AccessLogReportTimeseriesPoint[];
};

export type AccessLogReportWidgetData = {
  series?: AccessLogReportTimeseriesSeries[];
  items?: { label: string; value: number }[];
  value?: number;
  label?: string;
  columns?: string[];
  rows?: string[][];
  values?: number[];
  total?: number;
  page?: number;
  page_size?: number;
};

export type AccessLogReportTableRequest = {
  version: 1;
  time: { from: string; to: string };
  filters: AccessLogReportFilters;
  widget: AccessLogReportWidgetSpec;
  page?: number;
  page_size?: number;
  search?: string;
};

export type AccessLogReportWidgetResult = {
  id: string;
  type: AccessLogReportWidgetType;
  title?: string;
  data: AccessLogReportWidgetData;
};

export type AccessLogReportResponse = {
  widgets: AccessLogReportWidgetResult[];
};
