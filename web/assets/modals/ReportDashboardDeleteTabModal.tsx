"use client";

import { useTranslation } from "@/contexts/LocaleContext";
import { Modal, Typography } from "antd";

type ReportDashboardDeleteTabModalProps = {
  open: boolean;
  tabTitle: string;
  onCancel: () => void;
  onConfirm: () => void;
};

export function ReportDashboardDeleteTabModal({
  open,
  tabTitle,
  onCancel,
  onConfirm,
}: ReportDashboardDeleteTabModalProps) {
  const { t } = useTranslation();

  return (
    <Modal
      title={t("reports.deleteTabConfirmTitle", { title: tabTitle })}
      open={open}
      onCancel={onCancel}
      onOk={onConfirm}
      okText={t("common.delete")}
      okButtonProps={{ danger: true }}
      cancelText={t("common.cancel")}
      destroyOnClose
    >
      <Typography.Paragraph type="secondary" className="!mb-0">
        {t("reports.deleteTabConfirmBody")}
      </Typography.Paragraph>
    </Modal>
  );
}
