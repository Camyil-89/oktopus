import type { ProxyTrafficBucket } from "@/types/proxy";

/** Суммирует 10s-бакеты по полю `t` (сводка fleet с API или fallback по инстансам). */
export function mergeTrafficBuckets(
  lists: ProxyTrafficBucket[][],
): ProxyTrafficBucket[] {
  const byT = new Map<number, ProxyTrafficBucket>();
  const order: number[] = [];
  for (const list of lists) {
    for (const b of list) {
      const prev = byT.get(b.t);
      if (!prev) {
        byT.set(b.t, { ...b });
        order.push(b.t);
        continue;
      }
      byT.set(b.t, {
        t: b.t,
        requests: prev.requests + b.requests,
        allow: prev.allow + b.allow,
        deny: prev.deny + b.deny,
        bytes_up: (prev.bytes_up ?? 0) + (b.bytes_up ?? 0),
        bytes_down: (prev.bytes_down ?? 0) + (b.bytes_down ?? 0),
        bytes_up_allow: (prev.bytes_up_allow ?? 0) + (b.bytes_up_allow ?? 0),
        bytes_up_deny: (prev.bytes_up_deny ?? 0) + (b.bytes_up_deny ?? 0),
        bytes_down_allow:
          (prev.bytes_down_allow ?? 0) + (b.bytes_down_allow ?? 0),
        bytes_down_deny: (prev.bytes_down_deny ?? 0) + (b.bytes_down_deny ?? 0),
      });
    }
  }
  order.sort((a, b) => a - b);
  return order.map((t) => byT.get(t)!);
}
