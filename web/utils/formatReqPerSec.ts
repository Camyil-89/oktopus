export function formatReqPerSec(rps: number | null | undefined): string {
  if (rps == null || Number.isNaN(rps)) {
    return "—";
  }
  if (rps === 0) {
    return "0";
  }
  if (rps < 10) {
    return rps.toFixed(2);
  }
  if (rps < 100) {
    return rps.toFixed(1);
  }
  return Math.round(rps).toString();
}
