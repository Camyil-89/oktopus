import { ACCESS_LOG_REPORT_COLUMN_FIELDS } from "@/assets/accessLog/reportSpecReference";
import type {
  AccessLogReportGroupByField,
  AccessLogReportSpec,
  AccessLogReportWidgetSpec,
  AccessLogReportWidgetType,
} from "@/types/accessLogReport";
import {
  isInspectLogFieldKey,
  parseInspectLogGroupByField,
} from "@/utils/accessLogInspectLog";

const GROUP_BY_SET = new Set<string>(ACCESS_LOG_REPORT_COLUMN_FIELDS);

const WIDGET_TYPES = new Set<AccessLogReportWidgetType>([
  "timeseries",
  "bar",
  "table",
  "stat",
]);

export function parseAccessLogReportSpec(raw: string): AccessLogReportSpec {
  let parsed: unknown;
  try {
    parsed = JSON.parse(raw);
  } catch {
    throw new Error("Некорректный JSON");
  }
  if (!parsed || typeof parsed !== "object") {
    throw new Error("Ожидается объект");
  }
  const o = parsed as Record<string, unknown>;
  if (o.version !== 1) {
    throw new Error("version должна быть 1");
  }
  const time = o.time;
  if (!time || typeof time !== "object") {
    throw new Error("time обязателен");
  }
  const t = time as Record<string, unknown>;
  if (typeof t.from !== "string" || typeof t.to !== "string") {
    throw new Error("time.from и time.to обязательны");
  }
  const widgetsRaw = o.widgets;
  if (!Array.isArray(widgetsRaw) || widgetsRaw.length === 0) {
    throw new Error("widgets не пустой массив");
  }
  if (widgetsRaw.length > 12) {
    throw new Error("не более 12 виджетов");
  }
  const widgets: AccessLogReportWidgetSpec[] = [];
  const ids = new Set<string>();
  for (const item of widgetsRaw) {
    widgets.push(parseWidget(item, ids));
  }
  const filters = parseFilters(o.filters);
  return {
    version: 1,
    time: { from: t.from.trim(), to: t.to.trim() },
    filters,
    widgets,
  };
}

function parseFilters(raw: unknown): AccessLogReportSpec["filters"] {
  if (!raw || typeof raw !== "object") {
    return {};
  }
  const f = raw as Record<string, unknown>;
  const out: AccessLogReportSpec["filters"] = {};
  if (typeof f.instance_id === "string" && f.instance_id.trim()) {
    out.instance_id = f.instance_id.trim();
  }
  if (typeof f.user === "string" && f.user) out.user = f.user;
  if (typeof f.source === "string" && f.source) out.source = f.source;
  if (typeof f.destination === "string" && f.destination)
    out.destination = f.destination;
  if (typeof f.url === "string" && f.url) out.url = f.url;
  if (f.search_only === true) out.search_only = true;
  if (f.action === 0 || f.action === 1) out.action = f.action;
  if (typeof f.decision_rule_ref === "string" && f.decision_rule_ref)
    out.decision_rule_ref = f.decision_rule_ref;
  if (typeof f.inspect_rule_id === "string" && f.inspect_rule_id)
    out.inspect_rule_id = f.inspect_rule_id;
  if (f.inspect_log && typeof f.inspect_log === "object") {
    const il = f.inspect_log as Record<string, unknown>;
    const inspectLog: Record<string, string> = {};
    for (const [k, v] of Object.entries(il)) {
      if (!isInspectLogFieldKey(k)) {
        throw new Error(`filters.inspect_log: недопустимый ключ ${k}`);
      }
      if (typeof v !== "string" || !v.trim()) {
        continue;
      }
      inspectLog[k] = v.trim();
    }
    if (Object.keys(inspectLog).length > 0) {
      out.inspect_log = inspectLog;
    }
  }
  const fieldNonempty = parseFieldNonemptyList(f.field_nonempty, "filters.field_nonempty");
  if (fieldNonempty.length > 0) {
    out.field_nonempty = fieldNonempty;
  }
  return out;
}

function parseWidget(
  raw: unknown,
  ids: Set<string>,
): AccessLogReportWidgetSpec {
  if (!raw || typeof raw !== "object") {
    throw new Error("виджет: ожидается объект");
  }
  const w = raw as Record<string, unknown>;
  if (typeof w.id !== "string" || !w.id.trim()) {
    throw new Error("виджет: id обязателен");
  }
  const id = w.id.trim();
  if (ids.has(id)) {
    throw new Error(`виджет: дублирующийся id ${id}`);
  }
  ids.add(id);
  if (typeof w.type !== "string" || !WIDGET_TYPES.has(w.type as AccessLogReportWidgetType)) {
    throw new Error(`виджет ${id}: неизвестный type`);
  }
  const type = w.type as AccessLogReportWidgetType;
  const queryRaw = w.query;
  if (!queryRaw || typeof queryRaw !== "object") {
    throw new Error(`виджет ${id}: query обязателен`);
  }
  const q = queryRaw as Record<string, unknown>;
  const query = parseQuery(type, id, q);
  const title = typeof w.title === "string" ? w.title : undefined;
  return { id, type, title, query };
}

function parseQuery(
  type: AccessLogReportWidgetType,
  id: string,
  q: Record<string, unknown>,
): AccessLogReportWidgetSpec["query"] {
  const metric =
    q.metric === "avg_decide_duration_us" ? "avg_decide_duration_us" : "count";
  const out: AccessLogReportWidgetSpec["query"] = { metric };
  if (typeof q.limit === "number" && q.limit > 0) {
    out.limit = Math.min(100, Math.floor(q.limit));
  }
  if (q.order === "asc" || q.order === "desc") {
    out.order = q.order;
  }
  const fieldNonempty = parseFieldNonemptyList(
    q.field_nonempty,
    `виджет ${id}: field_nonempty`,
  );
  if (fieldNonempty.length > 0) {
    out.field_nonempty = fieldNonempty;
  }
  switch (type) {
    case "timeseries": {
      const g = q.group_by_time;
      if (
        g !== "10m" &&
        g !== "1h" &&
        g !== "1d" &&
        g !== "hour_of_day" &&
        g !== "day_of_week" &&
        g !== "month_of_year"
      ) {
        throw new Error(
          `виджет ${id}: group_by_time 10m|1h|1d|hour_of_day|day_of_week|month_of_year`,
        );
      }
      out.group_by_time = g;
      if (q.split_by === "action" || q.split_by === "denied_by") {
        out.split_by = q.split_by;
      }
      break;
    }
    case "bar": {
      const gb = parseGroupBy(q.group_by, id);
      out.group_by = gb;
      break;
    }
    case "table": {
      let tableCols: AccessLogReportGroupByField[] | undefined;
      if (Array.isArray(q.group_by_cols) && q.group_by_cols.length > 0) {
        tableCols = q.group_by_cols.map((c, i) =>
          parseGroupBy(c, `${id}.group_by_cols[${i}]`),
        );
        out.group_by_cols = tableCols;
      } else {
        out.group_by = parseGroupBy(q.group_by, id);
        tableCols = [out.group_by];
      }
      if (Array.isArray(q.search_columns) && q.search_columns.length > 0) {
        const colSet = new Set(tableCols);
        const searchCols: AccessLogReportGroupByField[] = [];
        for (let i = 0; i < q.search_columns.length; i++) {
          const sc = parseGroupBy(
            q.search_columns[i],
            `${id}.search_columns[${i}]`,
          );
          if (!colSet.has(sc)) {
            throw new Error(
              `виджет ${id}: search_columns[${i}] должно быть в group_by_cols`,
            );
          }
          searchCols.push(sc);
        }
        out.search_columns = searchCols;
      }
      break;
    }
    case "stat":
      break;
  }
  return out;
}

function parseGroupBy(raw: unknown, ctx: string): AccessLogReportGroupByField {
  if (typeof raw !== "string" || !raw.trim()) {
    throw new Error(`виджет ${ctx}: недопустимый group_by`);
  }
  const name = raw.trim();
  if (GROUP_BY_SET.has(name)) {
    return name as AccessLogReportGroupByField;
  }
  if (parseInspectLogGroupByField(name)) {
    return name as AccessLogReportGroupByField;
  }
  throw new Error(`виджет ${ctx}: недопустимый group_by`);
}

function parseFieldNonemptyList(
  raw: unknown,
  ctx: string,
): AccessLogReportGroupByField[] {
  if (!Array.isArray(raw) || raw.length === 0) {
    return [];
  }
  const out: AccessLogReportGroupByField[] = [];
  const seen = new Set<string>();
  for (let i = 0; i < raw.length; i++) {
    const name = parseGroupBy(raw[i], `${ctx}[${i}]`);
    if (seen.has(name)) {
      continue;
    }
    seen.add(name);
    out.push(name);
  }
  return out;
}
