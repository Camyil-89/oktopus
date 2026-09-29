import type { ProxyAccessLogRow } from "@/types/accessLog";
import { parseAccessLogExtra } from "@/utils/accessLogExtra";

export function normalizeAccessLogRow(row: ProxyAccessLogRow): ProxyAccessLogRow {
  return {
    ...row,
    extra: parseAccessLogExtra(row.extra),
  };
}

export function accessLogDecisionRuleLabel(
  ruleId: string | undefined,
  ruleNames: ReadonlyMap<string, string>,
): string {
  if (!ruleId) {
    return "—";
  }
  const name = ruleNames.get(ruleId);
  if (name) {
    return name;
  }
  return ruleId;
}
