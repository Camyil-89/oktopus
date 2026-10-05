import type {
  AccessLogReportSpec,
  AccessLogReportWidgetQuery,
} from "@/types/accessLogReport";

const CYCLIC_GROUP_BY_TIME = new Set([
  "hour_of_day",
  "day_of_week",
  "month_of_year",
]);

export type CalendarGroupByTime = "10m" | "1h" | "1d";

/** Шаг оси времени по длительности выбранного периода (календарные бакеты). */
export function autoGroupByTimeForRange(
  fromIso: string,
  toIso: string,
): CalendarGroupByTime {
  const from = new Date(fromIso).getTime();
  const to = new Date(toIso).getTime();
  const ms = Math.max(0, to - from);
  const hours = ms / (3600 * 1000);
  const days = hours / 24;

  if (hours <= 6) {
    return "10m";
  }
  if (days <= 2) {
    return "1h";
  }
  return "1d";
}

function isCyclicGroupByTime(
  g?: AccessLogReportWidgetQuery["group_by_time"],
): boolean {
  return g !== undefined && CYCLIC_GROUP_BY_TIME.has(g);
}

/** Подставляет auto group_by_time в timeseries с календарным шагом перед запросом отчёта. */
export function applyAutoTimeBucketsToSpec(
  spec: AccessLogReportSpec,
): AccessLogReportSpec {
  const bucket = autoGroupByTimeForRange(spec.time.from, spec.time.to);
  return {
    ...spec,
    widgets: spec.widgets.map((w) => {
      if (w.type !== "timeseries") {
        return w;
      }
      if (isCyclicGroupByTime(w.query.group_by_time)) {
        return w;
      }
      return {
        ...w,
        query: {
          ...w.query,
          group_by_time: bucket,
        },
      };
    }),
  };
}
