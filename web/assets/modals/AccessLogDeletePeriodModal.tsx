"use client";

import { scrollableModalProps } from "@/assets/modals/modalConfig";
import { DatePicker, Form, Modal } from "antd";
import type { Dayjs } from "dayjs";

type FormValues = {
  period: [Dayjs, Dayjs];
};

type AccessLogDeletePeriodModalProps = {
  open: boolean;
  loading: boolean;
  onCancel: () => void;
  onSubmit: (values: { from: string; to: string }) => void;
};

export function AccessLogDeletePeriodModal({
  open,
  loading,
  onCancel,
  onSubmit,
}: AccessLogDeletePeriodModalProps) {
  const [form] = Form.useForm<FormValues>();

  return (
    <Modal
      title="Удалить записи за период"
      open={open}
      okText="Удалить"
      okType="danger"
      cancelText="Отмена"
      confirmLoading={loading}
      onCancel={() => {
        form.resetFields();
        onCancel();
      }}
      onOk={() => {
        form.validateFields().then((values) => {
          const [from, to] = values.period;
          onSubmit({
            from: from.toDate().toISOString(),
            to: to.toDate().toISOString(),
          });
        });
      }}
      {...scrollableModalProps}
    >
      <Form form={form} layout="vertical">
        <Form.Item
          name="period"
          label="Период"
          rules={[{ required: true, message: "Выберите период" }]}
        >
          <DatePicker.RangePicker
            showTime
            className="w-full"
            format="DD.MM.YYYY HH:mm:ss"
          />
        </Form.Item>
      </Form>
    </Modal>
  );
}
