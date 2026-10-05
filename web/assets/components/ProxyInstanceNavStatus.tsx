"use client";

import type { ProxyInstanceRuntimeStatus } from "@/types/proxy";
import { useTranslation } from "@/contexts/LocaleContext";
import type { TranslateFn } from "@/i18n/translate";
import { Tooltip } from "antd";

type StatusTone = "pending" | "success" | "warning" | "danger";

function aclSummaryLabel(
  st: ProxyInstanceRuntimeStatus["acl"] | undefined,
  t: TranslateFn,
): string {
  switch (st?.build_status) {
    case "building":
      return t("dashboard.aclBuilding");
    case "error":
      return t("dashboard.aclError");
    case "ready":
      return "OK";
    default:
      return t("dashboard.aclPending");
  }
}

function resolveTone(runtime: ProxyInstanceRuntimeStatus | undefined): StatusTone {
  if (!runtime) {
    return "pending";
  }
  if (!runtime.active) {
    return "danger";
  }
  switch (runtime.acl?.build_status) {
    case "error":
      return "danger";
    case "building":
      return "warning";
    case "ready":
      return "success";
    default:
      return "warning";
  }
}

const dotClass: Record<StatusTone, string> = {
  pending: "bg-zinc-500",
  success: "bg-teal-400",
  warning: "bg-amber-400",
  danger: "bg-red-400",
};

type ProxyInstanceNavStatusProps = {
  runtime: ProxyInstanceRuntimeStatus | undefined;
  listenFallback?: string;
};

export function ProxyInstanceNavStatus({
  runtime,
  listenFallback,
}: ProxyInstanceNavStatusProps) {
  const { t } = useTranslation();
  const tone = resolveTone(runtime);
  const listen = runtime?.listen ?? listenFallback ?? "";

  const proxyLine = runtime
    ? runtime.active
      ? t("nav.instanceStatusRunning")
      : t("nav.instanceStatusStopped")
    : t("instance.runtimePending");

  const tooltipLines = [proxyLine];
  if (listen) {
    tooltipLines.push(listen);
  }
  if (runtime) {
    tooltipLines.push(`${t("dashboard.aclBuild")}: ${aclSummaryLabel(runtime.acl, t)}`);
    if (runtime.proxy_start_error) {
      tooltipLines.push(runtime.proxy_start_error);
    }
  }

  return (
    <Tooltip title={<span className="font-mono text-[11px]">{tooltipLines.join(" · ")}</span>}>
      <span
        className={`h-2 w-2 shrink-0 rounded-full ${dotClass[tone]} ${tone === "warning" ? "animate-pulse" : ""}`}
        aria-hidden
      />
    </Tooltip>
  );
}
