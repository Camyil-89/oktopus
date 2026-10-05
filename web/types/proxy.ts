export type HostSettings = {
  id: string;
  access_log_retention_days: number;
  updated_at: string;
};

export type ProxyInstance = {
  id: string;
  name: string;
  enabled: boolean;
  listen: string;
  connect_mode: "mitm" | "tunnel";
  auth_enabled: boolean;
  auth_static_users: string;
  auth_realm: string;
  auth_backend: "static" | "ldap";
  auth_cache_ttl_minutes: number;
  ldap_url: string;
  ldap_base_dn: string;
  ldap_bind_dn: string;
  ldap_bind_password_set: boolean;
  sort_order: number;
  updated_at: string;
};

/** @deprecated use ProxyInstance */
export type ProxySettings = ProxyInstance & {
  proxy_enabled: boolean;
  access_log_retention_days?: number;
};

export type ProxyCAStatus = {
  cert_installed: boolean;
  key_installed: boolean;
  valid_until?: string;
};

export type ProxyForbiddenPageStatus = {
  using_custom: boolean;
  custom_file: boolean;
  default_path: string;
  actual_path: string;
};

export type ProxyGatewayPageStatus = ProxyForbiddenPageStatus;

export type ProxyErrorPagePreviewVariant = "default" | "custom";

export type HostSettingsPatch = Partial<{
  access_log_retention_days: number;
}>;

export type ProxyInstancePatch = Partial<{
  name: string;
  enabled: boolean;
  listen: string;
  connect_mode: string;
  auth_enabled: boolean;
  auth_static_users: string;
  auth_realm: string;
  auth_backend: string;
  auth_cache_ttl_minutes: number;
  ldap_url: string;
  ldap_base_dn: string;
  ldap_bind_dn: string;
  ldap_bind_password: string;
  sort_order: number;
}>;

/** @deprecated */
export type ProxySettingsPatch = ProxyInstancePatch & Partial<{ proxy_enabled: boolean }>;

export type ProxyACLPolicy = {
  config_text: string;
  updated_at: string;
};

export type ProxyACLPolicyDiagnostic = {
  line: number;
  column: number;
  end_column?: number;
  severity: "error" | "warning";
  code: string;
  message: string;
  acl_name?: string;
};

export type ProxyACLPolicyAnalyzeResult = {
  ok: boolean;
  defined_acls: string[];
  diagnostics: ProxyACLPolicyDiagnostic[];
};

export type ProxyACLListSourceMode = "manual" | "remote";

export type ProxyACLNamedListSummary = {
  id: string;
  name: string;
  list_type: "src" | "dstdomain" | "port";
  source_mode: ProxyACLListSourceMode;
  source_url: string;
  poll_interval_minutes: number;
  body_line_count: number;
  body_preview: string;
  created_at: string;
  updated_at: string;
};

/** Полный list с body (модалка, опрос). */
export type ProxyACLNamedList = ProxyACLNamedListSummary & {
  body: string;
};

export type ProxyACLNamedListDraft = {
  id?: string;
  name: string;
  list_type: "src" | "dstdomain" | "port";
  body: string;
  source_mode: ProxyACLListSourceMode;
  source_url: string;
  poll_interval_minutes: number;
};

export type ProxyACLCompileStatus = {
  build_status: "idle" | "building" | "ready" | "error";
  build_error?: string;
  compile_diagnostics?: ProxyACLPolicyDiagnostic[];
  build_started_at?: string;
  build_finished_at?: string;
  last_compile_duration_ms?: number;
  config_revision: string;
  active_revision: string;
  rules_in_sync: boolean;
  active_logical_rules: number;
  active_patterns: number;
  active_sni_patterns_indexed?: number;
  active_sni_patterns_regexp?: number;
  active_slow_rule_slots?: number;
  sni_regexp_reason_counts?: Record<string, number>;
  sni_regexp_samples?: { line: string; reason: string }[];
  enabled_rules_in_db: number;
};

export type ProxyTrafficBucket = {
  t: number;
  requests: number;
  allow: number;
  deny: number;
  bytes_up?: number;
  bytes_down?: number;
  bytes_up_allow?: number;
  bytes_up_deny?: number;
  bytes_down_allow?: number;
  bytes_down_deny?: number;
};

export type ProxyInspectBreakdown5m = {
  prepare_us_avg_5m: number;
  body_us_avg_5m: number;
  eval_us_avg_5m: number;
  prepare_us_p95_5m: number;
  body_us_p95_5m: number;
  eval_us_p95_5m: number;
};

export type ProxyDecideBreakdown5m = {
  prepare_us_avg_5m: number;
  engine_us_avg_5m: number;
  scope_us_avg_5m: number;
  match_sni_us_avg_5m: number;
  match_src_ip_us_avg_5m: number;
  match_dst_ip_us_avg_5m: number;
  match_slow_us_avg_5m: number;
  prepare_us_p95_5m: number;
  engine_us_p95_5m: number;
  scope_us_p95_5m: number;
  match_sni_us_p95_5m: number;
  match_src_ip_us_p95_5m: number;
  match_dst_ip_us_p95_5m: number;
  match_slow_us_p95_5m: number;
};

export type ProxyTrafficSnapshot = {
  active_connections: number;
  active_websocket_connections: number;
  requests_per_sec_avg_5m: number;
  requests_per_sec_now_5s: number;
  requests_1m: number;
  allowed_5m: number;
  denied_5m: number;
  decide_duration_us_avg_5m: number;
  decide_duration_us_p95_5m: number;
  decide_duration_us_p99_5m: number;
  decide_duration_us_max_5m: number;
  decide_breakdown_5m?: ProxyDecideBreakdown5m;
  inspect_duration_us_avg_5m: number;
  inspect_duration_us_p95_5m: number;
  inspect_duration_us_p99_5m: number;
  inspect_breakdown_5m?: ProxyInspectBreakdown5m;
  policy_duration_us_avg_5m: number;
  policy_duration_us_p95_5m: number;
  policy_duration_us_p99_5m: number;
  unique_users_5m: number;
  unique_sources_5m: number;
  access_log_queue_pending: number;
  bytes_total_up_allow?: number;
  bytes_total_up_deny?: number;
  bytes_total_down_allow?: number;
  bytes_total_down_deny?: number;
  buckets_10s: ProxyTrafficBucket[];
};

export type ProxyStartupTimings = {
  started_at?: string;
  proxy_ready_at?: string;
  acl_db_synced_at?: string;
  proxy_ready_ms?: number;
  acl_db_sync_ms?: number;
};

export type ProxyInstanceRuntimeStatus = {
  id: string;
  listen: string;
  active: boolean;
  proxy_start_error?: string;
  acl: ProxyACLCompileStatus;
  traffic: ProxyTrafficSnapshot;
};

export type ProxyRuntimeStatus = {
  proxy_active: boolean;
  traffic: ProxyTrafficSnapshot;
  instances: ProxyInstanceRuntimeStatus[];
  startup?: ProxyStartupTimings;
};

export type ProxyACLEvaluateStep = {
  kind: string;
  block_index: number;
  rule_index?: number;
  scope_type?: string;
  scope_name?: string;
  rule_type?: string;
  action?: string;
  pattern?: string;
  message?: string;
  sort_order?: number;
};

export type ProxyACLEvaluateResult = {
  allowed: boolean;
  steps: ProxyACLEvaluateStep[];
};

export type ProxyACLEvaluateRequest = {
  sni?: string;
  path?: string;
  src_ip?: string;
  dst_ip?: string;
  dst_port?: number;
  username?: string;
  groups?: string[];
};

export type GenerateCARequest = {
  key_bits?: number;
  common_name?: string;
  organization?: string;
  country?: string;
  valid_days?: number;
};
