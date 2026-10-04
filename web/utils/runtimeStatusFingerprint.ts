import type {
  ProxyACLCompileStatus,
  ProxyInstanceRuntimeStatus,
  ProxyRuntimeStatus,
  ProxyTrafficSnapshot,
} from "@/types/proxy";

function trafficFp(t: ProxyTrafficSnapshot | undefined): string {
  if (!t) {
    return "";
  }
  const buckets = (t.buckets_10s ?? [])
    .map(
      (b) =>
        `${b.t},${b.requests},${b.allow},${b.deny},${b.bytes_up ?? 0},${b.bytes_down ?? 0}`,
    )
    .join(";");
  return [
    t.active_connections,
    t.active_websocket_connections ?? 0,
    t.requests_1m,
    t.allowed_5m,
    t.denied_5m,
    t.requests_per_sec_avg_5m,
    t.requests_per_sec_now_5s,
    t.decide_duration_us_avg_5m,
    t.decide_duration_us_p95_5m,
    t.decide_duration_us_p99_5m,
    t.inspect_duration_us_avg_5m,
    t.inspect_duration_us_p95_5m,
    t.inspect_duration_us_p99_5m,
    t.policy_duration_us_avg_5m,
    t.policy_duration_us_p95_5m,
    t.policy_duration_us_p99_5m,
    t.unique_users_5m,
    t.unique_sources_5m,
    t.access_log_queue_pending,
    t.bytes_total_up_allow ?? 0,
    t.bytes_total_up_deny ?? 0,
    t.bytes_total_down_allow ?? 0,
    t.bytes_total_down_deny ?? 0,
    buckets,
  ].join("|");
}

function aclFp(a: ProxyACLCompileStatus | undefined): string {
  if (!a) {
    return "";
  }
  return [
    a.build_status,
    a.build_error ?? "",
    a.rules_in_sync,
    a.active_logical_rules,
    a.active_patterns,
    a.active_sni_patterns_indexed ?? 0,
    a.active_sni_patterns_regexp ?? 0,
    a.active_slow_rule_slots ?? 0,
  ].join("|");
}

export function instanceRuntimeFingerprint(
  i: ProxyInstanceRuntimeStatus,
): string {
  return instanceFp(i);
}

function instanceFp(i: ProxyInstanceRuntimeStatus): string {
  return [
    i.id,
    i.active,
    i.listen,
    i.proxy_start_error ?? "",
    trafficFp(i.traffic),
    aclFp(i.acl),
  ].join(":");
}

/** Стабильная строка для сравнения poll-ответов без лишнего setState. */
export function runtimeStatusFingerprint(st: ProxyRuntimeStatus): string {
  const instances = [...st.instances]
    .sort((a, b) => a.id.localeCompare(b.id))
    .map(instanceFp)
    .join("||");
  const startup = st.startup
    ? [
        st.startup.started_at ?? "",
        st.startup.proxy_ready_at ?? "",
        st.startup.proxy_ready_ms ?? "",
      ].join(",")
    : "";
  return [st.proxy_active, trafficFp(st.traffic), instances, startup].join("##");
}
