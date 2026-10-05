"use client";

import { DashboardRuntimePanel } from "@/assets/components/dashboard/DashboardRuntimePanel";
import { DashboardTrafficChartsSection } from "@/assets/components/dashboard/DashboardTrafficChartsSection";
import { InstanceRuntimeOverview } from "@/assets/components/dashboard/InstanceRuntimeOverview";
import type { ProxyInstanceRuntimeStatus } from "@/types/proxy";

type Props = {
  inst: ProxyInstanceRuntimeStatus;
  loading?: boolean;
  instanceName?: string;
  showAccessLogLink?: boolean;
};

/** Карточки + графики (2/3) и Runtime (1/3), как на главной дашборда. */
export function InstanceRuntimeDashboardLayout({
  inst,
  loading = false,
  instanceName,
  showAccessLogLink = false,
}: Props) {
  const buckets = inst.traffic?.buckets_10s ?? [];

  return (
    <div className="flex flex-col gap-4">
      <InstanceRuntimeOverview
        inst={inst}
        loading={loading}
        instanceName={instanceName}
        showCharts={false}
      />
      <div className="grid gap-3 lg:grid-cols-3">
        <DashboardTrafficChartsSection
          buckets={buckets}
          proxyUp={inst.active}
          loading={loading}
        />
        <DashboardRuntimePanel
          acl={inst.acl}
          traffic={inst.traffic}
          proxyUp={inst.active}
          loading={loading}
          showBreakdownTooltips
          showAccessLogLink={showAccessLogLink}
        />
      </div>
    </div>
  );
}
