"use client";

import { ApiError } from "@/api/base";
import { findProxyInspectRule, getProxyInspectRule } from "@/api/proxy";
import { ProxyInspectRuleModal } from "@/assets/modals/ProxyInspectRuleModal";
import {
  proxyInspectRuleToFormValues,
  type ProxyInspectRuleFormValues,
} from "@/assets/modals/ProxyInspectRuleFormFields";
import type { ProxyInspectRule } from "@/types/inspect";
import { useTranslation } from "@/contexts/LocaleContext";
import { Form } from "antd";
import { useEffect, useState } from "react";

type ProxyInspectRuleViewModalProps = {
  open: boolean;
  ruleId: string | null;
  /** Если не задан — правило ищется по всем инстансам. */
  instanceId?: string;
  initialRule?: ProxyInspectRule | null;
  onClose: () => void;
};

export function ProxyInspectRuleViewModal({
  open,
  ruleId,
  instanceId,
  initialRule,
  onClose,
}: ProxyInspectRuleViewModalProps) {
  const { t } = useTranslation();
  const [form] = Form.useForm<ProxyInspectRuleFormValues>();
  const [loading, setLoading] = useState(false);
  const [loadError, setLoadError] = useState<string | null>(null);

  useEffect(() => {
    if (!open || !ruleId) {
      setLoadError(null);
      setLoading(false);
      return;
    }
    if (initialRule && initialRule.id === ruleId) {
      form.setFieldsValue(proxyInspectRuleToFormValues(initialRule));
      setLoadError(null);
      setLoading(false);
      return;
    }
    let cancelled = false;
    setLoading(true);
    setLoadError(null);
    const load = instanceId
      ? () => getProxyInspectRule(instanceId, ruleId)
      : () => findProxyInspectRule(ruleId);
    void load()
      .then((r) => {
        if (!cancelled) {
          form.setFieldsValue(proxyInspectRuleToFormValues(r));
        }
      })
      .catch((e) => {
        if (cancelled) {
          return;
        }
        if (e instanceof ApiError && e.status === 404) {
          setLoadError(t("inspect.ruleNotFound"));
          return;
        }
        setLoadError(t("inspect.ruleLoadFailed"));
      })
      .finally(() => {
        if (!cancelled) {
          setLoading(false);
        }
      });
    return () => {
      cancelled = true;
    };
  }, [open, ruleId, instanceId, initialRule, form, t]);

  return (
    <ProxyInspectRuleModal
      mode="view"
      open={open}
      form={form}
      onCancel={onClose}
      loading={loading}
      errorMessage={loadError}
    />
  );
}
