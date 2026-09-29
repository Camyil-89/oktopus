"use client";

import { formatDurationUs } from "@/utils/formatDurationUs";

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

type Props = {
  label: string;
  avgUs: number;
  p95Us: number;
  totalAvgUs: number;
};

export function BreakdownTooltipRow({ label, avgUs, p95Us, totalAvgUs }: Props) {
  return (
    <div className="flex min-w-0 flex-col gap-0.5 text-[11px] leading-snug">
      <span className="text-zinc-400">{label}</span>
      <span className="wrap-break-word text-right font-mono text-zinc-200">
        {formatDurationUs(avgUs)}{" "}
        <span className="text-zinc-500">
          ({formatSharePct(avgUs, totalAvgUs)})
        </span>
        <span className="text-zinc-600"> · p95 </span>
        {formatDurationUs(p95Us)}
      </span>
    </div>
  );
}

export const breakdownTooltipShellClassName =
  "box-border flex max-h-[min(70vh,22rem)] w-[280px] max-w-[calc(100vw-3rem)] flex-col gap-2 overflow-x-hidden overflow-y-auto py-0.5";
