"use client";

import { Form, Input, Modal } from "antd";
import { scrollableModalProps } from "@/assets/modals/modalConfig";
import { useTranslation } from "@/contexts/LocaleContext";
import type { User } from "@/types/user";

type CreateUserModalProps = {
  open: boolean;
  loading: boolean;
  onCancel: () => void;
  onSubmit: (values: { username: string; password: string }) => void;
};

export function CreateUserModal({
  open,
  loading,
  onCancel,
  onSubmit,
}: CreateUserModalProps) {
  const [form] = Form.useForm();
  const { t } = useTranslation();

  return (
    <Modal
      title={t("users.newUser")}
      open={open}
      okText={t("common.create")}
      cancelText={t("common.cancel")}
      confirmLoading={loading}
      onCancel={() => {
        form.resetFields();
        onCancel();
      }}
      onOk={() => {
        form.validateFields().then((values) => {
          onSubmit(values);
        });
      }}
      {...scrollableModalProps}
    >
      <Form form={form} layout="vertical">
        <Form.Item
          name="username"
          label={t("common.username")}
          rules={[{ required: true, message: t("common.enterUsername") }]}
        >
          <Input autoComplete="off" />
        </Form.Item>
        <Form.Item
          name="password"
          label={t("common.password")}
          rules={[{ required: true, message: t("common.enterPassword") }]}
        >
          <Input.Password autoComplete="new-password" />
        </Form.Item>
      </Form>
    </Modal>
  );
}

type ChangePasswordModalProps = {
  open: boolean;
  loading: boolean;
  user: User | null;
  onCancel: () => void;
  onSubmit: (values: { password: string }) => void;
};

export function ChangePasswordModal({
  open,
  loading,
  user,
  onCancel,
  onSubmit,
}: ChangePasswordModalProps) {
  const [form] = Form.useForm();
  const { t } = useTranslation();

  return (
    <Modal
      title={
        user
          ? t("users.passwordTitle", { username: user.username })
          : t("users.changePassword")
      }
      open={open}
      okText={t("common.save")}
      cancelText={t("common.cancel")}
      confirmLoading={loading}
      onCancel={() => {
        form.resetFields();
        onCancel();
      }}
      onOk={() => {
        form.validateFields().then((values) => {
          onSubmit(values);
        });
      }}
      afterOpenChange={(visible) => {
        if (!visible) {
          form.resetFields();
        }
      }}
      {...scrollableModalProps}
    >
      <Form form={form} layout="vertical">
        <Form.Item
          name="password"
          label={t("users.newPassword")}
          rules={[{ required: true, message: t("common.enterPassword") }]}
        >
          <Input.Password autoComplete="new-password" />
        </Form.Item>
      </Form>
    </Modal>
  );
}
