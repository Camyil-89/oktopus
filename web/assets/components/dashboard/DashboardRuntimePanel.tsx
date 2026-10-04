"use client";

import { ACLDecideBreakdownTooltip } from "@/assets/components/dashboard/ACLDecideBreakdownTooltip";
import { InspectBreakdownTooltip } from "@/assets/components/dashboard/InspectBreakdownTooltip";
import { PolicyBreakdownTooltip } from "@/assets/components/dashboard/PolicyBreakdownTooltip";
import { OktopusLoading } from "@/assets/components/oktopus/OktopusLoading";
import { useTranslation } from "@/contexts/LocaleContext";
import type {
  ProxyACLCompileStatus,
  ProxyTrafficSnapshot,
} from "@/types/proxy";
import { byteUnitLabelsFromT, formatBytes } from "@/utils/formatBytes";
import { durationUnitLabelsFromT, formatDurationUs } from "@/utils/formatDurationUs";
import { Tag, Tooltip, Typography } from "antd";
import Link from "next/link";
import { useMemo, type ReactNode } from "react";

type DashboardRuntimePanelProps = {
  acl: ProxyACLCompileStatus | null;
  traffic: ProxyTrafficSnapshot | null;
  proxyUp: boolean;
  loading: boolean;
  /** Подпись под заголовком (например «среднее по инстансам»). */
  hintExtra?: string;
  showBreakdownTooltips?: boolean;
  showAccessLogLink?: boolean;
};

export function DashboardRuntimePanel({
  acl,
  traffic,
  proxyUp,
  loading,
  hintExtra,
  showBreakdownTooltips = true,
  showAccessLogLink = false,
}: DashboardRuntimePanelProps) {
  const { t } = useTranslation();
  const byteUnits = useMemo(() => byteUnitLabelsFromT(t), [t]);
  const durationUnits = useMemo(() => durationUnitLabelsFromT(t), [t]);
  const emDash = t("common.emDash");

  const bytesTotalUp =
    (traffic?.bytes_total_up_allow ?? 0) + (traffic?.bytes_total_up_deny ?? 0);
  const bytesTotalDown =
    (traffic?.bytes_total_down_allow ?? 0) +
    (traffic?.bytes_total_down_deny ?? 0);
  const bytesTotal = bytesTotalUp + bytesTotalDown;

  return (
    <div className="rounded-xl border border-white/10 bg-white/[0.05] p-5">
      <p className="text-[13px] font-medium text-zinc-100">
        {t("dashboard.runtime")}
      </p>
      <p className="mb-5 mt-0.5 font-mono text-[11.5px] text-zinc-500">
        {hintExtra ? `${t("dashboard.runtimeHint")} · ${hintExtra}` : t("dashboard.runtimeHint")}
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
                : emDash
            }
          />
          <RuntimeRow
            label={t("dashboard.wsMitm")}
            value={
              proxyUp && traffic
                ? String(traffic.active_websocket_connections ?? 0)
                : emDash
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
                emDash
              )
            }
          />
          <RuntimeRow
            label="SNI index / regexp"
            tooltip={
              showBreakdownTooltips && acl
                ? (
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
                )
                : undefined
            }
            value={
              acl
                ? `${acl.active_sni_patterns_indexed ?? emDash} / ${acl.active_sni_patterns_regexp ?? emDash} · slow ${acl.active_slow_rule_slots ?? emDash}`
                : emDash
            }
          />
          <RuntimeRow
            label={t("dashboard.usersSources")}
            value={
              traffic && proxyUp
                ? `${traffic.unique_users_5m} / ${traffic.unique_sources_5m}`
                : emDash
            }
          />
          <RuntimeRow
            label={t("dashboard.accessLogQueue")}
            value={traffic ? String(traffic.access_log_queue_pending) : emDash}
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
                : emDash
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
                showBreakdownTooltips &&
                  traffic &&
                  proxyUp &&
                  traffic.decide_breakdown_5m
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
                  : emDash
              }
            />
            <RuntimeRow
              label={t("dashboard.inspectLatency")}
              tooltip={
                showBreakdownTooltips &&
                  traffic &&
                  proxyUp &&
                  traffic.inspect_breakdown_5m
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
                  : emDash
              }
            />
            <RuntimeRow
              label={t("dashboard.policyLatency")}
              tooltip={
                showBreakdownTooltips && traffic && proxyUp
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
                  : emDash
              }
            />
          </ul>
        )}
      </div>
      {showAccessLogLink ? (
        <div className="mt-5 border-t border-white/8 pt-4">
          <Link
            href="/manage/access-log"
            className="text-[12px] text-zinc-500 transition hover:text-teal-300"
          >
            {t("accessLog.linkToLog")}
          </Link>
        </div>
      ) : null}
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
