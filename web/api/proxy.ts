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

function inst(instanceId: string, path: string) {
  return `/api/proxy/instances/${encodeURIComponent(instanceId)}${path}`;
}

export async function getHostSettings() {
  return fetchApi<import("@/types/proxy").HostSettings>("/api/proxy/host-settings", "GET");
}

export async function patchHostSettings(body: import("@/types/proxy").HostSettingsPatch) {
  return fetchApi<import("@/types/proxy").HostSettings>("/api/proxy/host-settings", "PATCH", body);
}

export async function listProxyInstances() {
  const list = await fetchApi<import("@/types/proxy").ProxyInstance[] | null>(
    "/api/proxy/instances",
    "GET",
  );
  return list ?? [];
}

export async function createProxyInstance(body: { name: string; listen: string }) {
  return fetchApi<import("@/types/proxy").ProxyInstance>("/api/proxy/instances", "POST", body);
}

export async function getProxyInstance(instanceId: string) {
  return fetchApi<import("@/types/proxy").ProxyInstance>(inst(instanceId, ""), "GET");
}

export async function patchProxyInstance(
  instanceId: string,
  body: import("@/types/proxy").ProxyInstancePatch,
) {
  return fetchApi<import("@/types/proxy").ProxyInstance>(inst(instanceId, ""), "PATCH", body);
}

export async function deleteProxyInstance(instanceId: string) {
  return fetchApi<void>(inst(instanceId, ""), "DELETE");
}

export async function getProxyCAStatus(instanceId: string) {
  return fetchApi<ProxyCAStatus>(inst(instanceId, "/ca/status"), "GET");
}

export async function generateProxyCA(instanceId: string, body: GenerateCARequest) {
  return fetchApi<import("@/types/proxy").ProxyInstance>(
    inst(instanceId, "/ca/generate"),
    "POST",
    body,
  );
}

export async function uploadProxyCA(instanceId: string, form: FormData) {
  return fetchApiForm<import("@/types/proxy").ProxyInstance>(
    inst(instanceId, "/ca/upload"),
    "POST",
    form,
  );
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

export function downloadProxyCACert(instanceId: string) {
  return downloadFile(inst(instanceId, "/ca/cert"), "ca.crt");
}

export function downloadProxyCAKey(instanceId: string) {
  return downloadFile(inst(instanceId, "/ca/key"), "ca.key");
}

export async function getProxyRuntimeStatus() {
  return fetchApi<ProxyRuntimeStatus>("/api/proxy/status", "GET");
}

export async function getProxyACLPolicy(instanceId: string) {
  return fetchApi<import("@/types/proxy").ProxyACLPolicy>(
    inst(instanceId, "/acl/policy"),
    "GET",
  );
}

export async function putProxyACLPolicy(instanceId: string, configText: string) {
  return fetchApi<import("@/types/proxy").ProxyACLPolicy>(
    inst(instanceId, "/acl/policy"),
    "PUT",
    { config_text: configText },
  );
}

export async function validateProxyACLPolicy(
  instanceId: string,
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
    inst(instanceId, "/acl/policy/validate"),
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

export async function getProxyACLStatus(instanceId: string) {
  return fetchApi<ProxyACLCompileStatus>(inst(instanceId, "/acl/status"), "GET");
}

export async function evaluateProxyACL(instanceId: string, body: ProxyACLEvaluateRequest) {
  return fetchApi<ProxyACLEvaluateResult>(
    inst(instanceId, "/acl/evaluate"),
    "POST",
    body,
  );
}

export async function getProxyInspectStatus(instanceId: string) {
  return fetchApi<ProxyInspectCompileStatus>(inst(instanceId, "/inspect/status"), "GET");
}

export async function listProxyInspectRules(instanceId: string) {
  return fetchApi<ProxyInspectRule[]>(inst(instanceId, "/inspect/rules"), "GET");
}

/** Все правила инспекции по инстансам (глобальный журнал / отчёты). */
export async function listAllProxyInspectRules() {
  const instances = await listProxyInstances();
  const batches = await Promise.all(
    instances.map((i) =>
      listProxyInspectRules(i.id).catch(() => [] as ProxyInspectRule[]),
    ),
  );
  return batches.flat();
}

export async function getProxyInspectRule(instanceId: string, ruleId: string) {
  return fetchApi<ProxyInspectRule>(
    `${inst(instanceId, "/inspect/rules")}/${encodeURIComponent(ruleId)}`,
    "GET",
  );
}

/** Ищет правило по id среди всех инстансов (глобальный журнал / отчёты). */
export async function findProxyInspectRule(ruleId: string) {
  const instances = await listProxyInstances();
  for (const instRow of instances) {
    try {
      const rule = await getProxyInspectRule(instRow.id, ruleId);
      return rule;
    } catch (e) {
      if (e instanceof ApiError && e.status === 404) {
        continue;
      }
      throw e;
    }
  }
  throw new ApiError(404, "inspect rule not found");
}

export async function syncProxyInspectRules(
  instanceId: string,
  rules: ProxyInspectRuleDraft[],
) {
  return fetchApi<ProxyInspectRuleListItem[]>(
    inst(instanceId, "/inspect/rules"),
    "PUT",
    { rules },
  );
}

export async function validateProxyInspectScript(
  instanceId: string,
  body: ValidateProxyInspectScriptPayload,
) {
  return fetchApi<ValidateProxyInspectScriptResponse>(
    inst(instanceId, "/inspect/validate"),
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
  if (params.instance_id) {
    q.set("instance_id", params.instance_id);
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
