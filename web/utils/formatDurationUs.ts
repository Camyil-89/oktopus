/** Человекочитаемая длительность из микросекунд (µs → ms → s → min). */
export function formatDurationUs(
  us: number | null | undefined,
  empty = "—",
): string {
  if (us == null || us < 0 || Number.isNaN(us)) {
    return empty;
  }
  if (us === 0) {
    return "0 µs";
  }
  if (us < 1) {
    const ns = us * 1_000;
    if (ns < 10) {
      return `${ns.toFixed(1)} ns`;
    }
    return `${Math.round(ns)} ns`;
  }
  if (us < 1_000) {
    const t = Math.round(us * 10) / 10;
    return Number.isInteger(t) ? `${t} µs` : `${t.toFixed(1)} µs`;
  }
  const ms = us / 1_000;
  if (ms < 1_000) {
    if (ms < 10) {
      return `${ms.toFixed(2)} ms`;
    }
    if (ms < 100) {
      return `${ms.toFixed(1)} ms`;
    }
    return `${Math.round(ms)} ms`;
  }
  const sec = ms / 1_000;
  if (sec < 60) {
    return sec < 10 ? `${sec.toFixed(2)} s` : `${sec.toFixed(1)} s`;
  }
  const min = sec / 60;
  return `${min.toFixed(1)} min`;
}
