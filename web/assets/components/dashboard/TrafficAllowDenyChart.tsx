"use client";

import type { ProxyTrafficBucket } from "@/types/proxy";
import { SkeletonLoader } from "@/assets/components/SkeletonLoader";
import { Tooltip } from "antd";
import { useMemo } from "react";

type Props = {
  buckets: ProxyTrafficBucket[];
  loading?: boolean;
};

function chartYMax(requests: number[]): number {
  const peak = requests.length ? Math.max(...requests) : 0;
  if (peak <= 0) {
    return 1;
  }
  const exp = 10 ** Math.floor(Math.log10(peak));
  const n = peak / exp;
  let nice = 10;
  if (n <= 1) {
    nice = 1;
  } else if (n <= 2) {
    nice = 2;
  } else if (n <= 5) {
    nice = 5;
  }
  return nice * exp;
}

function formatTick(value: number): string {
  if (value >= 1_000_000) {
    return `${(value / 1_000_000).toFixed(value >= 10_000_000 ? 0 : 1)}M`;
  }
  if (value >= 10_000) {
    return `${Math.round(value / 1000)}k`;
  }
  if (value >= 1000) {
    return `${(value / 1000).toFixed(value >= 5000 ? 0 : 1)}k`;
  }
  return String(Math.round(value));
}

function formatCount(n: number): string {
  return new Intl.NumberFormat("ru-RU").format(n);
}

function formatBucketRange(t: number): string {
  const start = new Date(t * 1000);
  const end = new Date((t + 10) * 1000);
  const fmt = (d: Date) =>
    d.toLocaleTimeString("ru-RU", {
      hour: "2-digit",
      minute: "2-digit",
      second: "2-digit",
    });
  return `${fmt(start)} – ${fmt(end)}`;
}

function sharePct(part: number, total: number): string {
  if (total <= 0) {
    return "0%";
  }
  const pct = (part / total) * 100;
  return pct < 10 ? `${pct.toFixed(1)}%` : `${Math.round(pct)}%`;
}

function BucketTooltipContent({ b }: { b: ProxyTrafficBucket }) {
  return (
    <div className="flex flex-col gap-1.5 font-mono text-[11px] leading-snug">
      <span className="text-zinc-300">{formatBucketRange(b.t)}</span>
      <span>
        <span className="text-zinc-500">всего </span>
        <span className="text-zinc-100">{formatCount(b.requests)}</span>
      </span>
      <span>
        <span className="text-teal-300">allow </span>
        <span className="text-zinc-100">
          {formatCount(b.allow)}
        </span>
        <span className="text-zinc-500">
          {" "}
          ({sharePct(b.allow, b.requests)})
        </span>
      </span>
      <span>
        <span className="text-red-300">deny </span>
        <span className="text-zinc-100">{formatCount(b.deny)}</span>
        <span className="text-zinc-500">
          {" "}
          ({sharePct(b.deny, b.requests)})
        </span>
      </span>
    </div>
  );
}

export function TrafficAllowDenyChart({ buckets, loading }: Props) {
  const data = buckets?.length ? buckets : [];

  const yMax = useMemo(
    () => chartYMax(data.map((b) => b.requests)),
    [data],
  );

  const yTicks = useMemo(() => {
    const mid = Math.round(yMax / 2);
    const ticks: number[] = [yMax];
    if (mid > 0 && mid < yMax) {
      ticks.push(mid);
    }
    ticks.push(0);
    return ticks;
  }, [yMax]);

  const chartSkeleton = (
    <div className="flex h-full items-end gap-[5px]">
      {Array.from({ length: 30 }).map((_, i) => (
        <div
          key={i}
          className="h-[30%] flex-1 animate-pulse rounded-sm bg-white/5"
        />
      ))}
    </div>
  );

  const chartBody = loading ? (
    <SkeletonLoader className="h-40 min-w-0 flex-1" loaderSize="md">
      {chartSkeleton}
    </SkeletonLoader>
  ) : (
    <div className="relative h-40 min-w-0 flex-1">
      <div className="pointer-events-none absolute inset-0">
        {yTicks.map((tick) => {
          const bottomPct = yMax > 0 ? (tick / yMax) * 100 : 0;
          const isEdge = tick === 0 || tick === yMax;
          return (
            <div
              key={tick}
              className={`absolute left-0 right-0 border-t ${isEdge ? "border-white/15" : "border-dashed border-white/10"}`}
              style={{ bottom: `${bottomPct}%` }}
            />
          );
        })}
      </div>
      <div className="relative flex h-full items-end gap-[5px]">
        {data.map((b) => {
              const totalH = (b.requests / yMax) * 100;
              const allowH =
                b.requests > 0 ? (b.allow / b.requests) * totalH : 0;
              const denyH =
                b.requests > 0 ? (b.deny / b.requests) * totalH : 0;
              return (
                <Tooltip
                  key={b.t}
                  title={<BucketTooltipContent b={b} />}
                  placement="top"
                  mouseEnterDelay={0.05}
                >
                  <div className="flex h-full min-w-0 flex-1 cursor-default flex-col justify-end gap-[2px] rounded-sm outline-none ring-teal-400/40 focus-visible:ring-2">
                    {allowH > 0 ? (
                      <div
                        className="traffic-bar-allow min-h-[2px] w-full rounded-sm"
                        style={{ height: `${allowH}%` }}
                      />
                    ) : null}
                    {denyH > 0 ? (
                      <div
                        className="traffic-bar-deny min-h-[2px] w-full rounded-sm"
                        style={{ height: `${denyH}%` }}
                      />
                    ) : null}
                    {b.requests === 0 ? (
                      <div className="h-[2px] w-full rounded-sm bg-white/5" />
                    ) : null}
                  </div>
                </Tooltip>
              );
            })}
      </div>
    </div>
  );

  return (
    <div className="flex flex-col gap-3">
      <div className="flex gap-2">
        <div
          className="flex h-40 w-10 shrink-0 flex-col justify-between py-px text-right font-mono text-[10px] leading-none text-zinc-500"
          aria-hidden
        >
          {loading
            ? [0, 1, 2].map((i) => (
                <span
                  key={i}
                  className="inline-block h-3 w-6 animate-pulse rounded bg-white/5"
                />
              ))
            : yTicks.map((tick) => (
                <span key={tick}>{formatTick(tick)}</span>
              ))}
        </div>
        {chartBody}
      </div>
      {!loading && data.length > 0 ? (
        <div className="flex justify-between pl-12 font-mono text-[10.5px] text-zinc-500">
          <span>−5м</span>
          <span className="text-zinc-600">5 мин · 10 с</span>
          <span>сейчас</span>
        </div>
      ) : null}
    </div>
  );
}
