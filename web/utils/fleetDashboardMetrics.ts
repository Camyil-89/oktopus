import type {
  ProxyACLCompileStatus,
  ProxyInstanceRuntimeStatus,
  ProxyTrafficSnapshot,
} from "@/types/proxy";

function avgInt(values: number[]): number {
  if (values.length === 0) {
    return 0;
  }
  return Math.round(values.reduce((a, b) => a + b, 0) / values.length);
}

function avgFloat(values: number[]): number {
  if (values.length === 0) {
    return 0;
  }
  return values.reduce((a, b) => a + b, 0) / values.length;
}

export function fleetAclBuildStatus(
  instances: ProxyInstanceRuntimeStatus[],
): ProxyACLCompileStatus["build_status"] {
  if (instances.length === 0) {
    return "idle";
  }
  const statuses = instances.map((i) => i.acl?.build_status ?? "idle");
  if (statuses.some((s) => s === "error")) {
    return "error";
  }
  if (statuses.some((s) => s === "building")) {
    return "building";
  }
  if (statuses.every((s) => s === "ready")) {
    return "ready";
  }
  return "idle";
}

/** Сводный ACL для карточек и fleet-runtime: сумма правил/паттернов, общий build status. */
export function fleetAclSummary(
  instances: ProxyInstanceRuntimeStatus[],
): ProxyACLCompileStatus | null {
  if (instances.length === 0) {
    return null;
  }
  const template = instances[0].acl;
  const rules = instances.reduce(
    (s, i) => s + (i.acl?.active_logical_rules ?? 0),
    0,
  );
  const patterns = instances.reduce(
    (s, i) => s + (i.acl?.active_patterns ?? 0),
    0,
  );
  const sniIndexed = avgInt(
    instances.map((i) => i.acl?.active_sni_patterns_indexed ?? 0),
  );
  const sniRegexp = avgInt(
    instances.map((i) => i.acl?.active_sni_patterns_regexp ?? 0),
  );
  const slowSlots = avgInt(
    instances.map((i) => i.acl?.active_slow_rule_slots ?? 0),
  );
  return {
    ...template,
    build_status: fleetAclBuildStatus(instances),
    build_error: instances
      .map((i) => i.acl?.build_error)
      .filter(Boolean)
      .join("; ") || undefined,
    rules_in_sync: instances.every((i) => i.acl?.rules_in_sync !== false),
    active_logical_rules: rules,
    active_patterns: patterns,
    active_sni_patterns_indexed: sniIndexed,
    active_sni_patterns_regexp: sniRegexp,
    active_slow_rule_slots: slowSlots,
    sni_regexp_reason_counts: undefined,
    sni_regexp_samples: undefined,
  };
}

/** Средние runtime-метрики по инстансам (главная, блок Runtime). */
export function averageInstanceTraffic(
  instances: ProxyInstanceRuntimeStatus[],
): ProxyTrafficSnapshot | null {
  const rows = instances.map((i) => i.traffic).filter(Boolean);
  if (rows.length === 0) {
    return null;
  }
  const pick = (fn: (t: ProxyTrafficSnapshot) => number) =>
    rows.map(fn);
  return {
    active_connections: avgInt(pick((t) => t.active_connections)),
    active_websocket_connections: avgInt(
      pick((t) => t.active_websocket_connections ?? 0),
    ),
    requests_per_sec_avg_5m: avgFloat(pick((t) => t.requests_per_sec_avg_5m)),
    requests_per_sec_now_5s: avgFloat(pick((t) => t.requests_per_sec_now_5s)),
    requests_1m: avgInt(pick((t) => t.requests_1m)),
    allowed_5m: avgInt(pick((t) => t.allowed_5m)),
    denied_5m: avgInt(pick((t) => t.denied_5m)),
    decide_duration_us_avg_5m: avgFloat(pick((t) => t.decide_duration_us_avg_5m)),
    decide_duration_us_p95_5m: avgInt(pick((t) => t.decide_duration_us_p95_5m)),
    decide_duration_us_p99_5m: avgInt(pick((t) => t.decide_duration_us_p99_5m)),
    decide_duration_us_max_5m: avgInt(pick((t) => t.decide_duration_us_max_5m)),
    inspect_duration_us_avg_5m: avgFloat(pick((t) => t.inspect_duration_us_avg_5m)),
    inspect_duration_us_p95_5m: avgInt(pick((t) => t.inspect_duration_us_p95_5m)),
    inspect_duration_us_p99_5m: avgInt(pick((t) => t.inspect_duration_us_p99_5m)),
    policy_duration_us_avg_5m: avgFloat(pick((t) => t.policy_duration_us_avg_5m)),
    policy_duration_us_p95_5m: avgInt(pick((t) => t.policy_duration_us_p95_5m)),
    policy_duration_us_p99_5m: avgInt(pick((t) => t.policy_duration_us_p99_5m)),
    unique_users_5m: avgInt(pick((t) => t.unique_users_5m)),
    unique_sources_5m: avgInt(pick((t) => t.unique_sources_5m)),
    access_log_queue_pending: avgInt(pick((t) => t.access_log_queue_pending)),
    bytes_total_up_allow: avgInt(pick((t) => t.bytes_total_up_allow ?? 0)),
    bytes_total_up_deny: avgInt(pick((t) => t.bytes_total_up_deny ?? 0)),
    bytes_total_down_allow: avgInt(pick((t) => t.bytes_total_down_allow ?? 0)),
    bytes_total_down_deny: avgInt(pick((t) => t.bytes_total_down_deny ?? 0)),
    buckets_10s: [],
  };
}
