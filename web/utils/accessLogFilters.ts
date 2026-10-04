import type { ListProxyAccessLogParams } from "@/types/accessLog";
import type { AccessLogSegment } from "@/utils/accessLogSegment";

/** Значение фильтра «Действие» (без errors — см. вкладку). */
export type AccessLogActionFilter = "" | "0" | "1";

export function parseAccessLogActionFilter(
  raw: unknown,
): AccessLogActionFilter {
  if (raw === "0" || raw === "1") {
    return raw;
  }
  return "";
}

/** Старые сохранённые фильтры: error_kind / action=errors → вкладка errors. */
export function migrateAccessLogSegmentFromStorage(
  parsed: Record<string, unknown>,
): AccessLogSegment | null {
  const seg = parsed.segment;
  if (seg === "traffic" || seg === "attacks" || seg === "errors") {
    return seg;
  }
  const action = parsed.action;
  const ek = parsed.error_kind;
  if (action === "errors" || ek === "any" || ek === "auth" || ek === "inspect" || ek === "gateway") {
    return "errors";
  }
  return null;
}

export function accessLogActionToListParams(
  action: AccessLogActionFilter,
): Pick<ListProxyAccessLogParams, "action"> {
  if (action === "0" || action === "1") {
    return { action };
  }
  return {};
}
