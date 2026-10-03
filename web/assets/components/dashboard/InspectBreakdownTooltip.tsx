"use client";

import {
  BreakdownTooltipRow,
  breakdownTooltipShellClassName,
} from "@/assets/components/dashboard/BreakdownTooltipRow";
import { useTranslation } from "@/contexts/LocaleContext";
import type { ProxyInspectBreakdown5m } from "@/types/proxy";
import { durationUnitLabelsFromT, formatDurationUs } from "@/utils/formatDurationUs";

type Props = {
  totalAvgUs: number;
  totalP95Us: number;
  totalP99Us?: number;
  breakdown: ProxyInspectBreakdown5m;
};

export function InspectBreakdownTooltip({
  totalAvgUs,
  totalP95Us,
  totalP99Us,
  breakdown,
}: Props) {
  const { t } = useTranslation();
  const units = durationUnitLabelsFromT(t);
  return (
    <div className={breakdownTooltipShellClassName}>
      <p className="m-0 text-[11px] text-zinc-400">
        {t("dashboard.inspectBreakdownIntro")}
      </p>
      <div className="flex flex-col gap-1.5 border-t border-white/10 pt-2">
        <BreakdownTooltipRow
          label={t("dashboard.inspectBreakdownTotal")}
          avgUs={totalAvgUs}
          p95Us={totalP95Us}
          totalAvgUs={totalAvgUs}
        />
        {totalP99Us != null ? (
          <p className="m-0 text-right font-mono text-[11px] text-zinc-500">
            p99 {formatDurationUs(totalP99Us, undefined, units)}
          </p>
        ) : null}
        <BreakdownTooltipRow
          label={t("dashboard.inspectBreakdownCtx")}
          avgUs={breakdown.prepare_us_avg_5m}
          p95Us={breakdown.prepare_us_p95_5m}
          totalAvgUs={totalAvgUs}
        />
        <BreakdownTooltipRow
          label={t("dashboard.inspectBreakdownBody")}
          avgUs={breakdown.body_us_avg_5m}
          p95Us={breakdown.body_us_p95_5m}
          totalAvgUs={totalAvgUs}
        />
        <BreakdownTooltipRow
          label="Lua Eval"
          avgUs={breakdown.eval_us_avg_5m}
          p95Us={breakdown.eval_us_p95_5m}
          totalAvgUs={totalAvgUs}
        />
      </div>
    </div>
  );
}
