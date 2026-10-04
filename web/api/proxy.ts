import { API_BASE, ApiError, fetchApi, fetchApiForm } from "./base";
import type {
  DeleteProxyAccessLogByPeriodPayload,
  DeleteProxyAccessLogByPeriodResponse,
  ListProxyAccessLogParams,
  ProxyAccessLogListResponse,
} from "@/types/accessLog";
import type {
  AccessLogReportResponse,
  AccessLogReportSpec,
  AccessLogReportTableRequest,
  AccessLogReportWidgetResult,
} from "@/types/accessLogReport";
import type {
  ProxyInspectCompileStatus,
  ProxyInspectRule,
  ProxyInspectRuleDraft,
  ProxyInspectRuleListItem,
  ValidateProxyInspectScriptPayload,
  ValidateProxyInspectScriptResponse,
} from "@/types/inspect";
import type {
  ProxyACLCompileStatus,
  ProxyACLEvaluateRequest,
  ProxyACLEvaluateResult,
  GenerateCARequest,
  ProxyCAStatus,
  ProxyErrorPagePreviewVariant,
  ProxyForbiddenPageStatus,
  ProxyGatewayPageStatus,
  ProxyRuntimeStatus,
  ProxySettings,
  ProxySettingsPatch,
} from "@/types/proxy";

export async function getProxySettings() {
  return fetchApi<ProxySettings>("/api/proxy/settings", "GET");
}

export async function patchProxySettings(body: ProxySettingsPatch) {
  return fetchApi<ProxySettings>("/api/proxy/settings", "PATCH", body);
}

export async function getProxyCAStatus() {
  return fetchApi<ProxyCAStatus>("/api/proxy/ca/status", "GET");
}

export async function generateProxyCA(body: GenerateCARequest) {
  return fetchApi<ProxySettings>("/api/proxy/ca/generate", "POST", body);
}

export async function uploadProxyCA(form: FormData) {
  return fetchApiForm<ProxySettings>("/api/proxy/ca/upload", "POST", form);
}

export async function getProxyForbiddenPageStatus() {
  return fetchApi<ProxyForbiddenPageStatus>("/api/proxy/forbidden/status", "GET");
}

export async function uploadProxyForbiddenPage(form: FormData) {
  return fetchApiForm<ProxyForbiddenPageStatus>("/api/proxy/forbidden/upload", "POST", form);
}

export async function clearProxyForbiddenPage() {
  return fetchApi<ProxyForbiddenPageStatus>("/api/proxy/forbidden/custom", "DELETE");
}

export async function getProxyGatewayPageStatus() {
  return fetchApi<ProxyGatewayPageStatus>("/api/proxy/gateway/status", "GET");
}

export async function uploadProxyGatewayPage(form: FormData) {
  return fetchApiForm<ProxyGatewayPageStatus>("/api/proxy/gateway/upload", "POST", form);
}

export async function clearProxyGatewayPage() {
  return fetchApi<ProxyGatewayPageStatus>("/api/proxy/gateway/custom", "DELETE");
}

export async function openProxyErrorPagePreview(
  kind: "forbidden" | "gateway",
  variant: ProxyErrorPagePreviewVariant,
) {
  const q = new URLSearchParams({ variant });
  const res = await fetch(`${API_BASE}/api/proxy/${kind}/preview?${q}`, {
    method: "GET",
    credentials: "include",
  });
  if (!res.ok) {
    const text = await res.text();
    let msg = res.statusText;
    try {
      const j = JSON.parse(text) as { error?: string };
      if (j.error) msg = j.error;
    } catch {
      if (text) msg = text;
    }
    throw new ApiError(res.status, msg);
  }
  const blob = await res.blob();
  const url = URL.createObjectURL(blob);
  window.open(url, "_blank", "noopener,noreferrer");
  window.setTimeout(() => URL.revokeObjectURL(url), 60_000);
}

async function downloadFile(path: string, filename: string) {
  const res = await fetch(`${API_BASE}${path}`, {
    method: "GET",
    credentials: "include",
  });
  if (!res.ok) {
    const text = await res.text();
    let msg = res.statusText;
    try {
      const j = JSON.parse(text) as { error?: string };
      if (j.error) msg = j.error;
    } catch {
      if (text) msg = text;
    }
    throw new ApiError(res.status, msg);
  }
  const blob = await res.blob();
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  a.click();
  URL.revokeObjectURL(url);
}

export function downloadProxyCACert() {
  return downloadFile("/api/proxy/ca/cert", "ca.crt");
}

export function downloadProxyCAKey() {
  return downloadFile("/api/proxy/ca/key", "ca.key");
}

export async function getProxyRuntimeStatus() {
  return fetchApi<ProxyRuntimeStatus>("/api/proxy/status", "GET");
}

export async function getProxyACLPolicy() {
  return fetchApi<import("@/types/proxy").ProxyACLPolicy>(
    "/api/proxy/acl/policy",
    "GET",
  );
}

export async function putProxyACLPolicy(configText: string) {
  return fetchApi<import("@/types/proxy").ProxyACLPolicy>(
    "/api/proxy/acl/policy",
    "PUT",
    { config_text: configText },
  );
}

export async function validateProxyACLPolicy(
  configText: string,
  lists?: import("@/types/proxy").ProxyACLNamedListDraft[],
) {
  const body: {
    config_text: string;
    lists?: import("@/types/proxy").ProxyACLNamedListDraft[];
  } = { config_text: configText };
  if (lists !== undefined) {
    body.lists = lists;
  }
  return fetchApi<import("@/types/proxy").ProxyACLPolicyAnalyzeResult>(
    "/api/proxy/acl/policy/validate",
    "POST",
    body,
  );
}

export async function listProxyACLNamedLists() {
  return fetchApi<import("@/types/proxy").ProxyACLNamedListSummary[]>(
    "/api/proxy/acl/lists",
    "GET",
  );
}

export async function getProxyACLNamedList(id: string) {
  return fetchApi<import("@/types/proxy").ProxyACLNamedList>(
    `/api/proxy/acl/lists/${encodeURIComponent(id)}`,
    "GET",
  );
}

export async function syncProxyACLNamedLists(
  lists: import("@/types/proxy").ProxyACLNamedListDraft[],
) {
  return fetchApi<import("@/types/proxy").ProxyACLNamedListSummary[]>(
    "/api/proxy/acl/lists",
    "PUT",
    { lists },
  );
}

export async function pollProxyACLNamedList(
  id: string,
  options?: { source_url?: string },
) {
  const body =
    options?.source_url?.trim()
      ? { source_url: options.source_url.trim() }
      : undefined;
  return fetchApi<import("@/types/proxy").ProxyACLNamedList>(
    `/api/proxy/acl/lists/${encodeURIComponent(id)}/poll`,
    "POST",
    body,
  );
}

export async function getProxyACLStatus() {
  return fetchApi<ProxyACLCompileStatus>("/api/proxy/acl/status", "GET");
}

export async function evaluateProxyACL(body: ProxyACLEvaluateRequest) {
  return fetchApi<ProxyACLEvaluateResult>(
    "/api/proxy/acl/evaluate",
    "POST",
    body,
  );
}

export async function getProxyInspectStatus() {
  return fetchApi<ProxyInspectCompileStatus>("/api/proxy/inspect/status", "GET");
}

export async function listProxyInspectRules() {
  return fetchApi<ProxyInspectRule[]>("/api/proxy/inspect/rules", "GET");
}

export async function getProxyInspectRule(id: string) {
  return fetchApi<ProxyInspectRule>(
    `/api/proxy/inspect/rules/${encodeURIComponent(id)}`,
    "GET",
  );
}

export async function syncProxyInspectRules(rules: ProxyInspectRuleDraft[]) {
  return fetchApi<ProxyInspectRuleListItem[]>(
    "/api/proxy/inspect/rules",
    "PUT",
    { rules },
  );
}

export async function validateProxyInspectScript(
  body: ValidateProxyInspectScriptPayload,
) {
  return fetchApi<ValidateProxyInspectScriptResponse>(
    "/api/proxy/inspect/validate",
    "POST",
    body,
  );
}

function buildAccessLogQuery(params: ListProxyAccessLogParams) {
  const q = new URLSearchParams();
  if (params.page) q.set("page", String(params.page));
  if (params.page_size) q.set("page_size", String(params.page_size));
  if (params.id) q.set("id", params.id);
  if (params.user) q.set("user", params.user);
  if (params.source) q.set("source", params.source);
  if (params.destination) q.set("destination", params.destination);
  if (params.url) q.set("url", params.url);
  if (params.search_only) q.set("search_only", "true");
  if (params.from) q.set("from", params.from);
  if (params.to) q.set("to", params.to);
  if (params.action === "0" || params.action === "1") {
    q.set("action", params.action);
  }
  if (params.error_kind) {
    q.set("error_kind", params.error_kind);
  }
  if (params.segment) {
    q.set("segment", params.segment);
  }
  if (params.attack_kind) {
    q.set("attack_kind", params.attack_kind);
  }
  if (params.policy_anomaly_q) {
    q.set("policy_anomaly_q", params.policy_anomaly_q);
  }
  if (params.decision_rule_ref) {
    q.set("decision_rule_ref", params.decision_rule_ref);
  }
  if (params.inspect_rule_id) {
    q.set("inspect_rule_id", params.inspect_rule_id);
  }
  const s = q.toString();
  return s ? `?${s}` : "";
}

export async function listProxyAccessLog(params: ListProxyAccessLogParams) {
  return fetchApi<ProxyAccessLogListResponse>(
    `/api/proxy/access-log${buildAccessLogQuery(params)}`,
    "GET",
  );
}

export async function deleteProxyAccessLogByPeriod(
  body: DeleteProxyAccessLogByPeriodPayload,
) {
  return fetchApi<DeleteProxyAccessLogByPeriodResponse>(
    "/api/proxy/access-log/delete",
    "POST",
    body,
  );
}

export async function runProxyAccessLogReport(spec: AccessLogReportSpec) {
  return fetchApi<AccessLogReportResponse>(
    "/api/proxy/access-log/report",
    "POST",
    spec,
  );
}

export async function runProxyAccessLogReportTable(
  body: AccessLogReportTableRequest,
) {
  return fetchApi<AccessLogReportWidgetResult>(
    "/api/proxy/access-log/report/table",
    "POST",
    body,
  );
}
