"use client";

import type { AccessLogExtra } from "@/utils/accessLogExtra";
import {
  accessLogPolicyAnomalyDetailRows,
  accessLogPolicyAnomalyKindLabel,
  orderedPolicyAnomalyItems,
} from "@/utils/accessLogExtra";
import { useTranslation } from "@/contexts/LocaleContext";
import { Descriptions, Tag, Typography } from "antd";

type AccessLogPolicyAnomalyPanelProps = {
  policyAnomaly: AccessLogExtra["policy_anomaly"];
};

export function AccessLogPolicyAnomalyPanel({
  policyAnomaly,
}: AccessLogPolicyAnomalyPanelProps) {
  const { t } = useTranslation();
  const items = orderedPolicyAnomalyItems(policyAnomaly);
  const notDetected = items.filter((it) => it.detect === false);
  const detected = items.filter((it) => it.detect === true);

  if (items.length === 0) {
    return (
      <Typography.Text type="secondary">
        {t("accessLog.policyAnomaly.empty")}
      </Typography.Text>
    );
  }

  return (
    <div className="flex flex-col gap-4">
      {notDetected.length > 0 && detected.length === 0 ? (
        <div
          className="flex flex-col gap-3 rounded-md border border-emerald-500/35 bg-emerald-500/10 px-4 py-3"
          role="status"
        >
          <Typography.Text className="text-xs font-medium text-emerald-200/90">
            {t("accessLog.policyAnomaly.notInRequest")}
          </Typography.Text>
          <div className="flex flex-row flex-wrap gap-2">
            {notDetected.map((it) => (
              <Tag
                key={it.kind}
                className="m-0 border-emerald-500/40 bg-emerald-500/20 text-emerald-100"
              >
                {accessLogPolicyAnomalyKindLabel(it.kind, t)}
              </Tag>
            ))}
          </div>
        </div>
      ) : null}

      {detected.length > 0 ? (
        <div className="flex flex-col gap-3">
          <Typography.Text className="text-xs font-medium uppercase tracking-wide text-amber-200/80">
            {t("accessLog.policyAnomaly.detectedTitle")}
          </Typography.Text>
          {detected.map((it) => {
            const rows = accessLogPolicyAnomalyDetailRows(it, t);
            return (
              <div
                key={it.kind}
                className="overflow-hidden rounded-md border border-amber-500/30 bg-amber-500/5"
              >
                <div className="border-b border-amber-500/20 px-4 py-3">
                  <Typography.Text className="font-medium">
                    {accessLogPolicyAnomalyKindLabel(it.kind, t)}
                  </Typography.Text>
                </div>
                {rows.length > 0 ? (
                  <div className="p-4 pt-3">
                    <Descriptions
                      column={1}
                      bordered
                      size="small"
                      items={rows.map((row, idx) => ({
                        key: `${it.kind}-${idx}`,
                        label: row.label,
                        children: (
                          <Typography.Text className="break-all font-mono text-sm">
                            {row.value}
                          </Typography.Text>
                        ),
                      }))}
                    />
                  </div>
                ) : null}
              </div>
            );
          })}
        </div>
      ) : null}
    </div>
  );
}
