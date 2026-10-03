"use client";

import type { ProxyTrafficBucket } from "@/types/proxy";
import { SkeletonLoader } from "@/assets/components/SkeletonLoader";
import { Tooltip } from "antd";
import { useTranslation } from "@/contexts/LocaleContext";
import { byteUnitLabelsFromT } from "@/utils/formatBytes";
import { useMemo } from "react";

type Props = {
  buckets: ProxyTrafficBucket[];
  loading?: boolean;
};

const bucketSec = 10;

/** Потолок одной половины (исх или вх) от нуля. */
function chartYMax(peak: number): number {
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

function formatBytesPerSec(
  bps: number,
  units: string[],
): string {
  if (bps <= 0) {
    return "0";
  }
  let v = bps;
  let i = 0;
  while (v >= 1024 && i < units.length - 1) {
    v /= 1024;
    i++;
  }
  const digits = v >= 100 ? 0 : v >= 10 ? 1 : 2;
  return `${v.toFixed(digits)} ${units[i]}`;
}

function formatChartBytes(n: number, units: string[]): string {
  if (n <= 0) {
    return `0 ${units[0]}`;
  }
  let v = n;
  let i = 0;
  while (v >= 1024 && i < units.length - 1) {
    v /= 1024;
    i++;
  }
  const digits = v >= 100 ? 0 : v >= 10 ? 1 : 2;
  return `${v.toFixed(digits)} ${units[i]}`;
}

function formatBucketRange(t: number): string {
  const start = new Date(t * 1000);
  const end = new Date((t + bucketSec) * 1000);
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

function bucketBytesUp(b: ProxyTrafficBucket) {
  const allow = b.bytes_up_allow ?? 0;
  const deny = b.bytes_up_deny ?? 0;
  if (allow + deny > 0) {
    return { allow, deny, total: allow + deny };
  }
  const total = b.bytes_up ?? 0;
  return { allow: total, deny: 0, total };
}

function bucketBytesDown(b: ProxyTrafficBucket) {
  const allow = b.bytes_down_allow ?? 0;
  const deny = b.bytes_down_deny ?? 0;
  if (allow + deny > 0) {
    return { allow, deny, total: allow + deny };
  }
  const total = b.bytes_down ?? 0;
  return { allow: total, deny: 0, total };
}

function BucketTooltipContent({
  b,
  upBps,
  downBps,
  byteUnits,
  bpsUnits,
  egressLabel,
  ingressLabel,
}: {
  b: ProxyTrafficBucket;
  upBps: number;
  downBps: number;
  byteUnits: string[];
  bpsUnits: string[];
  egressLabel: string;
  ingressLabel: string;
}) {
  const up = bucketBytesUp(b);
  const down = bucketBytesDown(b);
  return (
    <div className="flex flex-col gap-1.5 font-mono text-[11px] leading-snug">
      <span className="text-zinc-300">{formatBucketRange(b.t)}</span>
      <span>
        <span className="text-amber-300/90">{egressLabel} </span>
        <span className="text-zinc-100">{formatChartBytes(up.total, byteUnits)}</span>
        <span className="text-zinc-500"> · {formatBytesPerSec(upBps, bpsUnits)}</span>
      </span>
      <span>
        <span className="text-teal-300">allow </span>
        <span className="text-zinc-100">{formatChartBytes(up.allow, byteUnits)}</span>
        <span className="text-zinc-500"> ({sharePct(up.allow, up.total)})</span>
      </span>
      <span>
        <span className="text-red-300">deny </span>
        <span className="text-zinc-100">{formatChartBytes(up.deny, byteUnits)}</span>
        <span className="text-zinc-500"> ({sharePct(up.deny, up.total)})</span>
      </span>
      <span>
        <span className="text-sky-300/90">{ingressLabel} </span>
        <span className="text-zinc-100">{formatChartBytes(down.total, byteUnits)}</span>
        <span className="text-zinc-500"> · {formatBytesPerSec(downBps, bpsUnits)}</span>
      </span>
      <span>
        <span className="text-teal-300">allow </span>
        <span className="text-zinc-100">{formatChartBytes(down.allow, byteUnits)}</span>
        <span className="text-zinc-500">
          {" "}
          ({sharePct(down.allow, down.total)})
        </span>
      </span>
      <span>
        <span className="text-red-300">deny </span>
        <span className="text-zinc-100">{formatChartBytes(down.deny, byteUnits)}</span>
        <span className="text-zinc-500">
          {" "}
          ({sharePct(down.deny, down.total)})
        </span>
      </span>
    </div>
  );
}

/** Доли allow/deny внутри половины столбца (высоты в % от половины графика). */
function stackedHalfHeights(
  totalRate: number,
  allowBytes: number,
  denyBytes: number,
  yMax: number,
): { allowH: number; denyH: number } {
  const totalH = rateInHalfPct(totalRate, yMax);
  const sum = allowBytes + denyBytes;
  if (totalH <= 0 || sum <= 0) {
    return { allowH: 0, denyH: 0 };
  }
  const allowH = (allowBytes / sum) * totalH;
  const denyH = (denyBytes / sum) * totalH;
  return { allowH, denyH };
}

/** Скругление у внешнего края половины столбца (к нулю — без скругления). */
function barRadiusUpHalf(allowH: number, denyH: number, kind: "allow" | "deny"): string {
  const outer = denyH > 0 ? "deny" : "allow";
  if (kind === outer) {
    return "!rounded-b-none !rounded-t-sm";
  }
  return "!rounded-none";
}

function barRadiusDownHalf(allowH: number, denyH: number, kind: "allow" | "deny"): string {
  const outer = denyH > 0 ? "deny" : "allow";
  if (kind === outer) {
    return "!rounded-t-none !rounded-b-sm";
  }
  return "!rounded-none";
}

/** Высота столбца внутри верхней/нижней половины (0…100%). */
function rateInHalfPct(rate: number, yMax: number): number {
  if (rate <= 0 || yMax <= 0) {
    return 0;
  }
  return Math.min(100, (rate / yMax) * 100);
}

/** Позиция линии сетки от низа всего графика (0…100%). */
function gridBottomPct(rate: number, yMax: number, aboveCenter: boolean): number {
  const half = yMax > 0 ? (rate / yMax) * 50 : 0;
  return aboveCenter ? 50 + half : 50 - half;
}

export function TrafficThroughputChart({ buckets, loading }: Props) {
  const { t } = useTranslation();
  const byteLabels = byteUnitLabelsFromT(t);
  const byteUnits = [
    byteLabels.byte,
    byteLabels.kib,
    byteLabels.mib,
    byteLabels.gib,
  ];
  const bpsUnits = [
    t("units.bytesPerSec"),
    t("units.kibPerSec"),
    t("units.mibPerSec"),
    t("units.gibPerSec"),
  ];
  const data = buckets?.length ? buckets : [];

  const { upBps, downBps, yMax } = useMemo(() => {
    const up = data.map((b) => bucketBytesUp(b).total / bucketSec);
    const down = data.map((b) => bucketBytesDown(b).total / bucketSec);
    let peak = 0;
    for (let i = 0; i < up.length; i++) {
      peak = Math.max(peak, up[i], down[i]);
    }
    return {
      upBps: up,
      downBps: down,
      yMax: chartYMax(peak),
    };
  }, [data]);

  const gridLines = useMemo(() => {
    const mid = yMax / 2;
    const values = [yMax, mid, 0];
    const lines: { bottomPct: number; isZero: boolean; isEdge: boolean }[] =
      [];
    for (const v of values) {
      if (v <= 0) {
        continue;
      }
      lines.push({
        bottomPct: gridBottomPct(v, yMax, true),
        isZero: false,
        isEdge: v === yMax,
      });
      lines.push({
        bottomPct: gridBottomPct(v, yMax, false),
        isZero: false,
        isEdge: v === yMax,
      });
    }
    lines.push({ bottomPct: 50, isZero: true, isEdge: true });
    return lines;
  }, [yMax]);

  const chartSkeleton = (
    <div className="flex h-full gap-[5px]">
      {Array.from({ length: 30 }).map((_, i) => (
        <div key={i} className="flex h-full flex-1 flex-col">
          <div className="flex flex-1 items-end">
            <div className="h-[25%] w-full animate-pulse rounded-sm bg-white/5" />
          </div>
          <div className="h-px shrink-0 bg-white/10" />
          <div className="flex flex-1 items-start">
            <div className="h-[20%] w-full animate-pulse rounded-sm bg-white/5" />
          </div>
        </div>
      ))}
    </div>
  );

  const chartBody = loading ? (
    <SkeletonLoader className="h-44 min-w-0 flex-1" loaderSize="md">
      {chartSkeleton}
    </SkeletonLoader>
  ) : (
    <div className="relative h-44 min-w-0 flex-1">
      <div className="pointer-events-none absolute inset-0">
        {gridLines.map((line, i) => (
          <div
            key={`${line.bottomPct}-${i}`}
            className={`absolute left-0 right-0 border-t ${
              line.isZero
                ? "border-white/25"
                : line.isEdge
                  ? "border-white/15"
                  : "border-dashed border-white/10"
            }`}
            style={{ bottom: `${line.bottomPct}%` }}
          />
        ))}
      </div>
      <div className="relative flex h-full gap-[5px]">
        {data.map((b, idx) => {
          const up = upBps[idx];
          const down = downBps[idx];
          const upParts = bucketBytesUp(b);
          const downParts = bucketBytesDown(b);
          const upStack = stackedHalfHeights(
            up,
            upParts.allow,
            upParts.deny,
            yMax,
          );
          const downStack = stackedHalfHeights(
            down,
            downParts.allow,
            downParts.deny,
            yMax,
          );
          const empty = up <= 0 && down <= 0;
          return (
            <Tooltip
              key={b.t}
              title={
                <BucketTooltipContent
                  b={b}
                  upBps={up}
                  downBps={down}
                  byteUnits={byteUnits}
                  bpsUnits={bpsUnits}
                  egressLabel={t("dashboard.egress")}
                  ingressLabel={t("dashboard.ingress")}
                />
              }
              placement="top"
              mouseEnterDelay={0.05}
            >
              <div className="relative flex h-full min-w-0 flex-1 cursor-default flex-col rounded-sm outline-none ring-sky-400/40 focus-visible:ring-2">
                <div className="flex min-h-0 flex-1 flex-col justify-end gap-[2px]">
                  {upStack.denyH > 0 ? (
                    <div
                      className={`traffic-bar-deny min-h-[2px] w-full ${barRadiusUpHalf(upStack.allowH, upStack.denyH, "deny")}`}
                      style={{ height: `${upStack.denyH}%` }}
                    />
                  ) : null}
                  {upStack.allowH > 0 ? (
                    <div
                      className={`traffic-bar-allow min-h-[2px] w-full ${barRadiusUpHalf(upStack.allowH, upStack.denyH, "allow")}`}
                      style={{ height: `${upStack.allowH}%` }}
                    />
                  ) : null}
                </div>
                <div
                  className="w-full shrink-0 border-t border-white/20"
                  aria-hidden
                />
                <div className="flex min-h-0 flex-1 flex-col justify-start gap-[2px]">
                  {downStack.allowH > 0 ? (
                    <div
                      className={`traffic-bar-allow min-h-[2px] w-full ${barRadiusDownHalf(downStack.allowH, downStack.denyH, "allow")}`}
                      style={{ height: `${downStack.allowH}%` }}
                    />
                  ) : null}
                  {downStack.denyH > 0 ? (
                    <div
                      className={`traffic-bar-deny min-h-[2px] w-full ${barRadiusDownHalf(downStack.allowH, downStack.denyH, "deny")}`}
                      style={{ height: `${downStack.denyH}%` }}
                    />
                  ) : null}
                </div>
                {empty ? (
                  <div
                    className="pointer-events-none absolute left-0 right-0 top-1/2 h-[2px] -translate-y-1/2 rounded-sm bg-white/5"
                  />
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
          className="relative h-44 w-10 shrink-0 font-mono text-[10px] leading-none text-zinc-500"
          aria-hidden
        >
          {loading ? (
            <div className="flex h-full flex-col justify-between py-px">
              {[0, 1, 2, 3, 4].map((i) => (
                <span
                  key={i}
                  className="inline-block h-3 w-6 animate-pulse rounded bg-white/5"
                />
              ))}
            </div>
          ) : (
            <>
              <span className="absolute right-0 top-0 text-right">
                {formatBytesPerSec(yMax, bpsUnits)}
              </span>
              <span className="absolute right-0 top-1/2 -translate-y-1/2 text-right text-zinc-400">
                0
              </span>
              <span className="absolute right-0 bottom-0 text-right">
                {formatBytesPerSec(yMax, bpsUnits)}
              </span>
            </>
          )}
        </div>
        {chartBody}
      </div>
      {!loading && data.length > 0 ? (
        <div className="flex justify-between pl-12 font-mono text-[10.5px] text-zinc-500">
          <span>{t("dashboard.minus5m")}</span>
          <span className="text-zinc-600">{t("dashboard.chartBuckets")}</span>
          <span>{t("dashboard.now")}</span>
        </div>
      ) : null}
    </div>
  );
}
