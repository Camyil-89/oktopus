"use client";

import * as proxyApi from "@/api/proxy";
import { DashboardStatCard } from "@/assets/components/dashboard/DashboardStatCard";
import type { DashboardStatTone } from "@/assets/components/dashboard/DashboardStatCard";
import { DashboardRuntimePanel } from "@/assets/components/dashboard/DashboardRuntimePanel";
import { DashboardTrafficChartsSection } from "@/assets/components/dashboard/DashboardTrafficChartsSection";
import type {
  ProxyACLCompileStatus,
  ProxyInstanceRuntimeStatus,
  ProxyRuntimeStatus,
} from "@/types/proxy";
import { useApiErrorMessage, useTranslation } from "@/contexts/LocaleContext";
import type { TranslateFn } from "@/i18n/translate";
import { OktopusLoading } from "@/assets/components/oktopus/OktopusLoading";
import { mergeTrafficBuckets } from "@/utils/mergeTrafficBuckets";
import {
  averageInstanceTraffic,
  fleetAclSummary,
} from "@/utils/fleetDashboardMetrics";
import { CreateProxyInstanceModal } from "@/assets/modals/CreateProxyInstanceModal";
import { PlusOutlined } from "@ant-design/icons";
import { Alert, Button, Typography } from "antd";
import { ManageFleetInstanceBlock } from "@/assets/components/dashboard/ManageFleetInstanceBlock";
import { PROXY_INSTANCES_CHANGED_EVENT } from "@/assets/modals/CreateProxyInstanceModal";
import Link from "next/link";
import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  startTransition,
} from "react";
import { runtimeStatusFingerprint } from "@/utils/runtimeStatusFingerprint";
import type { ProxyInstance } from "@/types/proxy";

function aclBuildLabel(st: ProxyACLCompileStatus, t: TranslateFn): string {
  switch (st.build_status) {
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

function aclBuildTone(st: ProxyACLCompileStatus): DashboardStatTone {
  switch (st.build_status) {
    case "ready":
      return "success";
    case "error":
      return "danger";
    case "building":
      return "warning";
    default:
      return "neutral";
  }
}

function formatSharePct(part: number, total: number): string {
  if (total <= 0) {
    return "—";
  }
  const pct = (part / total) * 100;
  if (pct >= 100) {
    return "100%";
  }
  if (pct <= 0) {
    return "0%";
  }
  return pct < 10 ? `${pct.toFixed(1)}%` : `${Math.round(pct)}%`;
}

function formatInt(n: number, localeTag: string): string {
  return new Intl.NumberFormat(localeTag).format(n);
}

function formatStartupClock(iso?: string, localeTag = "ru-RU"): string {
  const emDash = "—";
  if (!iso) return emDash;
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return emDash;
  return d.toLocaleString(localeTag, {
    day: "2-digit",
    month: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  });
}

function aclCompileDurationMs(
  acl: ProxyACLCompileStatus | null | undefined,
): number | undefined {
  if (acl?.last_compile_duration_ms != null) {
    return acl.last_compile_duration_ms;
  }
  if (acl?.build_started_at && acl?.build_finished_at) {
    const ms =
      new Date(acl.build_finished_at).getTime() -
      new Date(acl.build_started_at).getTime();
    if (ms >= 0) {
      return ms > 0 ? ms : 1;
    }
  }
  return undefined;
}

function formatDurationMs(ms?: number, t?: TranslateFn): string {
  const emDash = "—";
  if (ms == null || ms < 0) return emDash;
  if (!t) {
    if (ms < 1000) return `${ms} ms`;
    const sec = ms / 1000;
    return sec < 60 ? `${sec.toFixed(1)} s` : `${Math.floor(sec / 60)} min`;
  }
  if (ms < 1000) return `${ms} ${t("units.ms")}`;
  const sec = ms / 1000;
  if (sec < 60) {
    return `${sec.toFixed(sec < 10 ? 1 : 0)} ${t("units.sec")}`;
  }
  const min = Math.floor(sec / 60);
  const rest = Math.round(sec % 60);
  return rest > 0
    ? t("units.minSec", { min, sec: rest })
    : t("units.minOnly", { min });
}

function formatDelayFromServeStart(ms?: number, t?: TranslateFn): string {
  if (!t) {
    if (ms == null || ms < 0) return "—";
    return formatDurationMs(ms);
  }
  if (ms == null || ms < 0) return t("dashboard.notYet");
  return t("dashboard.delayAfterServe", {
    duration: formatDurationMs(ms, t),
  });
}

function StatusDot({ tone }: { tone: DashboardStatTone }) {
  const dotClass: Record<DashboardStatTone, string> = {
    success: "bg-teal-400",
    warning: "bg-amber-400 animate-pulse",
    danger: "bg-red-400",
    neutral: "bg-zinc-600",
  };
  return (
    <span className={`h-1.5 w-1.5 rounded-full ${dotClass[tone]}`} />
  );
}

function proxyListenHint(instances: ProxyInstanceRuntimeStatus[]) {
  if (instances.length === 0) {
    return "—";
  }
  return (
    <>
      {instances.map((inst, i) => (
        <span key={inst.id}>
          {i > 0 ? <span className="text-zinc-600">, </span> : null}
          <span className={inst.active ? "text-teal-300" : "text-red-300/90"}>
            {inst.listen}
          </span>
        </span>
      ))}
    </>
  );
}

function instancesCatalogEqual(a: ProxyInstance[], b: ProxyInstance[]): boolean {
  if (a.length !== b.length) {
    return false;
  }
  const sa = [...a].sort((x, y) => x.id.localeCompare(y.id));
  const sb = [...b].sort((x, y) => x.id.localeCompare(y.id));
  for (let i = 0; i < sa.length; i++) {
    if (
      sa[i].id !== sb[i].id ||
      sa[i].name !== sb[i].name ||
      sa[i].listen !== sb[i].listen
    ) {
      return false;
    }
  }
  return true;
}

function MetricsWindow5mBadge({ label }: { label: string }) {
  return (
    <span className="font-mono text-[11px] text-teal-300/70">{label}</span>
  );
}

export default function ManagePage() {
  const { t, localeTag, apiErrorMessage } = useTranslation();
  const formatApiError = useApiErrorMessage();
  const emDash = t("common.emDash");
  const [status, setStatus] = useState<ProxyRuntimeStatus | null>(null);
  const [dbInstances, setDbInstances] = useState<ProxyInstance[]>([]);
  const [createOpen, setCreateOpen] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const statusFpRef = useRef<string | null>(null);

  const loadInstances = useCallback(async () => {
    try {
      const list = await proxyApi.listProxyInstances();
      setDbInstances((prev) => {
        const next = list ?? [];
        if (instancesCatalogEqual(prev, next)) {
          return prev;
        }
        return next;
      });
    } catch {
      /* каталог не блокирует дашборд */
    }
  }, []);

  const pollRuntime = useCallback(async () => {
    try {
      const st = await proxyApi.getProxyRuntimeStatus();
      const fp = runtimeStatusFingerprint(st);
      if (fp !== statusFpRef.current) {
        statusFpRef.current = fp;
        startTransition(() => {
          setStatus(st);
        });
      }
      setError(null);
    } catch (e) {
      setError(formatApiError(e, t("dashboard.loadFailed")));
    }
  }, [formatApiError, t]);

  useEffect(() => {
    void loadInstances();
    void pollRuntime();
    const onCatalogChange = () => void loadInstances();
    window.addEventListener(PROXY_INSTANCES_CHANGED_EVENT, onCatalogChange);
    const timer = setInterval(() => void pollRuntime(), 3000);
    return () => {
      clearInterval(timer);
      window.removeEventListener(PROXY_INSTANCES_CHANGED_EVENT, onCatalogChange);
    };
  }, [loadInstances, pollRuntime]);

  const instances = status?.instances ?? [];
  const safeDbInstances = dbInstances ?? [];
  const fleetAcl = useMemo(() => fleetAclSummary(instances), [instances]);
  const avgRuntimeTraffic = useMemo(
    () => averageInstanceTraffic(instances),
    [instances],
  );
  const traffic = status?.traffic;
  const loading = !status && !error;
  const proxyUp = Boolean(status?.proxy_active);
  const totalDecisions =
    (traffic?.allowed_5m ?? 0) + (traffic?.denied_5m ?? 0);
  const reqPerMin =
    traffic?.requests_1m ??
    (traffic ? Math.round(traffic.requests_per_sec_avg_5m * 60) : 0);
  const reqPerMinAvg5m = traffic
    ? Math.round(traffic.requests_per_sec_avg_5m * 60)
    : 0;
  const reqLast5s = traffic
    ? Math.round(traffic.requests_per_sec_now_5s * 5)
    : 0;
  const buckets = useMemo(() => {
    const fleet = traffic?.buckets_10s ?? [];
    if (fleet.length > 0) {
      return fleet;
    }
    const perInstance = instances
      .map((i) => i.traffic?.buckets_10s ?? [])
      .filter((b) => b.length > 0);
    if (perInstance.length === 0) {
      return [];
    }
    if (perInstance.length === 1) {
      return perInstance[0];
    }
    return mergeTrafficBuckets(perInstance);
  }, [traffic?.buckets_10s, instances]);
  const startup = status?.startup;

  return (
    <div className="flex w-full flex-col gap-3">
      {error ? (
        <Alert type="error" message={error} showIcon />
      ) : null}

      <CreateProxyInstanceModal
        open={createOpen}
        onClose={() => setCreateOpen(false)}
        onCreated={() => {
          void loadInstances();
          void pollRuntime();
        }}
      />

      {instances.some((i) => i.proxy_start_error) ? (
        <Alert
          type="warning"
          showIcon
          message={t("dashboard.proxyDown")}
          description={instances
            .filter((i) => i.proxy_start_error)
            .map((i) => `${i.listen}: ${apiErrorMessage(i.proxy_start_error!)}`)
            .join("; ")}
        />
      ) : null}

      <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
        <DashboardStatCard
          label={t("dashboard.proxy")}
          loading={loading}
          tone={proxyUp ? "success" : "danger"}
          badge={<StatusDot tone={proxyUp ? "success" : "danger"} />}
          value={proxyUp ? "RUNNING" : "STOPPED"}
          hint={proxyListenHint(instances)}
        />
        <DashboardStatCard
          label={t("dashboard.aclBuild")}
          loading={loading && !fleetAcl}
          tone={fleetAcl ? aclBuildTone(fleetAcl) : undefined}
          badge={
            fleetAcl ? <StatusDot tone={aclBuildTone(fleetAcl)} /> : undefined
          }
          value={fleetAcl ? aclBuildLabel(fleetAcl, t) : emDash}
          hint={
            fleetAcl
              ? t("dashboard.rulesPatterns", {
                rules: fleetAcl.active_logical_rules,
                patterns: fleetAcl.active_patterns,
              })
              : undefined
          }
        />
        <DashboardStatCard
          label={t("dashboard.perMinute")}
          loading={loading}
          badge={
            traffic && proxyUp ? (
              <span className="font-mono text-[11px] text-zinc-500">{t("dashboard.sliding")}</span>
            ) : undefined
          }
          value={
            proxyUp ? (
              <span className="font-mono">{formatInt(reqPerMin, localeTag)}</span>
            ) : (
              "—"
            )
          }
          hint={
            traffic && proxyUp
              ? t("dashboard.reqHint", {
                last5s: formatInt(reqLast5s, localeTag),
                avg: formatInt(reqPerMinAvg5m, localeTag),
              })
              : undefined
          }
        />
        <DashboardStatCard
          label="Allow / Deny"
          loading={loading}
          badge={
            traffic ? (
              <MetricsWindow5mBadge label={t("dashboard.window5m")} />
            ) : undefined
          }
          value={
            traffic ? (
              <span className="font-mono text-xl">
                <span className="text-teal-300">
                  {formatInt(traffic.allowed_5m, localeTag)}
                </span>
                <span className="mx-1.5 text-zinc-600">/</span>
                <span className="text-red-300/90">
                  {formatInt(traffic.denied_5m, localeTag)}
                </span>
              </span>
            ) : (
              "—"
            )
          }
          hint={
            traffic
              ? t("dashboard.trafficHint", {
                allowPct: formatSharePct(traffic.allowed_5m, totalDecisions),
                denyPct: formatSharePct(traffic.denied_5m, totalDecisions),
              })
              : undefined
          }
        />
      </div>

      {fleetAcl?.build_error ? (
        <Alert type="error" message={fleetAcl.build_error} showIcon />
      ) : null}

      <div className="rounded-xl border border-white/10 bg-white/5 p-5">
        <p className="text-[13px] font-medium text-zinc-100">{t("dashboard.startupTitle")}</p>
        <p className="mb-4 mt-0.5 font-mono text-[11.5px] text-zinc-500">
          {t("dashboard.startupHint")}
        </p>
        {loading ? (
          <div className="flex justify-center py-6">
            <OktopusLoading size="lg" />
          </div>
        ) : (
          <ul className="grid gap-3 sm:grid-cols-3">
            <li className="flex flex-col gap-1 rounded-lg border border-white/8 bg-black/20 px-4 py-3">
              <span className="text-[11px] text-zinc-500">
                {t("dashboard.serveStart")}
              </span>
              <span className="font-mono text-[13px] text-zinc-100">
                {formatStartupClock(startup?.started_at, localeTag)}
              </span>
            </li>
            <li className="flex flex-col gap-1.5 rounded-lg border border-white/8 bg-black/20 px-4 py-3">
              <span className="text-[11px] text-zinc-500">
                {t("dashboard.proxyReady")}
              </span>
              <span className="text-[13px] leading-snug text-teal-200/90">
                {formatDelayFromServeStart(startup?.proxy_ready_ms, t)}
              </span>
              <span className="font-mono text-[11px] text-zinc-600">
                {t("dashboard.moment", {
                  time: formatStartupClock(startup?.proxy_ready_at, localeTag),
                })}
              </span>
            </li>
            <li className="flex flex-col gap-1.5 rounded-lg border border-white/8 bg-black/20 px-4 py-3">
              <span className="text-[11px] text-zinc-500">
                {t("dashboard.aclCompile")}
              </span>
              <span className="text-[13px] leading-snug text-teal-200/90">
                {fleetAcl?.build_status === "building"
                  ? t("dashboard.inProgress")
                  : formatDurationMs(aclCompileDurationMs(fleetAcl), t)}
              </span>
              <span className="font-mono text-[11px] text-zinc-600">
                {fleetAcl?.build_finished_at
                  ? t("dashboard.aclDoneAt", {
                    time: formatStartupClock(fleetAcl.build_finished_at, localeTag),
                  })
                  : t("dashboard.aclNever")}
              </span>
            </li>
          </ul>
        )}
      </div>

      <div className="grid gap-3 lg:grid-cols-3">
        <DashboardTrafficChartsSection
          buckets={buckets}
          proxyUp={proxyUp}
          loading={loading}
        />

        <DashboardRuntimePanel
          acl={fleetAcl}
          traffic={avgRuntimeTraffic}
          proxyUp={proxyUp}
          loading={loading}
          hintExtra={t("dashboard.runtimeFleetAvg")}
          showBreakdownTooltips={false}
          showAccessLogLink
        />
      </div>
      <div className="rounded-xl border border-white/10 bg-white/[0.05] p-5">
        <div className="mb-4 flex flex-row flex-wrap items-center justify-between gap-3">
          <div>
            <p className="text-[13px] font-medium text-zinc-100">
              {t("instance.fleetTitle")}
            </p>
          </div>
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => setCreateOpen(true)}
          >
            {t("instance.add")}
          </Button>
        </div>
        {safeDbInstances.length === 0 && !loading ? (
          <Typography.Text type="secondary">{t("instance.fleetEmpty")}</Typography.Text>
        ) : (
          <div className="flex flex-col gap-6">
            {safeDbInstances.map((meta) => {
              const runtime =
                instances.find((i) => i.id === meta.id) ?? null;
              if (!runtime) {
                return (
                  <div
                    key={meta.id}
                    className="rounded-lg border border-white/8 bg-black/20 p-4"
                  >
                    <Link
                      href={`/manage/instances/${meta.id}`}
                      className="text-[13px] font-medium text-teal-300 hover:text-teal-200"
                    >
                      {meta.name}
                    </Link>
                    <Typography.Text type="secondary" className="ml-2 font-mono text-[11px]">
                      {meta.listen}
                    </Typography.Text>
                    <Typography.Paragraph type="secondary" className="!mb-0 mt-2 text-[12px]">
                      {t("instance.runtimePending")}
                    </Typography.Paragraph>
                  </div>
                );
              }
              return (
                <ManageFleetInstanceBlock
                  key={meta.id}
                  meta={meta}
                  runtime={runtime}
                  loading={loading}
                />
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
}
