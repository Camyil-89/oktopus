"use client";

import { ApiError } from "@/api/base";
import { getProxyInspectRule } from "@/api/proxy";
import { ProxyInspectRuleModal } from "@/assets/modals/ProxyInspectRuleModal";
import {
  proxyInspectRuleToFormValues,
  type ProxyInspectRuleFormValues,
} from "@/assets/modals/ProxyInspectRuleFormFields";
import type { ProxyInspectRule } from "@/types/inspect";
import { Form } from "antd";
import { useEffect, useState } from "react";

type ProxyInspectRuleViewModalProps = {
  open: boolean;
  ruleId: string | null;
  initialRule?: ProxyInspectRule | null;
  onClose: () => void;
};

export function ProxyInspectRuleViewModal({
  open,
  ruleId,
  initialRule,
  onClose,
}: ProxyInspectRuleViewModalProps) {
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
    void getProxyInspectRule(ruleId)
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
          setLoadError("Правило не найдено — возможно, его удалили.");
          return;
        }
        setLoadError("Не удалось загрузить правило.");
      })
      .finally(() => {
        if (!cancelled) {
          setLoading(false);
        }
      });
    return () => {
      cancelled = true;
    };
  }, [open, ruleId, initialRule, form]);

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
