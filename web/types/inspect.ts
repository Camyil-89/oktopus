export type ProxyInspectRuleListItem = {
  id: string;
  name: string;
  action: 0 | 1;
  enabled: boolean;
  sort_order: number;
  script_length: number;
  created_at: string;
  updated_at: string;
};

export type ProxyInspectRule = ProxyInspectRuleListItem & {
  script: string;
};

export type ProxyInspectCompileStatus = {
  build_status: string;
  build_error?: string;
  build_started_at?: string;
  build_finished_at?: string;
  config_revision: string;
  active_revision: string;
  rules_in_sync: boolean;
  active_rules: number;
  enabled_rules_in_db: number;
};

export type ProxyInspectRuleDraft = {
  id?: string;
  name: string;
  script: string;
  action: 0 | 1;
  enabled: boolean;
  sort_order: number;
};

export type SyncProxyInspectRulesPayload = {
  rules: ProxyInspectRuleDraft[];
};

export type ValidateProxyInspectScriptPayload = {
  script: string;
};

export type ValidateProxyInspectScriptResponse = {
  ok: boolean;
};
