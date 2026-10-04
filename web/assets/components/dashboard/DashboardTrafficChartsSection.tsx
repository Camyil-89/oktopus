"use client";

import { TrafficAllowDenyChart } from "@/assets/components/dashboard/TrafficAllowDenyChart";
import { TrafficThroughputChart } from "@/assets/components/dashboard/TrafficThroughputChart";
import { useTranslation } from "@/contexts/LocaleContext";
import type { ProxyTrafficBucket } from "@/types/proxy";
import { Typography } from "antd";

type Props = {
  buckets: ProxyTrafficBucket[];
  proxyUp: boolean;
  loading?: boolean;
};

export function DashboardTrafficChartsSection({
  buckets,
  proxyUp,
  loading = false,
}: Props) {
  const { t } = useTranslation();

  return (
    <div className="rounded-xl border border-white/10 bg-white/[0.05] p-5 lg:col-span-2">
      <div className="mb-6 flex flex-row flex-wrap items-center justify-between gap-3">
        <div>
          <p className="text-[13px] font-medium text-zinc-100">
            {t("dashboard.traffic")}
          </p>
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
  );
}
