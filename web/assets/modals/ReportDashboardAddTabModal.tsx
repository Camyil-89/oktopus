"use client";

import { useTranslation } from "@/contexts/LocaleContext";
import { Input, Modal } from "antd";
import { useEffect, useState } from "react";

type ReportDashboardAddTabModalProps = {
  open: boolean;
  onCancel: () => void;
  onConfirm: (title: string) => void;
};

export function ReportDashboardAddTabModal({
  open,
  onCancel,
  onConfirm,
}: ReportDashboardAddTabModalProps) {
  const { t } = useTranslation();
  const [title, setTitle] = useState("");

  useEffect(() => {
    if (open) {
      setTitle("");
    }
  }, [open]);

  const submit = () => {
    const trimmed = title.trim();
    if (!trimmed) {
      return;
    }
    onConfirm(trimmed);
  };

  return (
    <Modal
      title={t("reports.addTab")}
      open={open}
      onCancel={onCancel}
      onOk={submit}
      destroyOnClose
    >
      <Input
        value={title}
        onChange={(e) => setTitle(e.target.value)}
        placeholder={t("reports.tabTitlePlaceholder")}
        onPressEnter={submit}
        autoFocus
      />
    </Modal>
  );
}
