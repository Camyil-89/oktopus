"use client";

import { useTranslation } from "@/contexts/LocaleContext";
import type {
  ProxyDecideBreakdown5m,
  ProxyInspectBreakdown5m,
} from "@/types/proxy";
import { durationUnitLabelsFromT, formatDurationUs } from "@/utils/formatDurationUs";

type Props = {
  totalAvgUs: number;
  totalP95Us: number;
  totalP99Us?: number;
  aclAvgUs: number;
  aclP95Us: number;
  aclP99Us?: number;
  inspectAvgUs: number;
  inspectP95Us: number;
  inspectP99Us?: number;
  aclBreakdown?: ProxyDecideBreakdown5m;
  inspectBreakdown?: ProxyInspectBreakdown5m;
};

export function PolicyBreakdownTooltip({
  totalAvgUs,
  totalP95Us,
  totalP99Us,
  aclAvgUs,
  aclP95Us,
  aclP99Us,
  inspectAvgUs,
  inspectP95Us,
  inspectP99Us,
}: Props) {
  const { t } = useTranslation();
  const units = durationUnitLabelsFromT(t);
  return (
    <div className="flex max-w-[320px] flex-col gap-2 py-0.5 text-[11px]">
      <div className="flex flex-col gap-1.5 font-mono text-zinc-200">
        <div className="flex justify-between gap-4">
          <span className="text-zinc-400">{t("dashboard.policyTotal")}</span>
          <span>
            {formatDurationUs(totalAvgUs, undefined, units)} · p95{" "}
            {formatDurationUs(totalP95Us, undefined, units)}
            {totalP99Us != null
              ? ` · p99 ${formatDurationUs(totalP99Us, undefined, units)}`
              : ""}
          </span>
        </div>
        <div className="flex justify-between gap-4">
          <span className="text-zinc-400">ACL</span>
          <span>
            {formatDurationUs(aclAvgUs, undefined, units)} · p95{" "}
            {formatDurationUs(aclP95Us, undefined, units)}
            {aclP99Us != null
              ? ` · p99 ${formatDurationUs(aclP99Us, undefined, units)}`
              : ""}
          </span>
        </div>
        <div className="flex justify-between gap-4">
          <span className="text-zinc-400">Inspect</span>
          <span>
            {formatDurationUs(inspectAvgUs, undefined, units)} · p95{" "}
            {formatDurationUs(inspectP95Us, undefined, units)}
            {inspectP99Us != null
              ? ` · p99 ${formatDurationUs(inspectP99Us, undefined, units)}`
              : ""}
          </span>
        </div>
      </div>
    </div>
  );
}
