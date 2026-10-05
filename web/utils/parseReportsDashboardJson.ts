import type { AccessLogReportWidgetSpec } from "@/types/accessLogReport";
import type {
  ReportsDashboardDocument,
  ReportsDashboardTab,
} from "@/types/reportsDashboard";
import { parseAccessLogReportSpec } from "@/utils/accessLogReportValidate";
import {
  ensureDashboardWidgetIds,
  newDashboardWidgetId,
} from "@/utils/reportsDashboard";

const DRAFT_TIME = {
  from: "2026-01-01T00:00:00Z",
  to: "2026-01-02T00:00:00Z",
};

function parseTabWidgets(
  widgetsRaw: unknown,
  filters: ReportsDashboardTab["filters"],
): AccessLogReportWidgetSpec[] {
  if (!Array.isArray(widgetsRaw)) {
    throw new Error("invalid widgets");
  }
  const spec = parseAccessLogReportSpec(
    JSON.stringify({
      version: 1,
      time: DRAFT_TIME,
      filters,
      widgets: widgetsRaw,
    }),
  );
  return spec.widgets.map((w) =>
    w.id?.trim() ? w : { ...w, id: newDashboardWidgetId() },
  );
}

export function parseReportsDashboardTabJson(text: string): ReportsDashboardTab {
  const trimmed = text.trim();
  if (!trimmed) {
    throw new Error("empty");
  }
  let parsed: unknown;
  try {
    parsed = JSON.parse(trimmed);
  } catch {
    throw new Error("invalid json");
  }
  const tab = parseTab(parsed);
  return {
    ...tab,
    widgets: tab.widgets.map((w) =>
      w.id?.trim() ? w : { ...w, id: newDashboardWidgetId() },
    ),
  };
}

function parseTab(item: unknown): ReportsDashboardTab {
  if (!item || typeof item !== "object") {
    throw new Error("invalid tab");
  }
  const t = item as Record<string, unknown>;
  const id = typeof t.id === "string" ? t.id.trim() : "";
  const title = typeof t.title === "string" ? t.title.trim() : "";
  if (!id || !title) {
    throw new Error("invalid tab id or title");
  }
  const filters =
    t.filters && typeof t.filters === "object"
      ? (t.filters as ReportsDashboardTab["filters"])
      : {};
  const widgets = parseTabWidgets(t.widgets, filters);
  return { id, title, filters, widgets };
}

/** Парсит и валидирует JSON документа дашборда (version 1, tabs). */
export function parseReportsDashboardJson(text: string): ReportsDashboardDocument {
  const trimmed = text.trim();
  if (!trimmed) {
    throw new Error("empty");
  }
  let parsed: unknown;
  try {
    parsed = JSON.parse(trimmed);
  } catch {
    throw new Error("invalid json");
  }
  if (!parsed || typeof parsed !== "object") {
    throw new Error("invalid");
  }
  const o = parsed as Record<string, unknown>;
  if (o.version !== 1) {
    throw new Error("invalid version");
  }
  if (!Array.isArray(o.tabs)) {
    throw new Error("invalid tabs");
  }
  const tabs = o.tabs.map((item) => parseTab(item));
  return ensureDashboardWidgetIds({ version: 1, tabs });
}
