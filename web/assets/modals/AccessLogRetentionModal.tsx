"use client";

import { ApiError } from "@/api/base";
import { getProxySettings } from "@/api/proxy";
import { scrollableModalProps } from "@/assets/modals/modalConfig";
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
          const msg =
            e instanceof ApiError
              ? e.message
              : "Не удалось загрузить настройки";
          message.error(msg);
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
  }, [open, form, message]);

  return (
    <Modal
      title="Хранение записей"
      open={open}
      okText="Сохранить"
      cancelText="Отмена"
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
          <Checkbox>Без ограничения по времени</Checkbox>
        </Form.Item>
        {!unlimited && (
          <Form.Item
            name="days"
            label="Дней хранения"
            rules={[
              { required: true, message: "Укажите срок" },
              {
                type: "number",
                min: 1,
                message: "Минимум 1 день",
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
