"use client";

import { ErrorTraceCollapse } from "@/assets/components/ErrorTraceCollapse";
import { useTranslation } from "@/contexts/LocaleContext";
import { Button, Result } from "antd";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { useMemo } from "react";

export default function NotFoundPage() {
  const pathname = usePathname();
  const { t } = useTranslation();

  const trace = useMemo(() => {
    const payload = {
      type: "NotFound",
      pathname,
      href: typeof window !== "undefined" ? window.location.href : pathname,
      time: new Date().toISOString(),
    };
    return JSON.stringify(payload, null, 2);
  }, [pathname]);

  return (
    <main className="grid-bg flex min-h-full flex-1 flex-col items-center justify-center gap-6 bg-ink p-4 md:p-8">
      <Result
        status="404"
        title={t("notFound.title")}
        extra={[
          <Link key="home" href="/">
            <Button type="primary">{t("notFound.home")}</Button>
          </Link>,
          <Link key="login" href="/login">
            <Button>{t("notFound.signIn")}</Button>
          </Link>,
        ]}
        className="max-w-2xl"
      />
      <div className="w-full max-w-2xl">
        <ErrorTraceCollapse trace={trace} />
      </div>
    </main>
  );
}
