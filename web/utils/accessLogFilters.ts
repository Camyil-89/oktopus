import type { ListProxyAccessLogParams } from "@/types/accessLog";

/** Значение фильтра «Действие» в журнале (совпадает с исходами в колонке). */
export type AccessLogActionFilter = "" | "0" | "1" | "errors";

export function parseAccessLogActionFilter(
  raw: unknown,
): AccessLogActionFilter {
  if (raw === "0" || raw === "1" || raw === "errors") {
    return raw;
  }
  return "";
}

/** Старые сохранённые фильтры с error_kind → action=errors */
export function migrateAccessLogFiltersFromStorage(
  parsed: Record<string, unknown>,
): AccessLogActionFilter {
  const action = parseAccessLogActionFilter(parsed.action);
  if (action) {
    return action;
  }
  const ek = parsed.error_kind;
  if (ek === "any" || ek === "auth" || ek === "inspect" || ek === "gateway") {
    return "errors";
  }
  return "";
}

export function accessLogActionToListParams(
  action: AccessLogActionFilter,
): Pick<ListProxyAccessLogParams, "action" | "error_kind"> {
  if (action === "errors") {
    return { error_kind: "any" };
  }
  if (action === "0" || action === "1") {
    return { action };
  }
  return {};
}
