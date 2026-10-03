import type { TranslateFn } from "@/i18n/translate";

export const ACCESS_LOG_RULE_AUTH_FAIL = "system_auth_fail";
export const ACCESS_LOG_RULE_DEFAULT_DENY = "system_default_deny";
export const ACCESS_LOG_RULE_INSPECT_ERROR = "system_inspect_error";
export const ACCESS_LOG_RULE_GATEWAY_ERROR = "system_gateway_error";

/**
 * decision_rule_ref — итоговое правило записи:
 * текст http_access / ssl_verify, UUID инспекции или system_*.
 */

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
  [inspectRuleId: string]: unknown;
};

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
