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
import { formatBytes } from "@/utils/formatBytes";
import { formatDurationUs } from "@/utils/formatDurationUs";
import { OktopusLoading } from "@/assets/components/oktopus/OktopusLoading";
import { Alert, Tag, Tooltip, Typography } from "antd";
import Link from "next/link";
import { useCallback, useEffect, useState, type ReactNode } from "react";

function aclBuildLabel(st: ProxyRuntimeStatus["acl"]): string {
  switch (st.build_status) {
    case "building":
      return "СБОРКА";
    case "error":
      return "ОШИБКА";
    case "ready":
      return "OK";
    default:
      return "ОЖИДАНИЕ";
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

function formatInt(n: number): string {
  return new Intl.NumberFormat("ru-RU").format(n);
}

function formatStartupClock(iso?: string): string {
  if (!iso) return "—";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "—";
  return d.toLocaleString("ru-RU", {
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

function formatDurationMs(ms?: number): string {
  if (ms == null || ms < 0) return "—";
  if (ms < 1000) return `${ms} мс`;
  const sec = ms / 1000;
  if (sec < 60) return `${sec.toFixed(sec < 10 ? 1 : 0)} с`;
  const min = Math.floor(sec / 60);
  const rest = Math.round(sec % 60);
  return rest > 0 ? `${min} мин ${rest} с` : `${min} мин`;
}

/** Задержка от старта serve до события (прокси / ACL). */
function formatDelayFromServeStart(ms?: number): string {
  if (ms == null || ms < 0) return "ещё не произошло";
  return `через ${formatDurationMs(ms)} после старта serve`;
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

function MetricsWindow5mBadge() {
  return (
    <span className="font-mono text-[11px] text-teal-300/70">5m окно</span>
  );
}

export default function ManagePage() {
  const [status, setStatus] = useState<ProxyRuntimeStatus | null>(null);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    try {
      const st = await proxyApi.getProxyRuntimeStatus();
      setStatus(st);
      setError(null);
    } catch (e) {
      setError(
        e instanceof ApiError ? e.message : "Не удалось загрузить статус",
      );
    }
  }, []);

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
          message="Прокси не запущен"
          description={status.proxy_start_error}
        />
      ) : null}

      <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
        <DashboardStatCard
          label="Прокси"
          loading={loading}
          tone={proxyUp ? "success" : "danger"}
          badge={<StatusDot tone={proxyUp ? "success" : "danger"} />}
          value={proxyUp ? "RUNNING" : "STOPPED"}
          hint={status?.listen || "—"}
        />
        <DashboardStatCard
          label="Сборка ACL"
          loading={loading && !acl}
          tone={acl ? aclBuildTone(acl) : undefined}
          badge={
            acl ? <StatusDot tone={aclBuildTone(acl)} /> : undefined
          }
          value={acl ? aclBuildLabel(acl) : "—"}
          hint={
            acl
              ? `${acl.active_logical_rules} правил · ${acl.active_patterns} паттернов`
              : undefined
          }
        />
        <DashboardStatCard
          label="За 1 мин"
          loading={loading}
          badge={
            traffic && proxyUp ? (
              <span className="font-mono text-[11px] text-zinc-500">скольз.</span>
            ) : undefined
          }
          value={
            proxyUp ? (
              <span className="font-mono">{formatInt(reqPerMin)}</span>
            ) : (
              "—"
            )
          }
          hint={
            traffic && proxyUp
              ? `за 5 с: ${formatInt(reqLast5s)} · средн. 5 мин: ${formatInt(reqPerMinAvg5m)}/мин`
              : undefined
          }
        />
        <DashboardStatCard
          label="Allow / Deny"
          loading={loading}
          badge={traffic ? <MetricsWindow5mBadge /> : undefined}
          value={
            traffic ? (
              <span className="font-mono text-xl">
                <span className="text-teal-300">{formatInt(traffic.allowed_5m)}</span>
                <span className="mx-1.5 text-zinc-600">/</span>
                <span className="text-red-300/90">
                  {formatInt(traffic.denied_5m)}
                </span>
              </span>
            ) : (
              "—"
            )
          }
          hint={
            traffic
              ? `сумма за 5 мин · ${formatSharePct(traffic.allowed_5m, totalDecisions)} allow · ${formatSharePct(traffic.denied_5m, totalDecisions)} deny`
              : undefined
          }
        />
      </div>

      {acl?.build_error ? (
        <Alert type="error" message={acl.build_error} showIcon />
      ) : null}

      <div className="rounded-xl border border-white/10 bg-white/5 p-5">
        <p className="text-[13px] font-medium text-zinc-100">Запуск процесса</p>
        <p className="mb-4 mt-0.5 font-mono text-[11.5px] text-zinc-500">
          отсчёт от запуска serve · в карточках ниже — задержка до события
        </p>
        {loading ? (
          <div className="flex justify-center py-6">
            <OktopusLoading size="lg" />
          </div>
        ) : (
          <ul className="grid gap-3 sm:grid-cols-3">
            <li className="flex flex-col gap-1 rounded-lg border border-white/8 bg-black/20 px-4 py-3">
              <span className="text-[11px] text-zinc-500">
                Старт serve
              </span>
              <span className="font-mono text-[13px] text-zinc-100">
                {formatStartupClock(startup?.started_at)}
              </span>
            </li>
            <li className="flex flex-col gap-1.5 rounded-lg border border-white/8 bg-black/20 px-4 py-3">
              <span className="text-[11px] text-zinc-500">
                Прокси начал принимать запросы
              </span>
              <span className="text-[13px] leading-snug text-teal-200/90">
                {formatDelayFromServeStart(startup?.proxy_ready_ms)}
              </span>
              <span className="font-mono text-[11px] text-zinc-600">
                момент: {formatStartupClock(startup?.proxy_ready_at)}
              </span>
            </li>
            <li className="flex flex-col gap-1.5 rounded-lg border border-white/8 bg-black/20 px-4 py-3">
              <span className="text-[11px] text-zinc-500">
                Компиляция ACL
              </span>
              <span className="text-[13px] leading-snug text-teal-200/90">
                {acl?.build_status === "building"
                  ? "в процессе…"
                  : formatDurationMs(aclCompileDurationMs(acl))}
              </span>
              <span className="font-mono text-[11px] text-zinc-600">
                {acl?.build_finished_at
                  ? `готово: ${formatStartupClock(acl.build_finished_at)}`
                  : "ещё не выполнялась"}
              </span>
            </li>
          </ul>
        )}
      </div>

      <div className="grid gap-3 lg:grid-cols-3">
        <div className="rounded-xl border border-white/10 bg-white/[0.05] p-5 lg:col-span-2">
          <div className="mb-6 flex flex-row flex-wrap items-center justify-between gap-3">
            <div>
              <p className="text-[13px] font-medium text-zinc-100">Трафик</p>
              <p className="mt-0.5 font-mono text-[11.5px] text-zinc-500">
                запросы за последние 5 мин · интервал 10 с
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
              Прокси не запущен — график появится после старта.
            </Typography.Text>
          ) : (
            <TrafficAllowDenyChart buckets={buckets} loading={loading} />
          )}
          <div className="mt-8 border-t border-white/8 pt-6">
            <div className="mb-6 flex flex-row flex-wrap items-center justify-between gap-3">
              <div>
                <p className="text-[13px] font-medium text-zinc-100">
                  Пропускная способность
                </p>
                <p className="mt-0.5 font-mono text-[11.5px] text-zinc-500">
                  исх / вх за 5 мин · интервал 10 с
                </p>
              </div>
              <div className="flex flex-row items-center gap-3 font-mono text-[11px]">
                <span className="flex flex-row items-center gap-1.5 text-zinc-500">
                  <span className="h-2 w-2 rounded-sm bg-amber-400/70" />
                  исх
                </span>
                <span className="flex flex-row items-center gap-1.5 text-zinc-500">
                  <span className="h-2 w-2 rounded-sm bg-sky-400/70" />
                  вх
                </span>
              </div>
            </div>
            {!proxyUp && !loading ? (
              <Typography.Text type="secondary">
                Прокси не запущен — график появится после старта.
              </Typography.Text>
            ) : (
              <TrafficThroughputChart buckets={buckets} loading={loading} />
            )}
          </div>
        </div>

        <div className="rounded-xl border border-white/10 bg-white/[0.05] p-5">
          <p className="text-[13px] font-medium text-zinc-100">Runtime</p>
          <p className="mb-5 mt-0.5 font-mono text-[11.5px] text-zinc-500">
            нагрузка и ACL
          </p>
          {loading ? (
            <div className="flex justify-center py-8">
              <OktopusLoading size="lg" />
            </div>
          ) : (
            <ul className="flex flex-col gap-3.5 text-[12.5px]">
              <RuntimeRow
                label="соединения"
                value={
                  proxyUp && traffic
                    ? String(traffic.active_connections)
                    : "—"
                }
              />
              <RuntimeRow
                label="WebSocket (MITM)"
                value={
                  proxyUp && traffic
                    ? String(traffic.active_websocket_connections ?? 0)
                    : "—"
                }
              />
              <RuntimeRow
                label="правила в прокси"
                value={
                  acl ? (
                    acl.rules_in_sync ? (
                      <Tag color="success" className="!m-0">
                        актуальные
                      </Tag>
                    ) : (
                      <Tag color="warning" className="!m-0">
                        устарели
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
                        При компиляции каждая строка SNI проверяется на fast
                        index (exact / suffix / *.domain). Остальное — regexp
                        (проверяется только в slow-правилах).
                      </p>
                      {acl.sni_regexp_reason_counts &&
                      Object.keys(acl.sni_regexp_reason_counts).length > 0 ? (
                        <div className="border-t border-white/10 pt-2 text-zinc-500">
                          <p className="m-0 mb-1 text-[10px] uppercase tracking-wide">
                            причины regexp
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
                            примеры строк
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
                label="пользователи / источники"
                value={
                  traffic && proxyUp
                    ? `${traffic.unique_users_5m} / ${traffic.unique_sources_5m}`
                    : "—"
                }
              />
              <RuntimeRow
                label="очередь access log"
                value={
                  traffic ? String(traffic.access_log_queue_pending) : "—"
                }
              />
              <RuntimeRow
                label="трафик всего"
                tooltip={
                  traffic && proxyUp ? (
                    <div className="flex flex-col gap-1.5 font-mono text-[11px]">
                      <span>
                        <span className="text-amber-300/90">исх </span>
                        {formatBytes(bytesTotalUp)}
                      </span>
                      <span>
                        <span className="text-teal-300">allow </span>
                        {formatBytes(traffic.bytes_total_up_allow ?? 0)}
                      </span>
                      <span>
                        <span className="text-red-300">deny </span>
                        {formatBytes(traffic.bytes_total_up_deny ?? 0)}
                      </span>
                      <span className="mt-1 border-t border-white/10 pt-1">
                        <span className="text-sky-300/90">вх </span>
                        {formatBytes(bytesTotalDown)}
                      </span>
                      <span>
                        <span className="text-teal-300">allow </span>
                        {formatBytes(traffic.bytes_total_down_allow ?? 0)}
                      </span>
                      <span>
                        <span className="text-red-300">deny </span>
                        {formatBytes(traffic.bytes_total_down_deny ?? 0)}
                      </span>
                    </div>
                  ) : undefined
                }
                value={
                  proxyUp && traffic
                    ? formatBytes(bytesTotal)
                    : "—"
                }
              />
            </ul>
          )}
          <div className="mt-5 border-t border-white/8 pt-4">
            <p className="mb-3 font-mono text-[11px] text-zinc-600">
              avg / p95 / p99 · последние 1000 решений
            </p>
            {loading ? null : (
              <ul className="flex flex-col gap-3.5 text-[12.5px]">
                <RuntimeRow
                  label="ACL avg / p95 / p99"
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
                      ? `${formatDurationUs(traffic.decide_duration_us_avg_5m)} / ${formatDurationUs(traffic.decide_duration_us_p95_5m)} / ${formatDurationUs(traffic.decide_duration_us_p99_5m)}`
                      : "—"
                  }
                />
                <RuntimeRow
                  label="Inspect avg / p95 / p99"
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
                      ? `${formatDurationUs(traffic.inspect_duration_us_avg_5m)} / ${formatDurationUs(traffic.inspect_duration_us_p95_5m)} / ${formatDurationUs(traffic.inspect_duration_us_p99_5m)}`
                      : "—"
                  }
                />
                <RuntimeRow
                  label="Политика avg / p95 / p99"
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
                      ? `${formatDurationUs(traffic.policy_duration_us_avg_5m)} / ${formatDurationUs(traffic.policy_duration_us_p95_5m)} / ${formatDurationUs(traffic.policy_duration_us_p99_5m)}`
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
              Журнал доступа →
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
