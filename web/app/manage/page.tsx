"use client";

import { ApiError } from "@/api/base";
import * as proxyApi from "@/api/proxy";
import { DashboardStatCard } from "@/assets/components/dashboard/DashboardStatCard";
import type { DashboardStatTone } from "@/assets/components/dashboard/DashboardStatCard";
import { ACLDecideBreakdownTooltip } from "@/assets/components/dashboard/ACLDecideBreakdownTooltip";
import { InspectBreakdownTooltip } from "@/assets/components/dashboard/InspectBreakdownTooltip";
import { PolicyBreakdownTooltip } from "@/assets/components/dashboard/PolicyBreakdownTooltip";
import { TrafficAllowDenyChart } from "@/assets/components/dashboard/TrafficAllowDenyChart";
import { TrafficThroughputChart } from "@/assets/components/dashboard/TrafficThroughputChart";
import type { ProxyRuntimeStatus } from "@/types/proxy";
import { useApiErrorMessage, useTranslation } from "@/contexts/LocaleContext";
import type { TranslateFn } from "@/i18n/translate";
import { byteUnitLabelsFromT, formatBytes } from "@/utils/formatBytes";
import { durationUnitLabelsFromT, formatDurationUs } from "@/utils/formatDurationUs";
import { OktopusLoading } from "@/assets/components/oktopus/OktopusLoading";
import { Alert, Tag, Tooltip, Typography } from "antd";
import Link from "next/link";
import { useCallback, useEffect, useMemo, useState, type ReactNode } from "react";

function aclBuildLabel(st: ProxyRuntimeStatus["acl"], t: TranslateFn): string {
  switch (st.build_status) {
    case "building":
      return t("dashboard.aclBuilding");
    case "error":
      return t("dashboard.aclError");
    case "ready":
      return "OK";
    default:
      return t("dashboard.aclPending");
  }
}

function aclBuildTone(st: ProxyRuntimeStatus["acl"]): DashboardStatTone {
  switch (st.build_status) {
    case "ready":
      return "success";
    case "error":
      return "danger";
    case "building":
      return "warning";
    default:
      return "neutral";
  }
}

function formatSharePct(part: number, total: number): string {
  if (total <= 0) {
    return "—";
  }
  const pct = (part / total) * 100;
  if (pct >= 100) {
    return "100%";
  }
  if (pct <= 0) {
    return "0%";
  }
  return pct < 10 ? `${pct.toFixed(1)}%` : `${Math.round(pct)}%`;
}

function formatInt(n: number, localeTag: string): string {
  return new Intl.NumberFormat(localeTag).format(n);
}

function formatStartupClock(iso?: string, localeTag = "ru-RU"): string {
  const emDash = "—";
  if (!iso) return emDash;
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return emDash;
  return d.toLocaleString(localeTag, {
    day: "2-digit",
    month: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  });
}

function aclCompileDurationMs(
  acl: ProxyRuntimeStatus["acl"] | undefined,
): number | undefined {
  if (acl?.last_compile_duration_ms != null) {
    return acl.last_compile_duration_ms;
  }
  if (acl?.build_started_at && acl?.build_finished_at) {
    const ms =
      new Date(acl.build_finished_at).getTime() -
      new Date(acl.build_started_at).getTime();
    if (ms >= 0) {
      return ms > 0 ? ms : 1;
    }
  }
  return undefined;
}

function formatDurationMs(ms?: number, t?: TranslateFn): string {
  const emDash = "—";
  if (ms == null || ms < 0) return emDash;
  if (!t) {
    if (ms < 1000) return `${ms} ms`;
    const sec = ms / 1000;
    return sec < 60 ? `${sec.toFixed(1)} s` : `${Math.floor(sec / 60)} min`;
  }
  if (ms < 1000) return `${ms} ${t("units.ms")}`;
  const sec = ms / 1000;
  if (sec < 60) {
    return `${sec.toFixed(sec < 10 ? 1 : 0)} ${t("units.sec")}`;
  }
  const min = Math.floor(sec / 60);
  const rest = Math.round(sec % 60);
  return rest > 0
    ? t("units.minSec", { min, sec: rest })
    : t("units.minOnly", { min });
}

function formatDelayFromServeStart(ms?: number, t?: TranslateFn): string {
  if (!t) {
    if (ms == null || ms < 0) return "—";
    return formatDurationMs(ms);
  }
  if (ms == null || ms < 0) return t("dashboard.notYet");
  return t("dashboard.delayAfterServe", {
    duration: formatDurationMs(ms, t),
  });
}

function StatusDot({ tone }: { tone: DashboardStatTone }) {
  const dotClass: Record<DashboardStatTone, string> = {
    success: "bg-teal-400",
    warning: "bg-amber-400 animate-pulse",
    danger: "bg-red-400",
    neutral: "bg-zinc-600",
  };
  return (
    <span className={`h-1.5 w-1.5 rounded-full ${dotClass[tone]}`} />
  );
}

function MetricsWindow5mBadge({ label }: { label: string }) {
  return (
    <span className="font-mono text-[11px] text-teal-300/70">{label}</span>
  );
}

export default function ManagePage() {
  const { t, localeTag } = useTranslation();
  const formatApiError = useApiErrorMessage();
  const byteUnits = useMemo(() => byteUnitLabelsFromT(t), [t]);
  const durationUnits = useMemo(() => durationUnitLabelsFromT(t), [t]);
  const emDash = t("common.emDash");
  const [status, setStatus] = useState<ProxyRuntimeStatus | null>(null);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    try {
      const st = await proxyApi.getProxyRuntimeStatus();
      setStatus(st);
      setError(null);
    } catch (e) {
      setError(formatApiError(e, t("dashboard.loadFailed")));
    }
  }, [formatApiError, t]);

  useEffect(() => {
    void load();
    const t = setInterval(() => void load(), 3000);
    return () => clearInterval(t);
  }, [load]);

  const acl = status?.acl;
  const traffic = status?.traffic;
  const loading = !status && !error;
  const proxyUp = Boolean(status?.proxy_active);
  const totalDecisions =
    (traffic?.allowed_5m ?? 0) + (traffic?.denied_5m ?? 0);
  const reqPerMin =
    traffic?.requests_1m ??
    (traffic ? Math.round(traffic.requests_per_sec_avg_5m * 60) : 0);
  const reqPerMinAvg5m = traffic
    ? Math.round(traffic.requests_per_sec_avg_5m * 60)
    : 0;
  const reqLast5s = traffic
    ? Math.round(traffic.requests_per_sec_now_5s * 5)
    : 0;
  const buckets = traffic?.buckets_10s ?? [];
  const startup = status?.startup;
  const bytesTotalUp =
    (traffic?.bytes_total_up_allow ?? 0) + (traffic?.bytes_total_up_deny ?? 0);
  const bytesTotalDown =
    (traffic?.bytes_total_down_allow ?? 0) +
    (traffic?.bytes_total_down_deny ?? 0);
  const bytesTotal = bytesTotalUp + bytesTotalDown;

  return (
    <div className="flex w-full flex-col gap-3">
      {error ? (
        <Alert type="error" message={error} showIcon />
      ) : null}

      {!proxyUp && status?.proxy_start_error ? (
        <Alert
          type="warning"
          showIcon
          message={t("dashboard.proxyDown")}
          description={status.proxy_start_error}
        />
      ) : null}

      <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
        <DashboardStatCard
          label={t("dashboard.proxy")}
          loading={loading}
          tone={proxyUp ? "success" : "danger"}
          badge={<StatusDot tone={proxyUp ? "success" : "danger"} />}
          value={proxyUp ? "RUNNING" : "STOPPED"}
          hint={status?.listen || "—"}
        />
        <DashboardStatCard
          label={t("dashboard.aclBuild")}
          loading={loading && !acl}
          tone={acl ? aclBuildTone(acl) : undefined}
          badge={
            acl ? <StatusDot tone={aclBuildTone(acl)} /> : undefined
          }
          value={acl ? aclBuildLabel(acl, t) : emDash}
          hint={
            acl
              ? t("dashboard.rulesPatterns", {
                  rules: acl.active_logical_rules,
                  patterns: acl.active_patterns,
                })
              : undefined
          }
        />
        <DashboardStatCard
          label={t("dashboard.perMinute")}
          loading={loading}
          badge={
            traffic && proxyUp ? (
              <span className="font-mono text-[11px] text-zinc-500">{t("dashboard.sliding")}</span>
            ) : undefined
          }
          value={
            proxyUp ? (
              <span className="font-mono">{formatInt(reqPerMin, localeTag)}</span>
            ) : (
              "—"
            )
          }
          hint={
            traffic && proxyUp
              ? t("dashboard.reqHint", {
                  last5s: formatInt(reqLast5s, localeTag),
                  avg: formatInt(reqPerMinAvg5m, localeTag),
                })
              : undefined
          }
        />
        <DashboardStatCard
          label="Allow / Deny"
          loading={loading}
          badge={
            traffic ? (
              <MetricsWindow5mBadge label={t("dashboard.window5m")} />
            ) : undefined
          }
          value={
            traffic ? (
              <span className="font-mono text-xl">
                <span className="text-teal-300">
                  {formatInt(traffic.allowed_5m, localeTag)}
                </span>
                <span className="mx-1.5 text-zinc-600">/</span>
                <span className="text-red-300/90">
                  {formatInt(traffic.denied_5m, localeTag)}
                </span>
              </span>
            ) : (
              "—"
            )
          }
          hint={
            traffic
              ? t("dashboard.trafficHint", {
                  allowPct: formatSharePct(traffic.allowed_5m, totalDecisions),
                  denyPct: formatSharePct(traffic.denied_5m, totalDecisions),
                })
              : undefined
          }
        />
      </div>

      {acl?.build_error ? (
        <Alert type="error" message={acl.build_error} showIcon />
      ) : null}

      <div className="rounded-xl border border-white/10 bg-white/5 p-5">
        <p className="text-[13px] font-medium text-zinc-100">{t("dashboard.startupTitle")}</p>
        <p className="mb-4 mt-0.5 font-mono text-[11.5px] text-zinc-500">
          {t("dashboard.startupHint")}
        </p>
        {loading ? (
          <div className="flex justify-center py-6">
            <OktopusLoading size="lg" />
          </div>
        ) : (
          <ul className="grid gap-3 sm:grid-cols-3">
            <li className="flex flex-col gap-1 rounded-lg border border-white/8 bg-black/20 px-4 py-3">
              <span className="text-[11px] text-zinc-500">
                {t("dashboard.serveStart")}
              </span>
              <span className="font-mono text-[13px] text-zinc-100">
                {formatStartupClock(startup?.started_at, localeTag)}
              </span>
            </li>
            <li className="flex flex-col gap-1.5 rounded-lg border border-white/8 bg-black/20 px-4 py-3">
              <span className="text-[11px] text-zinc-500">
                {t("dashboard.proxyReady")}
              </span>
              <span className="text-[13px] leading-snug text-teal-200/90">
                {formatDelayFromServeStart(startup?.proxy_ready_ms, t)}
              </span>
              <span className="font-mono text-[11px] text-zinc-600">
                {t("dashboard.moment", {
                  time: formatStartupClock(startup?.proxy_ready_at, localeTag),
                })}
              </span>
            </li>
            <li className="flex flex-col gap-1.5 rounded-lg border border-white/8 bg-black/20 px-4 py-3">
              <span className="text-[11px] text-zinc-500">
                {t("dashboard.aclCompile")}
              </span>
              <span className="text-[13px] leading-snug text-teal-200/90">
                {acl?.build_status === "building"
                  ? t("dashboard.inProgress")
                  : formatDurationMs(aclCompileDurationMs(acl), t)}
              </span>
              <span className="font-mono text-[11px] text-zinc-600">
                {acl?.build_finished_at
                  ? t("dashboard.aclDoneAt", {
                      time: formatStartupClock(acl.build_finished_at, localeTag),
                    })
                  : t("dashboard.aclNever")}
              </span>
            </li>
          </ul>
        )}
      </div>

      <div className="grid gap-3 lg:grid-cols-3">
        <div className="rounded-xl border border-white/10 bg-white/[0.05] p-5 lg:col-span-2">
          <div className="mb-6 flex flex-row flex-wrap items-center justify-between gap-3">
            <div>
              <p className="text-[13px] font-medium text-zinc-100">{t("dashboard.traffic")}</p>
              <p className="mt-0.5 font-mono text-[11.5px] text-zinc-500">
                {t("dashboard.trafficHintChart")}
              </p>
            </div>
            <div className="flex flex-row items-center gap-3 font-mono text-[11px]">
              <span className="flex flex-row items-center gap-1.5 text-zinc-500">
                <span className="h-2 w-2 rounded-sm bg-teal-400/75" />
                allow
              </span>
              <span className="flex flex-row items-center gap-1.5 text-zinc-500">
                <span className="h-2 w-2 rounded-sm bg-red-400/50" />
                deny
              </span>
            </div>
          </div>
          {!proxyUp && !loading ? (
            <Typography.Text type="secondary">
              {t("dashboard.chartWhenUp")}
            </Typography.Text>
          ) : (
            <TrafficAllowDenyChart buckets={buckets} loading={loading} />
          )}
          <div className="mt-8 border-t border-white/8 pt-6">
            <div className="mb-6 flex flex-row flex-wrap items-center justify-between gap-3">
              <div>
                <p className="text-[13px] font-medium text-zinc-100">
                  {t("dashboard.throughput")}
                </p>
                <p className="mt-0.5 font-mono text-[11.5px] text-zinc-500">
                  {t("dashboard.throughputHint")}
                </p>
              </div>
              <div className="flex flex-row items-center gap-3 font-mono text-[11px]">
                <span className="flex flex-row items-center gap-1.5 text-zinc-500">
                  <span className="h-2 w-2 rounded-sm bg-amber-400/70" />
                  {t("dashboard.egress")}
                </span>
                <span className="flex flex-row items-center gap-1.5 text-zinc-500">
                  <span className="h-2 w-2 rounded-sm bg-sky-400/70" />
                  {t("dashboard.ingress")}
                </span>
              </div>
            </div>
            {!proxyUp && !loading ? (
              <Typography.Text type="secondary">
                {t("dashboard.chartWhenUp")}
              </Typography.Text>
            ) : (
              <TrafficThroughputChart buckets={buckets} loading={loading} />
            )}
          </div>
        </div>

        <div className="rounded-xl border border-white/10 bg-white/[0.05] p-5">
          <p className="text-[13px] font-medium text-zinc-100">Runtime</p>
          <p className="mb-5 mt-0.5 font-mono text-[11.5px] text-zinc-500">
            {t("dashboard.runtimeHint")}
          </p>
          {loading ? (
            <div className="flex justify-center py-8">
              <OktopusLoading size="lg" />
            </div>
          ) : (
            <ul className="flex flex-col gap-3.5 text-[12.5px]">
              <RuntimeRow
                label={t("dashboard.connections")}
                value={
                  proxyUp && traffic
                    ? String(traffic.active_connections)
                    : "—"
                }
              />
              <RuntimeRow
                label={t("dashboard.wsMitm")}
                value={
                  proxyUp && traffic
                    ? String(traffic.active_websocket_connections ?? 0)
                    : "—"
                }
              />
              <RuntimeRow
                label={t("dashboard.rulesInProxy")}
                value={
                  acl ? (
                    acl.rules_in_sync ? (
                      <Tag color="success" className="!m-0">
                        {t("dashboard.rulesCurrent")}
                      </Tag>
                    ) : (
                      <Tag color="warning" className="!m-0">
                        {t("dashboard.rulesStale")}
                      </Tag>
                    )
                  ) : (
                    "—"
                  )
                }
              />
              <RuntimeRow
                label="SNI index / regexp"
                tooltip={
                  acl ? (
                    <div className="flex max-w-[340px] flex-col gap-2 text-[11px]">
                      <p className="m-0 text-zinc-400">
                        {t("dashboard.sniTooltip")}
                      </p>
                      {acl.sni_regexp_reason_counts &&
                      Object.keys(acl.sni_regexp_reason_counts).length > 0 ? (
                        <div className="border-t border-white/10 pt-2 text-zinc-500">
                          <p className="m-0 mb-1 text-[10px] uppercase tracking-wide">
                            {t("dashboard.regexpReasons")}
                          </p>
                          <ul className="m-0 list-none space-y-0.5 font-mono text-zinc-300">
                            {Object.entries(acl.sni_regexp_reason_counts).map(
                              ([k, v]) => (
                                <li key={k}>
                                  {k}: {v}
                                </li>
                              ),
                            )}
                          </ul>
                        </div>
                      ) : null}
                      {acl.sni_regexp_samples &&
                      acl.sni_regexp_samples.length > 0 ? (
                        <div className="border-t border-white/10 pt-2 text-zinc-500">
                          <p className="m-0 mb-1 text-[10px] uppercase tracking-wide">
                            {t("dashboard.regexpSamples")}
                          </p>
                          <ul className="m-0 max-h-40 list-none space-y-1 overflow-y-auto font-mono text-[10px] text-zinc-400">
                            {acl.sni_regexp_samples.slice(0, 12).map((s) => (
                              <li key={`${s.reason}:${s.line}`}>
                                <span className="text-zinc-500">{s.reason}</span>{" "}
                                {s.line}
                              </li>
                            ))}
                          </ul>
                        </div>
                      ) : null}
                    </div>
                  ) : undefined
                }
                value={
                  acl
                    ? `${acl.active_sni_patterns_indexed ?? "—"} / ${acl.active_sni_patterns_regexp ?? "—"} · slow ${acl.active_slow_rule_slots ?? "—"}`
                    : "—"
                }
              />
              <RuntimeRow
                label={t("dashboard.usersSources")}
                value={
                  traffic && proxyUp
                    ? `${traffic.unique_users_5m} / ${traffic.unique_sources_5m}`
                    : "—"
                }
              />
              <RuntimeRow
                label={t("dashboard.accessLogQueue")}
                value={
                  traffic ? String(traffic.access_log_queue_pending) : "—"
                }
              />
              <RuntimeRow
                label={t("dashboard.trafficTotal")}
                tooltip={
                  traffic && proxyUp ? (
                    <div className="flex flex-col gap-1.5 font-mono text-[11px]">
                      <span>
                        <span className="text-amber-300/90">{t("dashboard.egress")} </span>
                        {formatBytes(bytesTotalUp, byteUnits)}
                      </span>
                      <span>
                        <span className="text-teal-300">allow </span>
                        {formatBytes(traffic.bytes_total_up_allow ?? 0, byteUnits)}
                      </span>
                      <span>
                        <span className="text-red-300">deny </span>
                        {formatBytes(traffic.bytes_total_up_deny ?? 0, byteUnits)}
                      </span>
                      <span className="mt-1 border-t border-white/10 pt-1">
                        <span className="text-sky-300/90">{t("dashboard.ingress")} </span>
                        {formatBytes(bytesTotalDown, byteUnits)}
                      </span>
                      <span>
                        <span className="text-teal-300">allow </span>
                        {formatBytes(traffic.bytes_total_down_allow ?? 0, byteUnits)}
                      </span>
                      <span>
                        <span className="text-red-300">deny </span>
                        {formatBytes(traffic.bytes_total_down_deny ?? 0, byteUnits)}
                      </span>
                    </div>
                  ) : undefined
                }
                value={
                  proxyUp && traffic
                    ? formatBytes(bytesTotal, byteUnits)
                    : "—"
                }
              />
            </ul>
          )}
          <div className="mt-5 border-t border-white/8 pt-4">
            <p className="mb-3 font-mono text-[11px] text-zinc-600">
              {t("dashboard.latencyHint")}
            </p>
            {loading ? null : (
              <ul className="flex flex-col gap-3.5 text-[12.5px]">
                <RuntimeRow
                  label={t("dashboard.aclLatency")}
                  tooltip={
                    traffic && proxyUp && traffic.decide_breakdown_5m
                      ? (
                          <ACLDecideBreakdownTooltip
                            totalAvgUs={traffic.decide_duration_us_avg_5m}
                            totalP95Us={traffic.decide_duration_us_p95_5m}
                            totalP99Us={traffic.decide_duration_us_p99_5m}
                            breakdown={traffic.decide_breakdown_5m}
                          />
                        )
                      : undefined
                  }
                  value={
                    traffic && proxyUp
                      ? `${formatDurationUs(traffic.decide_duration_us_avg_5m, emDash, durationUnits)} / ${formatDurationUs(traffic.decide_duration_us_p95_5m, emDash, durationUnits)} / ${formatDurationUs(traffic.decide_duration_us_p99_5m, emDash, durationUnits)}`
                      : "—"
                  }
                />
                <RuntimeRow
                  label={t("dashboard.inspectLatency")}
                  tooltip={
                    traffic && proxyUp && traffic.inspect_breakdown_5m
                      ? (
                          <InspectBreakdownTooltip
                            totalAvgUs={traffic.inspect_duration_us_avg_5m}
                            totalP95Us={traffic.inspect_duration_us_p95_5m}
                            totalP99Us={traffic.inspect_duration_us_p99_5m}
                            breakdown={traffic.inspect_breakdown_5m}
                          />
                        )
                      : undefined
                  }
                  value={
                    traffic && proxyUp
                      ? `${formatDurationUs(traffic.inspect_duration_us_avg_5m, emDash, durationUnits)} / ${formatDurationUs(traffic.inspect_duration_us_p95_5m, emDash, durationUnits)} / ${formatDurationUs(traffic.inspect_duration_us_p99_5m, emDash, durationUnits)}`
                      : "—"
                  }
                />
                <RuntimeRow
                  label={t("dashboard.policyLatency")}
                  tooltip={
                    traffic && proxyUp
                      ? (
                          <PolicyBreakdownTooltip
                            totalAvgUs={traffic.policy_duration_us_avg_5m}
                            totalP95Us={traffic.policy_duration_us_p95_5m}
                            totalP99Us={traffic.policy_duration_us_p99_5m}
                            aclAvgUs={traffic.decide_duration_us_avg_5m}
                            aclP95Us={traffic.decide_duration_us_p95_5m}
                            aclP99Us={traffic.decide_duration_us_p99_5m}
                            inspectAvgUs={traffic.inspect_duration_us_avg_5m}
                            inspectP95Us={traffic.inspect_duration_us_p95_5m}
                            inspectP99Us={traffic.inspect_duration_us_p99_5m}
                            aclBreakdown={traffic.decide_breakdown_5m}
                            inspectBreakdown={traffic.inspect_breakdown_5m}
                          />
                        )
                      : undefined
                  }
                  value={
                    traffic && proxyUp
                      ? `${formatDurationUs(traffic.policy_duration_us_avg_5m, emDash, durationUnits)} / ${formatDurationUs(traffic.policy_duration_us_p95_5m, emDash, durationUnits)} / ${formatDurationUs(traffic.policy_duration_us_p99_5m, emDash, durationUnits)}`
                      : "—"
                  }
                />
              </ul>
            )}
          </div>
          <div className="mt-5 border-t border-white/8 pt-4">
            <Link
              href="/manage/access-log"
              className="text-[12px] text-zinc-500 transition hover:text-teal-300"
            >
              {t("accessLog.linkToLog")}
            </Link>
          </div>
        </div>
      </div>
    </div>
  );
}

function RuntimeRow({
  label,
  value,
  tooltip,
}: {
  label: string;
  value: ReactNode;
  tooltip?: ReactNode;
}) {
  const valueNode = (
    <span
      className={`font-mono text-zinc-200 ${tooltip ? "cursor-help border-b border-dotted border-zinc-600" : ""}`}
    >
      {value}
    </span>
  );
  return (
    <li className="flex flex-row items-center justify-between gap-3">
      <span className="text-zinc-500">{label}</span>
      {tooltip ? (
        <Tooltip
          title={tooltip}
          placement="leftTop"
          styles={{ root: { maxWidth: 320 } }}
        >
          {valueNode}
        </Tooltip>
      ) : (
        valueNode
      )}
    </li>
  );
}
