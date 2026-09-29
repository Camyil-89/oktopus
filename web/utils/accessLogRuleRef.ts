const RULE_REF_GROUP_BY_COLUMNS = new Set([
  "decision_rule_ref",
  "inspect_rule_id",
]);

export function isAccessLogRuleRefColumn(column: string): boolean {
  return RULE_REF_GROUP_BY_COLUMNS.has(column);
}

const UUID_LIKE =
  /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

export function looksLikeRuleUUID(value: string): boolean {
  const v = value.trim();
  return v !== "" && v !== "—" && UUID_LIKE.test(v);
}
