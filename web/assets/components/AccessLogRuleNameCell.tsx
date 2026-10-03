"use client";

import {
  accessLogSystemRuleLabel,
  isSystemAccessLogRuleId,
} from "@/utils/accessLogExtra";
import { useTranslation } from "@/contexts/LocaleContext";
import { Typography } from "antd";

type AccessLogRuleNameCellProps = {
  ruleId: string | null | undefined;
  ruleNames: ReadonlyMap<string, string>;
};

export function AccessLogRuleNameCell({
  ruleId,
  ruleNames,
}: AccessLogRuleNameCellProps) {
  const { t } = useTranslation();
  const emDash = t("common.emDash");
  if (!ruleId) {
    return <>{emDash}</>;
  }
  if (isSystemAccessLogRuleId(ruleId)) {
    return (
      <Typography.Text
        type="secondary"
        className="text-xs"
        title={ruleId}
      >
        {accessLogSystemRuleLabel(ruleId, t)}
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
