"use client";

import { ApiError } from "@/api/base";
import { createProxyInstance } from "@/api/proxy";
import { scrollableModalProps } from "@/assets/modals/modalConfig";
import { useApiErrorMessage, useTranslation } from "@/contexts/LocaleContext";
import { Form, Input, Modal } from "antd";
import { useState } from "react";

export const PROXY_INSTANCES_CHANGED_EVENT = "oktopus:proxy-instances-changed";

type FormValues = {
  name: string;
  listen: string;
};

type CreateProxyInstanceModalProps = {
  open: boolean;
  onClose: () => void;
  onCreated?: () => void;
};

export function CreateProxyInstanceModal({
  open,
  onClose,
  onCreated,
}: CreateProxyInstanceModalProps) {
  const { t } = useTranslation();
  const formatApiError = useApiErrorMessage();
  const [form] = Form.useForm<FormValues>();
  const [saving, setSaving] = useState(false);

  const submit = async () => {
    const values = await form.validateFields();
    setSaving(true);
    try {
      await createProxyInstance({
        name: values.name.trim(),
        listen: values.listen.trim(),
      });
      window.dispatchEvent(new Event(PROXY_INSTANCES_CHANGED_EVENT));
      onCreated?.();
      form.resetFields();
      onClose();
    } catch (e) {
      if (e instanceof ApiError) {
        const code = e.message;
        if (code === "instance_name_required") {
          form.setFields([{ name: "name", errors: [t("instance.nameRequired")] }]);
          return;
        }
        if (code === "instance_name_taken") {
          form.setFields([{ name: "name", errors: [t("instance.nameTaken")] }]);
          return;
        }
        if (code === "listen_address_in_use") {
          form.setFields([{ name: "listen", errors: [t("instance.listenInUse")] }]);
          return;
        }
      }
      form.setFields([
        {
          name: "name",
          errors: [formatApiError(e, t("instance.createFailed"))],
        },
      ]);
    } finally {
      setSaving(false);
    }
  };

  return (
    <Modal
      title={t("instance.createTitle")}
      open={open}
      onCancel={onClose}
      onOk={() => void submit()}
      confirmLoading={saving}
      okText={t("instance.createSubmit")}
      cancelText={t("common.cancel")}
      {...scrollableModalProps}
    >
      <Form
        form={form}
        layout="vertical"
        initialValues={{ listen: ":8080" }}
        className="pt-2"
      >
        <Form.Item
          name="name"
          label={t("instance.nameLabel")}
          rules={[{ required: true, message: t("instance.nameRequired") }]}
        >
          <Input autoComplete="off" />
        </Form.Item>
        <Form.Item
          name="listen"
          label={t("instance.listenLabel")}
          rules={[{ required: true, message: t("instance.listenRequired") }]}
        >
          <Input placeholder=":8080" autoComplete="off" />
        </Form.Item>
      </Form>
    </Modal>
  );
}
