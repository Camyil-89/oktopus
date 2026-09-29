import type { ReactNode } from "react";
import { SkeletonLoader } from "@/assets/components/SkeletonLoader";

export type DashboardStatTone = "success" | "warning" | "danger" | "neutral";

type DashboardStatCardProps = {
  label: string;
  value: ReactNode;
  hint?: ReactNode;
  badge?: ReactNode;
  loading?: boolean;
  tone?: DashboardStatTone;
};

const toneCardClass: Record<DashboardStatTone, string> = {
  success: "border-teal-400/25 bg-teal-400/[0.06]",
  warning: "border-amber-400/25 bg-amber-400/[0.06]",
  danger: "border-red-400/25 bg-red-400/[0.06]",
  neutral: "border-white/10 bg-white/[0.05]",
};

const toneValueClass: Record<DashboardStatTone, string> = {
  success: "text-teal-300",
  warning: "text-amber-300",
  danger: "text-red-300/90",
  neutral: "text-zinc-400",
};

export function DashboardStatCard({
  label,
  value,
  hint,
  badge,
  loading,
  tone,
}: DashboardStatCardProps) {
  const cardTone = tone ?? "neutral";
  const cardClass = tone ? toneCardClass[cardTone] : toneCardClass.neutral;
  const valueClass = tone
    ? toneValueClass[cardTone]
    : "text-zinc-50";

  return (
    <div className={`rounded-xl border p-5 ${cardClass}`}>
      <div className="mb-4 flex items-center justify-between">
        <p className="font-mono text-[11px] uppercase tracking-[0.14em] text-zinc-400">
          {label}
        </p>
        {badge}
      </div>
      {loading ? (
        <SkeletonLoader className="min-h-8" loaderSize="sm">
          <div className="h-8 w-24 animate-pulse rounded bg-white/10" />
        </SkeletonLoader>
      ) : (
        <>
          <div
            className={`text-2xl font-semibold tracking-tight ${valueClass}`}
          >
            {value}
          </div>
          {hint ? (
            <p className="mt-1.5 font-mono text-[12px] text-zinc-500">{hint}</p>
          ) : null}
        </>
      )}
    </div>
  );
}
