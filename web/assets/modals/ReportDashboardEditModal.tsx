"use client";

import { scrollableModalProps } from "@/assets/modals/modalConfig";
import { useTranslation } from "@/contexts/LocaleContext";
import type { AccessLogReportWidgetSpec } from "@/types/accessLogReport";
import type { ReportsDashboardTab } from "@/types/reportsDashboard";
import { parseReportWidgetsJson } from "@/utils/parseReportWidgetsJson";
import { App, Input, Modal } from "antd";
import { useEffect, useState } from "react";

type ReportDashboardEditModalProps = {
  open: boolean;
  tab: ReportsDashboardTab | null;
  onCancel: () => void;
  onSubmit: (widgets: AccessLogReportWidgetSpec[]) => void;
};

export function ReportDashboardEditModal({
  open,
  tab,
  onCancel,
  onSubmit,
}: ReportDashboardEditModalProps) {
  const { message } = App.useApp();
  const { t } = useTranslation();
  const [jsonText, setJsonText] = useState("");

  useEffect(() => {
    if (!open || !tab) {
      return;
    }
    setJsonText(JSON.stringify(tab.widgets, null, 2));
  }, [open, tab]);

  const handleOk = () => {
    if (!tab) {
      return;
    }
    try {
      const widgets = parseReportWidgetsJson(jsonText);
      onSubmit(widgets);
    } catch {
      message.error(t("reports.widgetsJsonInvalid"));
    }
  };

  return (
    <Modal
      title={
        tab
          ? t("reports.editDashboardTab", { title: tab.title })
          : t("reports.editDashboard")
      }
      open={open}
      onCancel={onCancel}
      onOk={handleOk}
      destroyOnClose
      width={720}
      okText={t("common.save")}
      {...scrollableModalProps}
    >
      <Input.TextArea
        value={jsonText}
        onChange={(e) => setJsonText(e.target.value)}
        rows={22}
        className="font-mono text-[12px]"
        spellCheck={false}
      />
    </Modal>
  );
}
