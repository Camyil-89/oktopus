"use client";

import type { AccessLogReportTimeseriesSeries } from "@/types/accessLogReport";
import { formatDurationUs } from "@/utils/formatDurationUs";
import {
  buildValueLookup,
  groupTimeseriesBucketKeys,
  maxTimeseriesBarsForWidth,
  sumSeriesInBucketGroup,
  type TimeseriesBucketGroup,
} from "@/utils/mergeTimeseriesBuckets";
import { Tooltip } from "antd";
import { useEffect, useMemo, useRef, useState } from "react";

function normalizeBucketKey(iso: string): string {
  const t = new Date(iso).getTime();
  if (Number.isNaN(t)) {
    return iso;
  }
  return new Date(t).toISOString();
}

export function formatTimeseriesBucketLabel(
  iso: string,
  groupByTime?: string,
  localeTag = "ru-RU",
): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) {
    return iso;
  }
  switch (groupByTime) {
    case "hour_of_day":
      return d.toLocaleTimeString(localeTag, {
        hour: "2-digit",
        minute: "2-digit",
      });
    case "day_of_week":
      return d.toLocaleDateString(localeTag, { weekday: "short" });
    case "month_of_year":
      return d.toLocaleDateString(localeTag, { month: "short" });
    case "1d":
      return d.toLocaleDateString(localeTag, {
        day: "2-digit",
        month: "short",
      });
    case "10m":
      return d.toLocaleString(localeTag, {
        hour: "2-digit",
        minute: "2-digit",
      });
    default:
      return d.toLocaleString(localeTag, {
        day: "2-digit",
        month: "short",
        hour: "2-digit",
        minute: "2-digit",
      });
  }
}

function aggregateSeriesInGroup(
  group: TimeseriesBucketGroup,
  seriesLabel: string,
  lookup: Map<string, number>,
  metric?: string,
): number {
  if (isDurationMetric(metric)) {
    const vals = group.keys
      .map((k) => lookup.get(`${k}\0${seriesLabel}`) ?? 0)
      .filter((v) => v > 0);
    if (vals.length === 0) {
      return 0;
    }
    return vals.reduce((a, b) => a + b, 0) / vals.length;
  }
  return sumSeriesInBucketGroup(group.keys, seriesLabel, lookup);
}

function formatTimeseriesAxisLabel(
  iso: string,
  groupByTime?: string,
  localeTag = "ru-RU",
): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) {
    return iso;
  }
  if (groupByTime === "1d" || groupByTime === undefined) {
    return d.toLocaleDateString(localeTag, {
      day: "2-digit",
      month: "2-digit",
    });
  }
  return formatTimeseriesBucketLabel(iso, groupByTime, localeTag);
}

function formatTimeseriesAxisRange(
  start: string,
  end: string,
  groupByTime?: string,
  localeTag = "ru-RU",
): string {
  if (start === end) {
    return formatTimeseriesAxisLabel(start, groupByTime, localeTag);
  }
  return `${formatTimeseriesAxisLabel(start, groupByTime, localeTag)}–${formatTimeseriesAxisLabel(end, groupByTime, localeTag)}`;
}

function formatTimeseriesTooltipRange(
  group: TimeseriesBucketGroup,
  groupByTime?: string,
  localeTag = "ru-RU",
): string {
  if (group.keys.length <= 1) {
    return formatTimeseriesBucketLabel(group.sortKeyStart, groupByTime, localeTag);
  }
  return `${formatTimeseriesBucketLabel(group.sortKeyStart, groupByTime, localeTag)} – ${formatTimeseriesBucketLabel(group.sortKeyEnd, groupByTime, localeTag)}`;
}

function isDurationMetric(metric?: string): boolean {
  return metric === "avg_decide_duration_us";
}

function formatMetricValue(
  value: number,
  metric?: string,
  localeTag = "ru-RU",
): string {
  if (isDurationMetric(metric)) {
    return formatDurationUs(value);
  }
  return new Intl.NumberFormat(localeTag, { maximumFractionDigits: 2 }).format(
    value,
  );
}

function seriesColor(label: string): string {
  if (label === "allow") {
    return "#2dd4bf";
  }
  if (label === "deny") {
    return "#f87171";
  }
  const palette = ["#38bdf8", "#fbbf24", "#a78bfa", "#94a3b8"];
  return palette[Math.max(0, label.length % palette.length)]!;
}

type ChartRow = {
  bucket: string;
  series: string;
  value: number;
  sortKey: string;
};

export function buildTimeseriesChartRows(
  series: AccessLogReportTimeseriesSeries[],
  localeTag: string,
  groupByTime?: string,
): ChartRow[] {
  const bucketSet = new Set<string>();
  for (const s of series) {
    for (const p of s.points) {
      bucketSet.add(normalizeBucketKey(p.t));
    }
  }
  const bucketKeys = [...bucketSet].sort();
  const rows: ChartRow[] = [];

  if (bucketKeys.length === 0) {
    return rows;
  }

  const seriesList =
    series.length > 0
      ? series
      : [{ key: "", label: "—", points: [] as { t: string; value: number }[] }];

  for (const t of bucketKeys) {
    const bucket = formatTimeseriesBucketLabel(t, groupByTime, localeTag);
    for (const s of seriesList) {
      const p = s.points.find((x) => normalizeBucketKey(x.t) === t);
      rows.push({
        bucket,
        series: s.label,
        value: p?.value ?? 0,
        sortKey: t,
      });
    }
  }
  return rows;
}

function chartYMax(values: number[]): number {
  const peak = values.length ? Math.max(...values) : 0;
  if (peak <= 0) {
    return 1;
  }
  const exp = 10 ** Math.floor(Math.log10(peak));
  const n = peak / exp;
  let nice = 10;
  if (n <= 1) nice = 1;
  else if (n <= 2) nice = 2;
  else if (n <= 5) nice = 5;
  return nice * exp;
}

type Props = {
  series: AccessLogReportTimeseriesSeries[];
  metric?: string;
  groupByTime?: string;
  localeTag: string;
  emptyLabel: string;
};

export function AccessLogReportTimeseriesChart({
  series,
  metric,
  groupByTime,
  localeTag,
  emptyLabel,
}: Props) {
  const data = useMemo(
    () => buildTimeseriesChartRows(series, localeTag, groupByTime),
    [series, groupByTime, localeTag],
  );

  const bucketKeys = useMemo(() => {
    const seen = new Set<string>();
    const keys: string[] = [];
    for (const row of data) {
      if (!seen.has(row.sortKey)) {
        seen.add(row.sortKey);
        keys.push(row.sortKey);
      }
    }
    return keys;
  }, [data]);

  const plotRef = useRef<HTMLDivElement>(null);
  const [maxBars, setMaxBars] = useState(28);

  useEffect(() => {
    const el = plotRef.current;
    if (!el) {
      return;
    }
    const update = () => {
      setMaxBars(maxTimeseriesBarsForWidth(el.clientWidth));
    };
    update();
    const ro = new ResizeObserver(update);
    ro.observe(el);
    return () => ro.disconnect();
  }, []);

  const valueLookup = useMemo(() => buildValueLookup(data), [data]);

  const stacked = series.length > 1;

  const groups = useMemo(
    () => groupTimeseriesBucketKeys(bucketKeys, maxBars),
    [bucketKeys, maxBars],
  );

  const groupTotals = useMemo(
    () =>
      groups.map((g) => {
        if (stacked) {
          return series.reduce(
            (sum, s) =>
              sum +
              aggregateSeriesInGroup(g, s.label, valueLookup, metric),
            0,
          );
        }
        const only = series[0];
        return only
          ? aggregateSeriesInGroup(g, only.label, valueLookup, metric)
          : aggregateSeriesInGroup(g, "—", valueLookup, metric);
      }),
    [groups, series, valueLookup, metric, stacked],
  );

  const yMax = chartYMax(groupTotals);
  const hasAnyValue = data.some((r) => r.value > 0);

  if (data.length === 0 || !hasAnyValue) {
    return (
      <p className="m-0 py-8 text-center font-mono text-[12px] text-zinc-500">
        {emptyLabel}
      </p>
    );
  }

  return (
    <div className="flex flex-col gap-3">
      <div className="flex gap-2">
        <div
          className="flex h-60 w-10 shrink-0 flex-col justify-between py-px text-right font-mono text-[10px] text-zinc-500"
          aria-hidden
        >
          {[yMax, Math.round(yMax / 2), 0].map((tick) => (
            <span key={tick}>
              {isDurationMetric(metric)
                ? formatDurationUs(tick)
                : formatMetricValue(tick, metric, localeTag)}
            </span>
          ))}
        </div>
        <div ref={plotRef} className="flex min-w-0 flex-1 flex-col gap-1">
          <div className="flex h-60 items-stretch gap-px">
            {groups.map((group, gi) => {
              const total = groupTotals[gi] ?? 0;
              const totalH = (total / yMax) * 100;
              const tooltipTitle = formatTimeseriesTooltipRange(
                group,
                groupByTime,
                localeTag,
              );
              return (
                <Tooltip
                  key={group.sortKeyStart}
                  title={
                    <div className="flex flex-col gap-1 font-mono text-[11px]">
                      <span>{tooltipTitle}</span>
                      {stacked
                        ? series.map((s) => {
                            const v = aggregateSeriesInGroup(
                              group,
                              s.label,
                              valueLookup,
                              metric,
                            );
                            return (
                              <span key={s.key}>
                                {s.label}:{" "}
                                {formatMetricValue(v, metric, localeTag)}
                              </span>
                            );
                          })
                        : (
                            <span>
                              {formatMetricValue(total, metric, localeTag)}
                            </span>
                          )}
                    </div>
                  }
                  mouseEnterDelay={0.05}
                >
                  <div className="flex h-full min-h-full w-full min-w-0 flex-1 cursor-default flex-col justify-end gap-[1px]">
                    {stacked
                      ? series.map((s) => {
                          const v = aggregateSeriesInGroup(
                            group,
                            s.label,
                            valueLookup,
                            metric,
                          );
                          const h = total > 0 ? (v / total) * totalH : 0;
                          if (total <= 0) {
                            return (
                              <div
                                key={s.key}
                                className="h-[1px] w-full bg-white/10"
                              />
                            );
                          }
                          if (h <= 0) {
                            return null;
                          }
                          return (
                            <div
                              key={s.key}
                              className="min-h-[1px] w-full rounded-sm"
                              style={{
                                height: `${h}%`,
                                backgroundColor: seriesColor(s.label),
                              }}
                            />
                          );
                        })
                      : (
                          <div
                            className="w-full rounded-sm"
                            style={{
                              height: `${totalH}%`,
                              minHeight: total > 0 ? 2 : 1,
                              backgroundColor: seriesColor(
                                series[0]?.label ?? "—",
                              ),
                            }}
                          />
                        )}
                  </div>
                </Tooltip>
              );
            })}
          </div>
          <div className="flex gap-px">
            {groups.map((group) => (
              <div
                key={`lbl-${group.sortKeyStart}`}
                className="min-w-0 flex-1 truncate text-center font-mono text-[9px] leading-tight text-zinc-500"
                title={formatTimeseriesTooltipRange(
                  group,
                  groupByTime,
                  localeTag,
                )}
              >
                {formatTimeseriesAxisRange(
                  group.sortKeyStart,
                  group.sortKeyEnd,
                  groupByTime,
                  localeTag,
                )}
              </div>
            ))}
          </div>
        </div>
      </div>
      {stacked ? (
        <div className="flex flex-row flex-wrap gap-3 pl-12 font-mono text-[10px] text-zinc-500">
          {series.map((s) => (
            <span key={s.key} className="flex items-center gap-1.5">
              <span
                className="inline-block h-2 w-2 rounded-sm"
                style={{ backgroundColor: seriesColor(s.label) }}
              />
              {s.label}
            </span>
          ))}
        </div>
      ) : null}
    </div>
  );
}
