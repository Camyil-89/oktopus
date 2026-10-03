"use client";

import { accessLogDecisionRuleLabel } from "@/utils/accessLogExtra";
import { useTranslation } from "@/contexts/LocaleContext";
import { Typography } from "antd";

type AccessLogDecisionRuleCellProps = {
  decisionRuleRef: string | null | undefined;
  ruleNames: ReadonlyMap<string, string>;
};

export function AccessLogDecisionRuleCell({
  decisionRuleRef,
  ruleNames,
}: AccessLogDecisionRuleCellProps) {
  const { t } = useTranslation();
  const label = accessLogDecisionRuleLabel(decisionRuleRef, t, ruleNames);
  const emDash = t("common.emDash");
  if (label === emDash) {
    return <>{emDash}</>;
  }
  const raw = decisionRuleRef?.trim() ?? "";
  return (
    <Typography.Text
      ellipsis
      className="max-w-[280px] font-mono text-xs"
      title={raw || label}
    >
      {label}
    </Typography.Text>
  );
}
