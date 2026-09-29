"use client";

import {
  accessLogSystemRuleLabel,
  isSystemAccessLogRuleId,
} from "@/utils/accessLogExtra";
import { Typography } from "antd";

type AccessLogRuleNameCellProps = {
  ruleId: string | null | undefined;
  ruleNames: ReadonlyMap<string, string>;
};

export function AccessLogRuleNameCell({
  ruleId,
  ruleNames,
}: AccessLogRuleNameCellProps) {
  if (!ruleId) {
    return <>—</>;
  }
  if (isSystemAccessLogRuleId(ruleId)) {
    return (
      <Typography.Text
        type="secondary"
        className="text-xs"
        title={ruleId}
      >
        {accessLogSystemRuleLabel(ruleId)}
      </Typography.Text>
    );
  }
  const name = ruleNames.get(ruleId);
  const label = name ?? ruleId;
  return (
    <Typography.Text ellipsis className="max-w-[220px] text-xs" title={ruleId}>
      {label}
    </Typography.Text>
  );
}
