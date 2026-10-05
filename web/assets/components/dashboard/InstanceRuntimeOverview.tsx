"use client";

import { DashboardStatCard } from "@/assets/components/dashboard/DashboardStatCard";
import type { DashboardStatTone } from "@/assets/components/dashboard/DashboardStatCard";
import { TrafficAllowDenyChart } from "@/assets/components/dashboard/TrafficAllowDenyChart";
import { TrafficThroughputChart } from "@/assets/components/dashboard/TrafficThroughputChart";
import type {
  ProxyACLCompileStatus,
  ProxyInstanceRuntimeStatus,
} from "@/types/proxy";
import { useTranslation } from "@/contexts/LocaleContext";
import type { TranslateFn } from "@/i18n/translate";
import { Typography } from "antd";

function aclBuildLabel(st: ProxyACLCompileStatus, t: TranslateFn): string {
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

function aclBuildTone(st: ProxyACLCompileStatus): DashboardStatTone {
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

function formatInt(n: number, localeTag: string): string {
  return new Intl.NumberFormat(localeTag).format(n);
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

type InstanceRuntimeOverviewProps = {
  inst: ProxyInstanceRuntimeStatus;
  loading?: boolean;
  instanceName?: string;
  /** На главной в списке fleet графики уже есть сверху — не дублировать. */
  showCharts?: boolean;
};

export function InstanceRuntimeOverview({
  inst,
  loading = false,
  instanceName,
  showCharts = true,
}: InstanceRuntimeOverviewProps) {
  const { t, localeTag } = useTranslation();
  const traffic = inst.traffic;
  const acl = inst.acl;
  const proxyUp = inst.active;
  const totalDecisions = (traffic?.allowed_5m ?? 0) + (traffic?.denied_5m ?? 0);
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

  return (
    <div className="flex flex-col gap-3">
      <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
        <DashboardStatCard
          label={t("dashboard.proxy") + (instanceName ? `: ${instanceName}` : "")}
          loading={loading}
          tone={proxyUp ? "success" : "danger"}
          badge={
            <span
              className={`h-1.5 w-1.5 rounded-full ${proxyUp ? "bg-teal-400" : "bg-red-400"}`}
            />
          }
          value={proxyUp ? "RUNNING" : "STOPPED"}
          hint={inst.listen}
        />
        <DashboardStatCard
          label={t("dashboard.aclBuild")}
          loading={loading}
          tone={aclBuildTone(acl)}
          value={aclBuildLabel(acl, t)}
          hint={t("dashboard.rulesPatterns", {
            rules: acl.active_logical_rules,
            patterns: acl.active_patterns,
          })}
        />
        <DashboardStatCard
          label={t("dashboard.perMinute")}
          loading={loading}
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

      {inst.proxy_start_error ? (
        <Typography.Text type="danger">{inst.proxy_start_error}</Typography.Text>
      ) : null}

      {acl.build_error ? (
        <Typography.Text type="danger">{acl.build_error}</Typography.Text>
      ) : null}

      {showCharts ? (
        <div className="grid gap-3 lg:grid-cols-2">
          <div className="rounded-xl border border-white/10 bg-white/[0.05] p-5">
            <p className="text-[13px] font-medium text-zinc-100">{t("dashboard.traffic")}</p>
            {!proxyUp && !loading ? (
              <Typography.Text type="secondary">
                {t("dashboard.chartWhenUp")}
              </Typography.Text>
            ) : (
              <TrafficAllowDenyChart buckets={buckets} loading={loading} />
            )}
          </div>
          <div className="rounded-xl border border-white/10 bg-white/[0.05] p-5">
            <p className="text-[13px] font-medium text-zinc-100">
              {t("dashboard.throughput")}
            </p>
            {!proxyUp && !loading ? (
              <Typography.Text type="secondary">
                {t("dashboard.chartWhenUp")}
              </Typography.Text>
            ) : (
              <TrafficThroughputChart buckets={buckets} loading={loading} />
            )}
          </div>
        </div>
      ) : null}
    </div>
  );
}
