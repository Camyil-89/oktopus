"use client";

import { getProxySettings } from "@/api/proxy";
import { scrollableModalProps } from "@/assets/modals/modalConfig";
import { useApiErrorMessage, useTranslation } from "@/contexts/LocaleContext";
import { App, Checkbox, Form, InputNumber, Modal } from "antd";
import { useEffect, useState } from "react";

type FormValues = {
  unlimited: boolean;
  days: number;
};

type AccessLogRetentionModalProps = {
  open: boolean;
  saving: boolean;
  onCancel: () => void;
  onSubmit: (values: { access_log_retention_days: number }) => void;
};

export function AccessLogRetentionModal({
  open,
  saving,
  onCancel,
  onSubmit,
}: AccessLogRetentionModalProps) {
  const { message } = App.useApp();
  const { t } = useTranslation();
  const formatApiError = useApiErrorMessage();
  const [form] = Form.useForm<FormValues>();
  const [loading, setLoading] = useState(false);
  const unlimited = Form.useWatch("unlimited", form);

  useEffect(() => {
    if (!open) {
      return;
    }
    let cancelled = false;
    setLoading(true);
    void (async () => {
      try {
        const st = await getProxySettings();
        if (cancelled) return;
        const days = st.access_log_retention_days;
        form.setFieldsValue({
          unlimited: days === 0,
          days: days === 0 ? 3 : days,
        });
      } catch (e) {
        if (!cancelled) {
          message.error(formatApiError(e, t("accessLog.retentionLoadFailed")));
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [open, form, formatApiError, message, t]);

  return (
    <Modal
      title={t("accessLog.retentionTitle")}
      open={open}
      okText={t("common.save")}
      cancelText={t("common.cancel")}
      confirmLoading={saving}
      loading={loading}
      onCancel={() => {
        form.resetFields();
        onCancel();
      }}
      onOk={() => {
        form.validateFields().then((values) => {
          onSubmit({
            access_log_retention_days: values.unlimited ? 0 : values.days,
          });
        });
      }}
      {...scrollableModalProps}
    >
      <Form form={form} layout="vertical" initialValues={{ unlimited: false, days: 3 }}>
        <Form.Item name="unlimited" valuePropName="checked">
          <Checkbox>{t("accessLog.retentionUnlimited")}</Checkbox>
        </Form.Item>
        {!unlimited && (
          <Form.Item
            name="days"
            label={t("accessLog.retentionDays")}
            rules={[
              { required: true, message: t("accessLog.retentionRequired") },
              {
                type: "number",
                min: 1,
                message: t("accessLog.retentionMin"),
              },
            ]}
          >
            <InputNumber min={1} className="w-full" />
          </Form.Item>
        )}
      </Form>
    </Modal>
  );
}
