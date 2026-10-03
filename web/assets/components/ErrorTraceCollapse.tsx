"use client";

import { App, Button, Collapse, Typography } from "antd";
import { useTranslation } from "@/contexts/LocaleContext";

type ErrorTraceCollapseProps = {
  trace: string;
  label?: string;
};

export function ErrorTraceCollapse({
  trace,
  label,
}: ErrorTraceCollapseProps) {
  const { message } = App.useApp();
  const { t } = useTranslation();
  const collapseLabel = label ?? t("errorTrace.label");

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(trace);
      message.success(t("common.copied"));
    } catch {
      message.error(t("common.copyFailed"));
    }
  };

  return (
    <Collapse
      items={[
        {
          key: "trace",
          label: collapseLabel,
          children: (
            <div className="flex max-h-[min(50vh,20rem)] flex-col gap-3 overflow-y-auto">
              <Button
                size="small"
                type="default"
                className="w-fit shrink-0"
                onClick={() => void copy()}
              >
                {t("common.copy")}
              </Button>
              <Typography.Paragraph
                copyable={{ text: trace }}
                className="!mb-0 font-mono text-xs whitespace-pre-wrap break-all"
              >
                {trace}
              </Typography.Paragraph>
            </div>
          ),
        },
      ]}
    />
  );
}
