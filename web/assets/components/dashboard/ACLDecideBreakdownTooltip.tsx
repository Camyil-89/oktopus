"use client";

import {
  BreakdownTooltipRow,
  breakdownTooltipShellClassName,
} from "@/assets/components/dashboard/BreakdownTooltipRow";
import type { ProxyDecideBreakdown5m } from "@/types/proxy";
import { formatDurationUs } from "@/utils/formatDurationUs";

type Props = {
  totalAvgUs: number;
  totalP95Us: number;
  totalP99Us?: number;
  breakdown: ProxyDecideBreakdown5m;
};

export function ACLDecideBreakdownTooltip({
  totalAvgUs,
  totalP95Us,
  totalP99Us,
  breakdown,
}: Props) {
  const engineSubtotal =
    breakdown.scope_us_avg_5m +
    breakdown.match_sni_us_avg_5m +
    breakdown.match_src_ip_us_avg_5m +
    breakdown.match_dst_ip_us_avg_5m +
    breakdown.match_slow_us_avg_5m;

  return (
    <div className={breakdownTooltipShellClassName}>
      <p className="m-0 text-[11px] leading-snug text-zinc-400">
        ACL за 5 мин: подготовка полей и сопоставление правил. Запись в журнал и
        инспекция не входят.
      </p>
      <div className="flex flex-col gap-1.5 border-t border-white/10 pt-2">
        <BreakdownTooltipRow
          label="Итого"
          avgUs={totalAvgUs}
          p95Us={totalP95Us}
          totalAvgUs={totalAvgUs}
        />
        {totalP99Us != null ? (
          <p className="m-0 text-right font-mono text-[11px] text-zinc-500">
            p99 {formatDurationUs(totalP99Us)}
          </p>
        ) : null}
        <BreakdownTooltipRow
          label="Подготовка (SNI, path, IP)"
          avgUs={breakdown.prepare_us_avg_5m}
          p95Us={breakdown.prepare_us_p95_5m}
          totalAvgUs={totalAvgUs}
        />
        <BreakdownTooltipRow
          label="Engine"
          avgUs={breakdown.engine_us_avg_5m}
          p95Us={breakdown.engine_us_p95_5m}
          totalAvgUs={totalAvgUs}
        />
      </div>
      <div className="flex flex-col gap-1.5 border-t border-white/10 pt-2">
        <p className="m-0 text-[10px] uppercase tracking-wide text-zinc-500">
          внутри engine
        </p>
        <BreakdownTooltipRow
          label="Scope (блоки GROUP/ALL)"
          avgUs={breakdown.scope_us_avg_5m}
          p95Us={breakdown.scope_us_p95_5m}
          totalAvgUs={engineSubtotal > 0 ? engineSubtotal : totalAvgUs}
        />
        <BreakdownTooltipRow
          label="SNI index"
          avgUs={breakdown.match_sni_us_avg_5m}
          p95Us={breakdown.match_sni_us_p95_5m}
          totalAvgUs={engineSubtotal > 0 ? engineSubtotal : totalAvgUs}
        />
        <BreakdownTooltipRow
          label="SRC IP index"
          avgUs={breakdown.match_src_ip_us_avg_5m}
          p95Us={breakdown.match_src_ip_us_p95_5m}
          totalAvgUs={engineSubtotal > 0 ? engineSubtotal : totalAvgUs}
        />
        <BreakdownTooltipRow
          label="DST IP index"
          avgUs={breakdown.match_dst_ip_us_avg_5m}
          p95Us={breakdown.match_dst_ip_us_p95_5m}
          totalAvgUs={engineSubtotal > 0 ? engineSubtotal : totalAvgUs}
        />
        <BreakdownTooltipRow
          label="Slow rules (regex)"
          avgUs={breakdown.match_slow_us_avg_5m}
          p95Us={breakdown.match_slow_us_p95_5m}
          totalAvgUs={engineSubtotal > 0 ? engineSubtotal : totalAvgUs}
        />
      </div>
    </div>
  );
}
