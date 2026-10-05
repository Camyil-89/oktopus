import type { AccessLogReportWidgetSpec } from "@/types/accessLogReport";
import { parseAccessLogReportSpec } from "@/utils/accessLogReportValidate";
import { newDashboardWidgetId } from "@/utils/reportsDashboard";

function widgetDraftSpec(widget: unknown) {
  return JSON.stringify({
    version: 1,
    time: {
      from: "2026-01-01T00:00:00Z",
      to: "2026-01-02T00:00:00Z",
    },
    filters: {},
    widgets: [widget],
  });
}

function ensureWidgetId(w: AccessLogReportWidgetSpec): AccessLogReportWidgetSpec {
  const id = w.id?.trim() ? w.id : newDashboardWidgetId();
  return { ...w, id };
}

/** Парсит один виджет, массив виджетов или spec с полем widgets. */
export function parseReportWidgetsJson(text: string): AccessLogReportWidgetSpec[] {
  const trimmed = text.trim();
  if (!trimmed) {
    throw new Error("empty");
  }
  const parsed = JSON.parse(trimmed) as unknown;

  if (Array.isArray(parsed)) {
    const out: AccessLogReportWidgetSpec[] = [];
    for (const item of parsed) {
      const spec = parseAccessLogReportSpec(widgetDraftSpec(item));
      out.push(...spec.widgets.map(ensureWidgetId));
    }
    return out;
  }

  if (
    parsed &&
    typeof parsed === "object" &&
    "widgets" in parsed &&
    Array.isArray((parsed as { widgets: unknown }).widgets)
  ) {
    const spec = parseAccessLogReportSpec(trimmed);
    return spec.widgets.map(ensureWidgetId);
  }

  if (parsed && typeof parsed === "object" && "type" in parsed) {
    const spec = parseAccessLogReportSpec(widgetDraftSpec(parsed));
    return spec.widgets.map(ensureWidgetId);
  }

  throw new Error("invalid");
}
