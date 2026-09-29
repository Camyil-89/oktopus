"use client";

import { App, Button, Collapse, Typography } from "antd";

type ErrorTraceCollapseProps = {
  trace: string;
  label?: string;
};

export function ErrorTraceCollapse({
  trace,
  label = "Технические подробности",
}: ErrorTraceCollapseProps) {
  const { message } = App.useApp();

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(trace);
      message.success("Скопировано");
    } catch {
      message.error("Не удалось скопировать");
    }
  };

  return (
    <Collapse
      items={[
        {
          key: "trace",
          label,
          children: (
            <div className="flex max-h-[min(50vh,20rem)] flex-col gap-3 overflow-y-auto">
              <Button
                size="small"
                type="default"
                className="w-fit shrink-0"
                onClick={() => void copy()}
              >
                Копировать
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
