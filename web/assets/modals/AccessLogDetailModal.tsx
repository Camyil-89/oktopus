"use client";

import { ApiError } from "@/api/base";
import { getProxyInspectRule } from "@/api/proxy";
import { AccessLogPolicyAnomalyPanel } from "@/assets/modals/AccessLogPolicyAnomalyPanel";
import { scrollableModalProps } from "@/assets/modals/modalConfig";
import { ProxyInspectRuleViewModal } from "@/assets/modals/ProxyInspectRuleViewModal";
import type { ProxyAccessLogRow } from "@/types/accessLog";
import type { ProxyInspectRule } from "@/types/inspect";
import { AccessLogActionTag } from "@/utils/accessLogAction";
import {
  accessLogAclDecisionLabel,
  accessLogDecisionRuleLabel,
  accessLogSystemRuleLabel,
  isSystemAccessLogRuleId,
  parseAccessLogExtra,
  normalizePolicyAnomalies,
  ACCESS_LOG_RULE_INSPECT_ERROR,
} from "@/utils/accessLogExtra";
import { normalizeAccessLogRow } from "@/utils/accessLogRow";
import { OktopusLoading } from "@/assets/components/oktopus/OktopusLoading";
import { useTranslation } from "@/contexts/LocaleContext";
import type { TranslateFn } from "@/i18n/translate";
import { Descriptions, Modal, Tabs, Typography } from "antd";
import { useEffect, useState } from "react";

const ACCESS_LOG_DETAIL_BODY_MAX_HEIGHT = "min(85vh, 720px)";

type AccessLogDetailModalProps = {
  row: ProxyAccessLogRow | null;
  open: boolean;
  onClose: () => void;
  ruleNames?: ReadonlyMap<string, string>;
};

function useInspectRule(open: boolean, ruleId: string | undefined) {
  const [rule, setRule] = useState<{ id: string; name: string } | null>(null);
  const [ruleMissing, setRuleMissing] = useState(false);
  const [ruleLoading, setRuleLoading] = useState(false);

  useEffect(() => {
    if (!open || !ruleId || isSystemAccessLogRuleId(ruleId)) {
      setRule(null);
      setRuleMissing(false);
      setRuleLoading(false);
      return;
    }
    let cancelled = false;
    setRuleLoading(true);
    setRule(null);
    setRuleMissing(false);
    void getProxyInspectRule(ruleId)
      .then((r: ProxyInspectRule) => {
        if (!cancelled) {
          setRule({ id: r.id, name: r.name });
        }
      })
      .catch((e) => {
        if (cancelled) {
          return;
        }
        if (e instanceof ApiError && e.status === 404) {
          setRuleMissing(true);
          return;
        }
        setRuleMissing(true);
      })
      .finally(() => {
        if (!cancelled) {
          setRuleLoading(false);
        }
      });
    return () => {
      cancelled = true;
    };
  }, [open, ruleId]);

  return { rule, ruleMissing, ruleLoading };
}

function RuleLine({
  ruleId,
  loading,
  missing,
  rule,
  onOpen,
  t,
}: {
  ruleId: string | undefined;
  loading: boolean;
  missing: boolean;
  rule: { name: string } | null;
  onOpen?: () => void;
  t: TranslateFn;
}) {
  const emDash = t("common.emDash");
  if (!ruleId) {
    return <>{emDash}</>;
  }
  if (isSystemAccessLogRuleId(ruleId)) {
    return (
      <Typography.Text type="secondary">
        {accessLogSystemRuleLabel(ruleId, t)}
      </Typography.Text>
    );
  }
  if (loading) {
    return <OktopusLoading size="sm" />;
  }
  if (missing) {
    return (
      <Typography.Text type="warning">
        {t("accessLog.ruleNotFound", { ruleId })}
      </Typography.Text>
    );
  }
  if (!rule) {
    return <>{emDash}</>;
  }
  if (onOpen) {
    return <Typography.Link onClick={onOpen}>{rule.name}</Typography.Link>;
  }
  return <Typography.Text>{rule.name}</Typography.Text>;
}

export function AccessLogDetailModal({
  row,
  open,
  onClose,
  ruleNames,
}: AccessLogDetailModalProps) {
  const { t } = useTranslation();
  const emDash = t("common.emDash");
  const rowNorm = row ? normalizeAccessLogRow(row) : null;
  const extra = rowNorm ? parseAccessLogExtra(rowNorm.extra) : {};
  const extraJson =
    rowNorm != null
      ? JSON.stringify(parseAccessLogExtra(rowNorm.extra), null, 2)
      : "{}";
  const inspectRuleId = rowNorm?.inspect_rule_id ?? undefined;
  const decisionRef = rowNorm?.decision_rule_ref?.trim() ?? "";

  const inspect = useInspectRule(open, inspectRuleId);
  const [inspectViewOpen, setInspectViewOpen] = useState(false);
  const [activeTab, setActiveTab] = useState("general");

  useEffect(() => {
    if (!open) {
      setInspectViewOpen(false);
      setActiveTab("general");
    }
  }, [open]);

  const aclDecisionLabel = accessLogAclDecisionLabel(decisionRef, t);
  const decisionRuleLabel = accessLogDecisionRuleLabel(
    decisionRef,
    t,
    ruleNames,
  );
  const detectedAnomalyCount = normalizePolicyAnomalies(
    extra.policy_anomaly,
    { detectedOnly: true },
  ).length;
  const attackTabLabel =
    detectedAnomalyCount > 0 ? (
      <span>
        {t("accessLog.detailTabAttackVector")}
        <span className="ml-1 font-medium text-red-400">
          ({detectedAnomalyCount})
        </span>
      </span>
    ) : (
      t("accessLog.detailTabAttackVector")
    );

  const generalTab = rowNorm ? (
    <Descriptions column={1} bordered size="small">
      <Descriptions.Item label="ID">{rowNorm.id}</Descriptions.Item>
      <Descriptions.Item label={t("common.time")}>
        {new Date(rowNorm.created_at).toLocaleString()}
      </Descriptions.Item>
      <Descriptions.Item label={t("common.source")}>
        {rowNorm.source_address}
      </Descriptions.Item>
      <Descriptions.Item label={t("common.destination")}>
        {rowNorm.destination_address}
      </Descriptions.Item>
      <Descriptions.Item label={t("common.user")}>
        {rowNorm.user ?? emDash}
      </Descriptions.Item>
      <Descriptions.Item label={t("common.action")}>
        <AccessLogActionTag
          action={rowNorm.action}
          decisionRuleRef={rowNorm.decision_rule_ref}
          deniedBy={rowNorm.denied_by}
        />
      </Descriptions.Item>
      <Descriptions.Item label={t("accessLog.detailAclDecision")}>
        <Typography.Text className="break-all font-mono text-sm">
          {aclDecisionLabel}
        </Typography.Text>
      </Descriptions.Item>
      <Descriptions.Item label={t("accessLog.detailDecisionRule")}>
        <Typography.Text className="break-all font-mono text-sm">
          {decisionRuleLabel}
        </Typography.Text>
      </Descriptions.Item>
      {inspectRuleId ? (
        <Descriptions.Item label={t("accessLog.detailInspectRule")}>
          <RuleLine
            ruleId={inspectRuleId}
            loading={inspect.ruleLoading}
            missing={inspect.ruleMissing}
            rule={inspect.rule}
            onOpen={() => setInspectViewOpen(true)}
            t={t}
          />
        </Descriptions.Item>
      ) : rowNorm.denied_by === "inspect" &&
        rowNorm.decision_rule_ref === ACCESS_LOG_RULE_INSPECT_ERROR ? (
        <Descriptions.Item label={t("accessLog.detailInspect")}>
          <Typography.Text type="secondary">
            {accessLogSystemRuleLabel(rowNorm.decision_rule_ref, t)}
            {extra.inspect_error ? (
              <span className="block text-xs opacity-80">
                {extra.inspect_error}
              </span>
            ) : null}
          </Typography.Text>
        </Descriptions.Item>
      ) : null}
      <Descriptions.Item label={t("accessLog.col.decide_duration_us")}>
        {rowNorm.decide_duration_us}
      </Descriptions.Item>
      <Descriptions.Item label="URL">
        <Typography.Text className="break-all">
          {rowNorm.full_url}
        </Typography.Text>
      </Descriptions.Item>
    </Descriptions>
  ) : null;

  const extraTab = (
    <pre
      className="m-0 overflow-auto whitespace-pre-wrap rounded-md border border-white/10 bg-black/25 p-4 font-mono text-xs leading-relaxed"
      style={{ maxHeight: "min(62vh, 560px)" }}
    >
      {extraJson}
    </pre>
  );

  return (
    <>
      <Modal
        title={t("accessLog.detailTitle")}
        open={open}
        onCancel={onClose}
        footer={null}
        width={720}
        {...scrollableModalProps}
        styles={{
          ...scrollableModalProps.styles,
          body: {
            ...scrollableModalProps.styles?.body,
            maxHeight: ACCESS_LOG_DETAIL_BODY_MAX_HEIGHT,
          },
        }}
      >
        {rowNorm ? (
          <Tabs
            activeKey={activeTab}
            onChange={setActiveTab}
            className="[&_.ant-tabs-content-holder]:pt-4"
            items={[
              {
                key: "general",
                label: t("accessLog.detailTabGeneral"),
                children: generalTab,
              },
              {
                key: "attack",
                label: attackTabLabel,
                children: (
                  <AccessLogPolicyAnomalyPanel
                    policyAnomaly={extra.policy_anomaly}
                  />
                ),
              },
              {
                key: "extra",
                label: t("accessLog.detailTabExtra"),
                children: extraTab,
              },
            ]}
          />
        ) : null}
      </Modal>
      <ProxyInspectRuleViewModal
        open={inspectViewOpen}
        ruleId={inspectRuleId ?? null}
        initialRule={null}
        onClose={() => setInspectViewOpen(false)}
      />
    </>
  );
}
