import type {
  AccessLogReportFilters,
  AccessLogReportWidgetSpec,
} from "@/types/accessLogReport";

export type ReportsDashboardDocument = {
  version: 1;
  tabs: ReportsDashboardTab[];
};

export type ReportsDashboardTab = {
  id: string;
  title: string;
  filters: AccessLogReportFilters;
  widgets: AccessLogReportWidgetSpec[];
};
