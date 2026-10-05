import type {
  AccessLogReportWidgetSpec,
  AccessLogReportWidgetType,
} from "@/types/accessLogReport";
import type {
  ReportsDashboardDocument,
  ReportsDashboardTab,
} from "@/types/reportsDashboard";

export function emptyReportsDashboard(): ReportsDashboardDocument {
  return { version: 1, tabs: [] };
}

export function normalizeReportsDashboard(
  raw: unknown,
): ReportsDashboardDocument {
  if (!raw || typeof raw !== "object") {
    return emptyReportsDashboard();
  }
  const o = raw as Record<string, unknown>;
  const tabsRaw = Array.isArray(o.tabs) ? o.tabs : [];
  const tabs: ReportsDashboardTab[] = [];
  for (const item of tabsRaw) {
    if (!item || typeof item !== "object") {
      continue;
    }
    const t = item as Record<string, unknown>;
    const id = typeof t.id === "string" ? t.id.trim() : "";
    const title = typeof t.title === "string" ? t.title.trim() : "";
    if (!id || !title) {
      continue;
    }
    const filters =
      t.filters && typeof t.filters === "object"
        ? (t.filters as ReportsDashboardTab["filters"])
        : {};
    const widgets = Array.isArray(t.widgets)
      ? (t.widgets as AccessLogReportWidgetSpec[])
      : [];
    tabs.push({ id, title, filters, widgets });
  }
  return { version: 1, tabs };
}

export function newDashboardTabId(): string {
  return crypto.randomUUID();
}

export function newDashboardWidgetId(): string {
  return `w_${crypto.randomUUID().replace(/-/g, "").slice(0, 12)}`;
}

export function ensureDashboardWidgetIds(
  doc: ReportsDashboardDocument,
): ReportsDashboardDocument {
  return {
    ...doc,
    tabs: doc.tabs.map((tab) => ({
      ...tab,
      widgets: tab.widgets.map((w) =>
        w.id?.trim() ? w : { ...w, id: newDashboardWidgetId() },
      ),
    })),
  };
}

export function createDashboardTab(title: string): ReportsDashboardTab {
  return {
    id: newDashboardTabId(),
    title: title.trim(),
    filters: {},
    widgets: [],
  };
}

export function defaultWidgetSpec(
  type: AccessLogReportWidgetType,
  title: string,
): AccessLogReportWidgetSpec {
  const id = newDashboardWidgetId();
  switch (type) {
    case "timeseries":
      return {
        id,
        type,
        title,
        query: {
          metric: "count",
          group_by_time: "1h",
          split_by: "action",
        },
      };
    case "bar":
      return {
        id,
        type,
        title,
        query: {
          metric: "count",
          group_by: "destination_address",
          limit: 12,
          order: "desc",
        },
      };
    case "stat":
      return {
        id,
        type,
        title,
        query: { metric: "count" },
      };
    case "table":
      return {
        id,
        type,
        title,
        query: {
          metric: "count",
          group_by_cols: ["destination_address", "user"],
          limit: 20,
          order: "desc",
        },
      };
    default:
      return {
        id,
        type: "stat",
        title,
        query: { metric: "count" },
      };
  }
}
