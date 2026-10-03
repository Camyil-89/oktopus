"use client";

import { scrollableModalProps } from "@/assets/modals/modalConfig";
import {
  ProxyInspectRuleFormFields,
  type ProxyInspectRuleFormValues,
} from "@/assets/modals/ProxyInspectRuleFormFields";
import { OktopusLoading } from "@/assets/components/oktopus/OktopusLoading";
import { useTranslation } from "@/contexts/LocaleContext";
import { Button, Modal, Typography } from "antd";
import type { FormInstance } from "antd";

export type ProxyInspectRuleModalMode = "view";

type ProxyInspectRuleModalProps = {
  mode: ProxyInspectRuleModalMode;
  open: boolean;
  form: FormInstance<ProxyInspectRuleFormValues>;
  onCancel: () => void;
  loading?: boolean;
  errorMessage?: string | null;
};

export function ProxyInspectRuleModal({
  mode,
  open,
  form,
  onCancel,
  loading,
  errorMessage,
}: ProxyInspectRuleModalProps) {
  const { t } = useTranslation();
  const readOnly = mode === "view";

  return (
    <Modal
      title={t("inspect.viewTitle")}
      open={open}
      onCancel={onCancel}
      footer={
        readOnly
          ? [
              <Button key="close" type="primary" onClick={onCancel}>
                {t("common.close")}
              </Button>,
            ]
          : undefined
      }
      width={720}
      destroyOnHidden
      {...scrollableModalProps}
    >
      {errorMessage ? (
        <Typography.Text type="warning">{errorMessage}</Typography.Text>
      ) : loading ? (
        <div className="flex justify-center py-10">
          <OktopusLoading size="md" />
        </div>
      ) : (
        <ProxyInspectRuleFormFields form={form} readOnly={readOnly} />
      )}
    </Modal>
  );
}
