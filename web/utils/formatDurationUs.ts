import type { TranslateFn } from "@/i18n/translate";

export type DurationUnitLabels = {
  us: string;
  ns: string;
  ms: string;
  sec: string;
  min: string;
};

export function durationUnitLabelsFromT(t: TranslateFn): DurationUnitLabels {
  return {
    us: t("units.us"),
    ns: t("units.ns"),
    ms: t("units.ms"),
    sec: t("units.sec"),
    min: t("units.min"),
  };
}

/** Человекочитаемая длительность из микросекунд (µs → ms → s → min). */
export function formatDurationUs(
  us: number | null | undefined,
  empty = "—",
  units: DurationUnitLabels = {
    us: "µs",
    ns: "ns",
    ms: "ms",
    sec: "s",
    min: "min",
  },
): string {
  if (us == null || us < 0 || Number.isNaN(us)) {
    return empty;
  }
  if (us === 0) {
    return `0 ${units.us}`;
  }
  if (us < 1) {
    const ns = us * 1_000;
    if (ns < 10) {
      return `${ns.toFixed(1)} ${units.ns}`;
    }
    return `${Math.round(ns)} ${units.ns}`;
  }
  if (us < 1_000) {
    const t = Math.round(us * 10) / 10;
    return Number.isInteger(t) ? `${t} ${units.us}` : `${t.toFixed(1)} ${units.us}`;
  }
  const ms = us / 1_000;
  if (ms < 1_000) {
    if (ms < 10) {
      return `${ms.toFixed(2)} ${units.ms}`;
    }
    if (ms < 100) {
      return `${ms.toFixed(1)} ${units.ms}`;
    }
    return `${Math.round(ms)} ${units.ms}`;
  }
  const sec = ms / 1_000;
  if (sec < 60) {
    return sec < 10 ? `${sec.toFixed(2)} ${units.sec}` : `${sec.toFixed(1)} ${units.sec}`;
  }
  const min = sec / 60;
  return `${min.toFixed(1)} ${units.min}`;
}
