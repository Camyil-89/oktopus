"use client";

import { useTranslation } from "@/contexts/LocaleContext";
import { APP_VERSION, GITHUB_REPO_URL } from "@/version";

type AppVersionProps = {
  className?: string;
};

export function AppVersion({ className }: AppVersionProps) {
  const { t } = useTranslation();
  return (
    <p
      className={`flex flex-row flex-wrap items-center justify-center gap-x-1.5 gap-y-0.5 ${className ?? ""}`}
    >
      <span>{t("app.version", { version: APP_VERSION })}</span>
      <span className="text-zinc-700" aria-hidden>
        ·
      </span>
      <a
        href={GITHUB_REPO_URL}
        target="_blank"
        rel="noopener noreferrer"
        className="text-zinc-500 underline-offset-2 hover:text-teal-400/90 hover:underline"
        aria-label={t("app.githubAriaLabel")}
      >
        {t("app.github")}
      </a>
    </p>
  );
}
