import type { TranslateFn } from "@/i18n/translate";

export const ACCESS_LOG_RULE_AUTH_FAIL = "system_auth_fail";
export const ACCESS_LOG_RULE_DEFAULT_DENY = "system_default_deny";
export const ACCESS_LOG_RULE_INSPECT_ERROR = "system_inspect_error";
export const ACCESS_LOG_RULE_GATEWAY_ERROR = "system_gateway_error";

/**
 * decision_rule_ref — итоговое правило записи:
 * текст http_access / ssl_verify, UUID инспекции или system_*.
 */

export type PolicyAnomalyCheck = {
  detect?: boolean;
  connect_host?: string;
  tls_client_sni?: string;
  policy_host?: string;
  http_host?: string;
  url_host?: string;
  connect_port?: number;
  http_port?: number;
  resolved_ips?: string[];
};

/** kind + поля проверки (только detect=true в summary по умолчанию). */
export type PolicyAnomalyItem = PolicyAnomalyCheck & { kind: string };

/** Поля JSON extra: поиск, inspect_error, ctx:log по UUID правила. */
export type AccessLogExtra = {
  inspect_error?: string;
  gateway_error?: {
    type?: string;
    message?: string;
  };
  search?: {
    engine?: string;
    query?: string;
  };
  /** kind → { detect, … }; legacy: items[] или flat { kind }. */
  policy_anomaly?:
    | Record<string, PolicyAnomalyCheck>
    | { items?: PolicyAnomalyItem[] }
    | (PolicyAnomalyCheck & { kind?: string });
  [inspectRuleId: string]: unknown;
};

function isLegacyItemsPayload(
  raw: NonNullable<AccessLogExtra["policy_anomaly"]>,
): raw is { items: PolicyAnomalyItem[] } {
  return "items" in raw && Array.isArray(raw.items);
}

/** Нормализация: map по kind, legacy items/flat. */
export function normalizePolicyAnomalies(
  raw: AccessLogExtra["policy_anomaly"],
  opts?: { detectedOnly?: boolean },
): PolicyAnomalyItem[] {
  if (!raw || typeof raw !== "object") {
    return [];
  }
  const detectedOnly = opts?.detectedOnly ?? false;

  if (isLegacyItemsPayload(raw)) {
    return raw.items
      .filter((it) => it && typeof it === "object")
      .filter((it) => !detectedOnly || it.detect !== false);
  }
  if ("kind" in raw && typeof raw.kind === "string" && raw.kind) {
    const it = raw as PolicyAnomalyItem;
    if (detectedOnly && it.detect === false) {
      return [];
    }
    return [{ ...it, kind: raw.kind }];
  }

  const out: PolicyAnomalyItem[] = [];
  for (const [kind, check] of Object.entries(raw)) {
    if (!check || typeof check !== "object" || Array.isArray(check)) {
      continue;
    }
    const c = check as PolicyAnomalyCheck;
    if (detectedOnly && !c.detect) {
      continue;
    }
    out.push({ kind, ...c });
  }
  return out;
}

export function isSystemAccessLogRuleId(ruleId: string | undefined): boolean {
  return Boolean(ruleId?.startsWith("system_"));
}

const UUID_LIKE =
  /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

export function isAccessLogSquidDirectiveRef(ref: string): boolean {
  const s = ref.trim();
  return (
    s.startsWith("http_access ") ||
    s.startsWith("http_access:") ||
    s.startsWith("ssl_verify ")
  );
}

export function isAccessLogRecordId(value: string): boolean {
  return UUID_LIKE.test(value.trim());
}

const SYSTEM_RULE_KEYS: Record<string, Parameters<TranslateFn>[0]> = {
  [ACCESS_LOG_RULE_AUTH_FAIL]: "accessLog.system.auth_fail",
  [ACCESS_LOG_RULE_DEFAULT_DENY]: "accessLog.system.default_deny",
  [ACCESS_LOG_RULE_INSPECT_ERROR]: "accessLog.system.inspect_error",
  [ACCESS_LOG_RULE_GATEWAY_ERROR]: "accessLog.system.gateway_error",
};

export function accessLogSystemRuleLabel(ruleId: string, t: TranslateFn): string {
  const key = SYSTEM_RULE_KEYS[ruleId];
  if (key) {
    return t(key);
  }
  return t("accessLog.system.fallback", {
    id: ruleId.replace(/^system_/, ""),
  });
}

/** Текст для колонки «Правило ACL» — только Squid (не UUID инспекции). */
export function accessLogAclDecisionLabel(
  decisionRuleRef: string | undefined,
  t: TranslateFn,
): string {
  const ref = decisionRuleRef?.trim() ?? "";
  if (!ref) {
    return t("common.emDash");
  }
  if (isSystemAccessLogRuleId(ref)) {
    return accessLogSystemRuleLabel(ref, t);
  }
  if (isAccessLogSquidDirectiveRef(ref)) {
    return ref;
  }
  return t("common.emDash");
}

/** Подпись для decision_rule_ref в таблице и карточке записи. */
export function accessLogDecisionRuleLabel(
  decisionRuleRef: string | null | undefined,
  t: TranslateFn,
  ruleNames?: ReadonlyMap<string, string>,
): string {
  const ref = decisionRuleRef?.trim() ?? "";
  if (!ref) {
    return t("common.emDash");
  }
  if (isSystemAccessLogRuleId(ref)) {
    return accessLogSystemRuleLabel(ref, t);
  }
  if (isAccessLogSquidDirectiveRef(ref)) {
    return ref;
  }
  const inspectName = ruleNames?.get(ref);
  if (inspectName) {
    return inspectName;
  }
  if (isAccessLogRecordId(ref)) {
    return ref;
  }
  return ref;
}

const RESERVED_ACCESS_LOG_EXTRA_KEYS = new Set([
  "inspect_error",
  "gateway_error",
  "search",
  "policy_anomaly",
]);

export function accessLogInspectPayloads(
  extra: AccessLogExtra,
): { ruleId: string; data: Record<string, unknown> }[] {
  const out: { ruleId: string; data: Record<string, unknown> }[] = [];
  for (const [key, value] of Object.entries(extra)) {
    if (RESERVED_ACCESS_LOG_EXTRA_KEYS.has(key)) {
      continue;
    }
    if (!UUID_LIKE.test(key)) {
      continue;
    }
    if (value && typeof value === "object" && !Array.isArray(value)) {
      out.push({ ruleId: key, data: value as Record<string, unknown> });
    }
  }
  return out;
}

export function accessLogPolicyAnomalyItemSummary(
  anomaly: PolicyAnomalyItem,
  t: TranslateFn,
): string {
  if (!anomaly?.kind) {
    return "";
  }
  const kindKey = `accessLog.policyAnomaly.${anomaly.kind}` as Parameters<
    TranslateFn
  >[0];
  const kindLabel = t(kindKey);
  const parts: string[] = [kindLabel];
  if (anomaly.detect === false) {
    parts.push("detect=false");
  }
  if (anomaly.connect_host) {
    parts.push(`CONNECT=${anomaly.connect_host}`);
  }
  if (anomaly.tls_client_sni) {
    parts.push(`SNI=${anomaly.tls_client_sni}`);
  }
  if (anomaly.policy_host) {
    parts.push(`policy=${anomaly.policy_host}`);
  }
  if (anomaly.url_host) {
    parts.push(`URL=${anomaly.url_host}`);
  }
  if (anomaly.http_host) {
    parts.push(`Host=${anomaly.http_host}`);
  }
  if (anomaly.connect_port) {
    parts.push(`connect_port=${anomaly.connect_port}`);
  }
  if (anomaly.http_port) {
    parts.push(`http_port=${anomaly.http_port}`);
  }
  if (anomaly.resolved_ips?.length) {
    parts.push(`resolved=${anomaly.resolved_ips.join(",")}`);
  }
  return parts.join(" · ");
}

export function accessLogPolicyAnomalySummary(
  anomaly: AccessLogExtra["policy_anomaly"],
  t: TranslateFn,
): string {
  const items = normalizePolicyAnomalies(anomaly, { detectedOnly: true });
  if (items.length === 0) {
    return "";
  }
  return items
    .map((it) => accessLogPolicyAnomalyItemSummary(it, t))
    .filter(Boolean)
    .join("; ");
}

export function parseAccessLogExtra(extra: unknown): AccessLogExtra {
  if (!extra || typeof extra !== "object") {
    return {};
  }
  return extra as AccessLogExtra;
}

export function searchEngineLabel(
  engine: string | undefined,
  t: TranslateFn,
): string {
  if (!engine) return t("common.emDash");
  return engine;
}
