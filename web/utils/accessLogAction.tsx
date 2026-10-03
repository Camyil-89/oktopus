"use client";

import { Tag } from "antd";
import type { ReactElement } from "react";
import { useTranslation } from "@/contexts/LocaleContext";
import type { MessageKey } from "@/i18n/translate";
import type { TranslateFn } from "@/i18n/translate";
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

const OUTCOME_LABEL_KEYS: Record<AccessLogOutcomeKind, MessageKey> = {
  allow: "accessLog.action.allow",
  auth_fail: "accessLog.action.auth_fail",
  default_deny: "accessLog.action.default_deny",
  deny_rule: "accessLog.action.deny_rule",
  inspect_deny: "accessLog.action.inspect_deny",
  inspect_error: "accessLog.action.inspect_error",
  gateway_error: "accessLog.action.gateway_error",
  deny: "accessLog.action.deny",
};

const OUTCOME_COLORS: Record<AccessLogOutcomeKind, string> = {
  allow: "green",
  auth_fail: "gold",
  default_deny: "red",
  deny_rule: "red",
  inspect_deny: "red",
  inspect_error: "red",
  gateway_error: "red",
  deny: "red",
};

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

export function accessLogOutcomeLabel(
  action: AccessLogActionCode | number,
  t: TranslateFn,
  decisionRuleRef?: string | null,
  deniedBy?: "acl" | "inspect" | "gateway" | null,
): string {
  const kind = accessLogOutcomeKind(action, decisionRuleRef, deniedBy);
  return t(OUTCOME_LABEL_KEYS[kind]);
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
  const { t } = useTranslation();
  const kind = accessLogOutcomeKind(action, decisionRuleRef, deniedBy);
  return <Tag color={OUTCOME_COLORS[kind]}>{t(OUTCOME_LABEL_KEYS[kind])}</Tag>;
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
  const { t } = useTranslation();
  return action === 1 ? (
    <Tag color="green">{t("accessLog.aggregateAllowed")}</Tag>
  ) : (
    <Tag color="red">{t("accessLog.aggregateDenied")}</Tag>
  );
}
