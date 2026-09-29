"use client";

import { Form, Input, Modal } from "antd";
import { scrollableModalProps } from "@/assets/modals/modalConfig";
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

  return (
    <Modal
      title="Новый пользователь"
      open={open}
      okText="Создать"
      cancelText="Отмена"
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
          label="Логин"
          rules={[{ required: true, message: "Введите логин" }]}
        >
          <Input autoComplete="off" />
        </Form.Item>
        <Form.Item
          name="password"
          label="Пароль"
          rules={[{ required: true, message: "Введите пароль" }]}
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

  return (
    <Modal
      title={user ? `Пароль: ${user.username}` : "Смена пароля"}
      open={open}
      okText="Сохранить"
      cancelText="Отмена"
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
          label="Новый пароль"
          rules={[{ required: true, message: "Введите пароль" }]}
        >
          <Input.Password autoComplete="new-password" />
        </Form.Item>
      </Form>
    </Modal>
  );
}
