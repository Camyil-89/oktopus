"use client";

import type { ProxyRuleKind } from "@/assets/hooks/useProxyRuleNameMap";
import {
  accessLogSystemRuleLabel,
  isAccessLogSquidDirectiveRef,
  isSystemAccessLogRuleId,
} from "@/utils/accessLogExtra";
import {
  isAccessLogRuleRefColumn,
  looksLikeRuleUUID,
} from "@/utils/accessLogRuleRef";
import {
  AccessLogAggregateActionTag,
  parseAccessLogActionAggregateValue,
} from "@/utils/accessLogAction";
import { Typography } from "antd";

type ReportRuleRefLabelProps = {
  value: string;
  column?: string;
  ruleNames: ReadonlyMap<string, string>;
  ruleKinds: ReadonlyMap<string, ProxyRuleKind>;
  onOpenRule?: (ruleId: string, kind: ProxyRuleKind) => void;
  className?: string;
};

export function ReportRuleRefLabel({
  value,
  column,
  ruleNames,
  ruleKinds,
  onOpenRule,
  className,
}: ReportRuleRefLabelProps) {
  const raw = value.trim();
  if (!raw || raw === "—") {
    return <span className={className}>—</span>;
  }

  if (column === "action") {
    const code = parseAccessLogActionAggregateValue(raw);
    if (code !== null) {
      return (
        <span className={className}>
          <AccessLogAggregateActionTag action={code} />
        </span>
      );
    }
  }

  const isRuleField =
    (column && isAccessLogRuleRefColumn(column)) ||
    (!column && looksLikeRuleUUID(raw));

  if (!isRuleField) {
    return (
      <span className={className} title={raw}>
        {raw}
      </span>
    );
  }

  if (isSystemAccessLogRuleId(raw)) {
    return (
      <Typography.Text
        type="secondary"
        className={className}
        title={raw}
      >
        {accessLogSystemRuleLabel(raw)}
      </Typography.Text>
    );
  }

  if (isAccessLogSquidDirectiveRef(raw)) {
    return (
      <Typography.Text
        className={`font-mono text-xs ${className ?? ""}`}
        title={raw}
      >
        {raw}
      </Typography.Text>
    );
  }

  const name = ruleNames.get(raw);
  const kind = ruleKinds.get(raw);
  const display = name ?? raw;

  if (name && kind && onOpenRule) {
    return (
      <Typography.Link
        className={`max-w-full truncate text-[11px] ${className ?? ""}`}
        title={raw}
        onClick={(e) => {
          e.preventDefault();
          e.stopPropagation();
          onOpenRule(raw, kind);
        }}
      >
        {name}
      </Typography.Link>
    );
  }

  return (
    <span className={`truncate ${className ?? ""}`} title={raw}>
      {display}
    </span>
  );
}
