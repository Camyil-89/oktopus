"use client";

import { scrollableModalProps } from "@/assets/modals/modalConfig";
import { useTranslation } from "@/contexts/LocaleContext";
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
  const { t } = useTranslation();

  return (
    <Modal
      title={t("accessLog.deletePeriodTitle")}
      open={open}
      okText={t("common.delete")}
      okType="danger"
      cancelText={t("common.cancel")}
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
          label={t("common.period")}
          rules={[{ required: true, message: t("accessLog.selectPeriod") }]}
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
