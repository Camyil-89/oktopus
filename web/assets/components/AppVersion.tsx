"use client";

import { useTranslation } from "@/contexts/LocaleContext";
import { APP_VERSION } from "@/version";

type AppVersionProps = {
  className?: string;
};

export function AppVersion({ className }: AppVersionProps) {
  const { t } = useTranslation();
  return (
    <p className={className}>
      {t("app.version", { version: APP_VERSION })}
    </p>
  );
}
