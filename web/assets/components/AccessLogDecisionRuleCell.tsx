"use client";

import { accessLogDecisionRuleLabel } from "@/utils/accessLogExtra";
import { Typography } from "antd";

type AccessLogDecisionRuleCellProps = {
  decisionRuleRef: string | null | undefined;
  ruleNames: ReadonlyMap<string, string>;
};

export function AccessLogDecisionRuleCell({
  decisionRuleRef,
  ruleNames,
}: AccessLogDecisionRuleCellProps) {
  const label = accessLogDecisionRuleLabel(decisionRuleRef, ruleNames);
  if (label === "—") {
    return <>—</>;
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
