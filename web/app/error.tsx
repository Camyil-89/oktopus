"use client";

import { ErrorTraceCollapse } from "@/assets/components/ErrorTraceCollapse";
import { useTranslation } from "@/contexts/LocaleContext";
import { formatErrorTrace } from "@/utils/formatErrorTrace";
import { Button, Result, Typography } from "antd";
import { useEffect, useMemo } from "react";

export default function ErrorPage({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  const { t } = useTranslation();

  useEffect(() => {
    console.error(error);
  }, [error]);

  const trace = useMemo(
    () =>
      formatErrorTrace(error, {
        url: typeof window !== "undefined" ? window.location.href : undefined,
        time: new Date().toISOString(),
      }),
    [error],
  );

  return (
    <main className="grid-bg flex min-h-full flex-1 flex-col items-center justify-center bg-ink p-4 md:p-8">
      <div className="flex w-full max-w-2xl flex-col gap-6">
        <Result
          status="error"
          title={t("errorPage.title")}
          subTitle={
            <div className="flex flex-col gap-4">
              <Typography.Text type="secondary">{error.message}</Typography.Text>
              <ErrorTraceCollapse trace={trace} />
            </div>
          }
          extra={[
            <Button key="retry" type="primary" onClick={() => reset()}>
              {t("errorPage.retry")}
            </Button>,
            <Button key="home" href="/">
              {t("errorPage.home")}
            </Button>,
          ]}
        />
      </div>
    </main>
  );
}
