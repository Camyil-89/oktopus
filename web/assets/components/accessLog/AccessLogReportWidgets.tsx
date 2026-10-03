"use client";

import type {
  AccessLogReportWidgetData,
  AccessLogReportWidgetResult,
  AccessLogReportWidgetType,
} from "@/types/accessLogReport";
import { ReportRuleRefLabel } from "@/assets/components/accessLog/ReportRuleRefLabel";
import type { ProxyRuleKind } from "@/assets/hooks/useProxyRuleNameMap";
import { SkeletonLoader } from "@/assets/components/SkeletonLoader";
import { Card, Input, Table, Tooltip } from "antd";
import type { ColumnsType } from "antd/es/table";
import { formatDurationUs } from "@/utils/formatDurationUs";
import { useTranslation } from "@/contexts/LocaleContext";
import { useEffect, useMemo, useRef, useState } from "react";

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
  return formatCount(value, localeTag);
}

function formatMetricAxisTick(value: number, metric?: string): string {
  if (isDurationMetric(metric)) {
    return formatDurationUs(value);
  }
  return formatTick(value);
}

export type ReportWidgetQueryMeta = {
  groupBy?: string;
  groupByCols?: string[];
  searchColumns?: string[];
};

export type ReportTableFetchParams = {
  page: number;
  page_size: number;
  search: string;
};

export type ReportTableFetchFn = (
  widgetId: string,
  params: ReportTableFetchParams,
) => Promise<AccessLogReportWidgetData>;

export type ReportRuleLabelContext = {
  ruleNames: ReadonlyMap<string, string>;
  ruleKinds: ReadonlyMap<string, ProxyRuleKind>;
  onOpenRule?: (ruleId: string, kind: ProxyRuleKind) => void;
};

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

function formatCount(n: number, localeTag: string): string {
  return new Intl.NumberFormat(localeTag, { maximumFractionDigits: 2 }).format(n);
}

function formatTick(value: number): string {
  if (value >= 1_000_000) {
    return `${(value / 1_000_000).toFixed(value >= 10_000_000 ? 0 : 1)}M`;
  }
  if (value >= 10_000) {
    return `${Math.round(value / 1000)}k`;
  }
  if (value >= 1000) {
    return `${(value / 1000).toFixed(value >= 5000 ? 0 : 1)}k`;
  }
  return String(Math.round(value));
}

function seriesColor(label: string, index: number): string {
  if (label === "allow") return "bg-teal-400/80";
  if (label === "deny") return "bg-red-400/75";
  const palette = [
    "bg-teal-400/70",
    "bg-sky-400/70",
    "bg-amber-400/70",
    "bg-violet-400/70",
  ];
  return palette[index % palette.length];
}

function TimeseriesBody({
  data,
  metric,
}: {
  data: AccessLogReportWidgetData;
  metric?: string;
}) {
  const { t, localeTag } = useTranslation();
  const series = data.series ?? [];
  const bucketKeys = useMemo(() => {
    const set = new Set<string>();
    for (const s of series) {
      for (const p of s.points) {
        set.add(p.t);
      }
    }
    return [...set].sort();
  }, [series]);

  const totals = useMemo(
    () =>
      bucketKeys.map((t) => {
        let sum = 0;
        for (const s of series) {
          const p = s.points.find((x) => x.t === t);
          if (p) sum += p.value;
        }
        return sum;
      }),
    [bucketKeys, series],
  );

  const yMax = useMemo(() => chartYMax(totals), [totals]);

  if (bucketKeys.length === 0) {
    return (
      <p className="m-0 py-8 text-center font-mono text-[12px] text-zinc-500">
        {t("reports.noData")}
      </p>
    );
  }

  return (
    <div className="flex flex-col gap-3">
      <div className="flex gap-2">
        <div
          className="flex h-36 w-10 shrink-0 flex-col justify-between py-px text-right font-mono text-[10px] text-zinc-500"
          aria-hidden
        >
          {[yMax, Math.round(yMax / 2), 0].map((tick) => (
            <span key={tick}>{formatMetricAxisTick(tick, metric)}</span>
          ))}
        </div>
        <div className="relative flex h-36 min-w-0 flex-1 items-end gap-1">
          {bucketKeys.map((t, bi) => {
            const total = totals[bi] ?? 0;
            const totalH = (total / yMax) * 100;
            return (
              <Tooltip
                key={t}
                title={
                  <div className="flex flex-col gap-1 font-mono text-[11px]">
                    <span>{new Date(t).toLocaleString(localeTag)}</span>
                    {series.map((s) => {
                      const p = s.points.find((x) => x.t === t);
                      if (!p || p.value <= 0) return null;
                      return (
                        <span key={s.key}>
                          {s.label}: {formatMetricValue(p.value, metric, localeTag)}
                        </span>
                      );
                    })}
                  </div>
                }
              >
                <div className="flex h-full min-w-0 flex-1 flex-col justify-end gap-[1px]">
                  {series.map((s, si) => {
                    const p = s.points.find((x) => x.t === t);
                    const v = p?.value ?? 0;
                    const h =
                      total > 0 ? (v / total) * totalH : 0;
                    if (h <= 0) return null;
                    return (
                      <div
                        key={s.key}
                        className={`min-h-[2px] w-full rounded-sm ${seriesColor(s.label, si)}`}
                        style={{ height: `${h}%` }}
                      />
                    );
                  })}
                  {total === 0 ? (
                    <div className="h-[2px] w-full rounded-sm bg-white/5" />
                  ) : null}
                </div>
              </Tooltip>
            );
          })}
        </div>
      </div>
      {series.length > 1 ? (
        <div className="flex flex-row flex-wrap gap-3 pl-12 font-mono text-[10px] text-zinc-500">
          {series.map((s, i) => (
            <span key={s.key} className="flex items-center gap-1.5">
              <span
                className={`inline-block h-2 w-2 rounded-sm ${seriesColor(s.label, i)}`}
              />
              {s.label}
            </span>
          ))}
        </div>
      ) : null}
    </div>
  );
}

function BarBody({
  data,
  groupBy,
  rules,
  metric,
}: {
  data: AccessLogReportWidgetData;
  groupBy?: string;
  rules: ReportRuleLabelContext;
  metric?: string;
}) {
  const { t } = useTranslation();
  const items = data.items ?? [];
  if (items.length === 0) {
    return (
      <p className="m-0 py-6 text-center font-mono text-[12px] text-zinc-500">
        {t("reports.noData")}
      </p>
    );
  }
  const max = Math.max(...items.map((i) => i.value), 1);
  return (
    <div className="flex flex-col gap-2">
      {items.map((item) => (
        <div key={item.label} className="flex flex-col gap-1">
          <div className="flex justify-between gap-2 font-mono text-[11px]">
            <ReportRuleRefLabel
              value={item.label}
              column={groupBy}
              ruleNames={rules.ruleNames}
              ruleKinds={rules.ruleKinds}
              onOpenRule={rules.onOpenRule}
              className="min-w-0 flex-1 text-zinc-400"
            />
            <span className="shrink-0 text-zinc-200">
              {formatMetricValue(item.value, metric)}
            </span>
          </div>
          <div className="h-2 overflow-hidden rounded-sm bg-white/5">
            <div
              className="h-full rounded-sm bg-teal-400/70"
              style={{ width: `${(item.value / max) * 100}%` }}
            />
          </div>
        </div>
      ))}
    </div>
  );
}

function StatBody({
  data,
  metric,
}: {
  data: AccessLogReportWidgetData;
  metric?: string;
}) {
  const { localeTag } = useTranslation();
  const v = data.value ?? 0;
  return (
    <p className="m-0 py-4 font-mono text-3xl tracking-tight text-zinc-50">
      {formatMetricValue(v, metric, localeTag)}
    </p>
  );
}

const TABLE_PAGE_SIZE_OPTIONS = ["20", "50", "100"];

function TableBody({
  widgetId,
  data,
  metric,
  rules,
  searchColumns,
  onFetchTable,
  onData,
}: {
  widgetId: string;
  data: AccessLogReportWidgetData;
  metric?: string;
  rules: ReportRuleLabelContext;
  searchColumns?: string[];
  onFetchTable?: ReportTableFetchFn;
  onData?: (data: AccessLogReportWidgetData) => void;
}) {
  const { t, localeTag } = useTranslation();
  const columns = data.columns ?? [];
  const rows = data.rows ?? [];
  const values = data.values ?? [];
  const [search, setSearch] = useState("");
  const [tableLoading, setTableLoading] = useState(false);
  const searchDebounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const page = data.page ?? 1;
  const pageSize = data.page_size ?? 20;
  const total = data.total ?? rows.length;

  const canSearch =
    (searchColumns?.length ?? 0) > 0 && onFetchTable !== undefined;

  const loadTable = async (params: ReportTableFetchParams) => {
    if (!onFetchTable) {
      return;
    }
    setTableLoading(true);
    try {
      const next = await onFetchTable(widgetId, params);
      onData?.(next);
    } finally {
      setTableLoading(false);
    }
  };

  useEffect(() => {
    return () => {
      if (searchDebounceRef.current) {
        clearTimeout(searchDebounceRef.current);
      }
    };
  }, []);

  if (columns.length === 0) {
    return null;
  }
  const metricLabel = isDurationMetric(metric)
    ? t("reports.metricAvg")
    : t("reports.metricCount");
  const tableColumns: ColumnsType<Record<string, string>> = [
    ...columns.map((c) => ({
      key: c,
      title: c,
      dataIndex: c,
      ellipsis: true,
      render: (v: string) => (
        <ReportRuleRefLabel
          value={v}
          column={c}
          ruleNames={rules.ruleNames}
          ruleKinds={rules.ruleKinds}
          onOpenRule={rules.onOpenRule}
        />
      ),
    })),
    {
      key: "value",
      title: metricLabel,
      dataIndex: "value",
      width: 120,
      align: "right" as const,
      render: (v: string) => formatMetricValue(Number(v), metric, localeTag),
    },
  ];
  const dataSource = rows.map((row, i) => {
    const rec: Record<string, string> = { key: String(i), value: String(values[i] ?? 0) };
    columns.forEach((c, ci) => {
      rec[c] = row[ci] ?? "—";
    });
    return rec;
  });
  return (
    <div className="flex flex-col gap-2">
      {canSearch ? (
        <Input.Search
          allowClear
          placeholder={t("reports.searchPlaceholder")}
          value={search}
          loading={tableLoading}
          onChange={(e) => {
            const q = e.target.value;
            setSearch(q);
            if (searchDebounceRef.current) {
              clearTimeout(searchDebounceRef.current);
            }
            searchDebounceRef.current = setTimeout(() => {
              void loadTable({ page: 1, page_size: pageSize, search: q });
            }, 350);
          }}
          onSearch={(q) => {
            if (searchDebounceRef.current) {
              clearTimeout(searchDebounceRef.current);
            }
            setSearch(q);
            void loadTable({ page: 1, page_size: pageSize, search: q });
          }}
        />
      ) : null}
      <Table
        size="small"
        loading={tableLoading}
        columns={tableColumns}
        dataSource={dataSource}
        scroll={{ x: true }}
        pagination={{
          current: page,
          pageSize,
          total,
          showSizeChanger: Boolean(onFetchTable),
          pageSizeOptions: TABLE_PAGE_SIZE_OPTIONS,
          hideOnSinglePage: false,
          onChange: (nextPage, nextSize) => {
            if (!onFetchTable) {
              return;
            }
            void loadTable({
              page: nextPage,
              page_size: nextSize ?? pageSize,
              search,
            });
          },
        }}
      />
    </div>
  );
}

function WidgetInner({
  widgetId,
  type,
  data,
  metric,
  queryMeta,
  rules,
  onFetchTable,
  onTableData,
}: {
  widgetId: string;
  type: AccessLogReportWidgetType;
  data: AccessLogReportWidgetData;
  metric?: string;
  queryMeta?: ReportWidgetQueryMeta;
  rules: ReportRuleLabelContext;
  onFetchTable?: ReportTableFetchFn;
  onTableData?: (data: AccessLogReportWidgetData) => void;
}) {
  switch (type) {
    case "timeseries":
      return <TimeseriesBody data={data} metric={metric} />;
    case "bar":
      return (
        <BarBody
          data={data}
          groupBy={queryMeta?.groupBy}
          rules={rules}
          metric={metric}
        />
      );
    case "stat":
      return <StatBody data={data} metric={metric} />;
    case "table":
      return (
        <TableBody
          widgetId={widgetId}
          data={data}
          metric={metric}
          rules={rules}
          searchColumns={queryMeta?.searchColumns}
          onFetchTable={onFetchTable}
          onData={onTableData}
        />
      );
    default:
      return null;
  }
}

type ReportWidgetCardProps = {
  widget: AccessLogReportWidgetResult;
  metric?: string;
  queryMeta?: ReportWidgetQueryMeta;
  rules: ReportRuleLabelContext;
  loading?: boolean;
  onFetchTable?: ReportTableFetchFn;
  onTableData?: (data: AccessLogReportWidgetData) => void;
};

export function AccessLogReportWidgetCard({
  widget,
  metric,
  queryMeta,
  rules,
  loading,
  onFetchTable,
  onTableData,
}: ReportWidgetCardProps) {
  return (
    <Card
      size="small"
      title={
        widget.title ? (
          <span className="text-[13px] font-medium text-zinc-200">
            {widget.title}
          </span>
        ) : (
          <span className="font-mono text-[12px] text-zinc-500">{widget.id}</span>
        )
      }
      className="border-white/10 bg-white/[0.03]"
    >
      {loading ? (
        <SkeletonLoader className="h-24" loaderSize="md">
          <div className="h-24 animate-pulse rounded bg-white/5" />
        </SkeletonLoader>
      ) : (
        <WidgetInner
          widgetId={widget.id}
          type={widget.type}
          data={widget.data}
          metric={metric}
          queryMeta={queryMeta}
          rules={rules}
          onFetchTable={onFetchTable}
          onTableData={onTableData}
        />
      )}
    </Card>
  );
}

type ReportGridProps = {
  widgets: AccessLogReportWidgetResult[];
  loading?: boolean;
  widgetMetrics?: Record<string, string>;
  widgetQueryMeta?: Record<string, ReportWidgetQueryMeta>;
  rules: ReportRuleLabelContext;
  onFetchTable?: ReportTableFetchFn;
  onTableWidgetData?: (
    widgetId: string,
    data: AccessLogReportWidgetData,
  ) => void;
};

export function AccessLogReportWidgetGrid({
  widgets,
  loading,
  widgetMetrics,
  widgetQueryMeta,
  rules,
  onFetchTable,
  onTableWidgetData,
}: ReportGridProps) {
  if (!loading && widgets.length === 0) {
    return null;
  }
  return (
    <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
      {widgets.map((w) => (
        <div
          key={w.id}
          className={w.type === "table" ? "lg:col-span-2" : undefined}
        >
          <AccessLogReportWidgetCard
            widget={w}
            metric={widgetMetrics?.[w.id]}
            queryMeta={widgetQueryMeta?.[w.id]}
            rules={rules}
            loading={loading}
            onFetchTable={onFetchTable}
            onTableData={
              onTableWidgetData
                ? (data) => onTableWidgetData(w.id, data)
                : undefined
            }
          />
        </div>
      ))}
    </div>
  );
}
