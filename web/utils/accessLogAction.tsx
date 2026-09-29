import { Tag } from "antd";
import type { ReactElement } from "react";
import {
  ACCESS_LOG_RULE_AUTH_FAIL,
  ACCESS_LOG_RULE_DEFAULT_DENY,
  ACCESS_LOG_RULE_GATEWAY_ERROR,
  ACCESS_LOG_RULE_INSPECT_ERROR,
  isSystemAccessLogRuleId,
} from "@/utils/accessLogExtra";

export type AccessLogActionCode = 0 | 1;

export type AccessLogOutcomeKind =
  | "allow"
  | "auth_fail"
  | "default_deny"
  | "deny_rule"
  | "inspect_deny"
  | "inspect_error"
  | "gateway_error"
  | "deny";

export function accessLogOutcomeKind(
  action: AccessLogActionCode | number,
  decisionRuleRef?: string | null,
  deniedBy?: "acl" | "inspect" | "gateway" | null,
): AccessLogOutcomeKind {
  if (Number(action) === 1) {
    return "allow";
  }
  if (deniedBy === "gateway") {
    return "gateway_error";
  }
  if (deniedBy === "inspect") {
    if (decisionRuleRef === ACCESS_LOG_RULE_INSPECT_ERROR) {
      return "inspect_error";
    }
    return "inspect_deny";
  }
  if (decisionRuleRef === ACCESS_LOG_RULE_AUTH_FAIL) {
    return "auth_fail";
  }
  if (decisionRuleRef === ACCESS_LOG_RULE_DEFAULT_DENY) {
    return "default_deny";
  }
  if (decisionRuleRef === ACCESS_LOG_RULE_GATEWAY_ERROR) {
    return "gateway_error";
  }
  if (decisionRuleRef && !isSystemAccessLogRuleId(decisionRuleRef)) {
    return "deny_rule";
  }
  return "deny";
}

const OUTCOME_META: Record<
  AccessLogOutcomeKind,
  { label: string; color: string }
> = {
  allow: { label: "Разрешено", color: "green" },
  auth_fail: { label: "Ошибка авторизации", color: "gold" },
  default_deny: { label: "Отказано (по умолчанию)", color: "red" },
  deny_rule: { label: "Запрещено правилом", color: "red" },
  inspect_deny: { label: "Запрещено инспекцией", color: "red" },
  inspect_error: { label: "Отклонено (ошибка инспекции)", color: "red" },
  gateway_error: { label: "Ошибка шлюза (502)", color: "red" },
  deny: { label: "Отказано", color: "red" },
};

export function accessLogOutcomeLabel(
  action: AccessLogActionCode | number,
  decisionRuleRef?: string | null,
  deniedBy?: "acl" | "inspect" | "gateway" | null,
): string {
  return OUTCOME_META[
    accessLogOutcomeKind(action, decisionRuleRef, deniedBy)
  ].label;
}

export function AccessLogActionTag({
  action,
  decisionRuleRef,
  deniedBy,
}: {
  action: AccessLogActionCode | number;
  decisionRuleRef?: string | null;
  deniedBy?: "acl" | "inspect" | "gateway" | null;
}): ReactElement {
  const kind = accessLogOutcomeKind(action, decisionRuleRef, deniedBy);
  const { label, color } = OUTCOME_META[kind];
  return <Tag color={color}>{label}</Tag>;
}

/** Значение action из агрегаций отчёта (текст "0"/"1", allow/deny). */
export function parseAccessLogActionAggregateValue(
  value: string,
): AccessLogActionCode | null {
  const v = value.trim().toLowerCase();
  if (v === "1" || v === "allow") {
    return 1;
  }
  if (v === "0" || v === "deny") {
    return 0;
  }
  return null;
}

/** allow/deny без decision_rule_ref (таблицы отчётов, group_by action). */
export function AccessLogAggregateActionTag({
  action,
}: {
  action: AccessLogActionCode;
}): ReactElement {
  return action === 1 ? (
    <Tag color="green">Разрешено</Tag>
  ) : (
    <Tag color="red">Запрещено</Tag>
  );
}
