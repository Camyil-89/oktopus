"use client";

import * as proxyApi from "@/api/proxy";
import { InstanceRuntimeDashboardLayout } from "@/assets/components/dashboard/InstanceRuntimeDashboardLayout";
import type { ProxyInstance, ProxyInstanceRuntimeStatus } from "@/types/proxy";
import { useApiErrorMessage, useTranslation } from "@/contexts/LocaleContext";
import { Alert } from "antd";
import { useParams } from "next/navigation";
import { useCallback, useEffect, useMemo, useState } from "react";

function emptyRuntimeStub(
  id: string,
  listen: string,
): ProxyInstanceRuntimeStatus {
  return {
    id,
    listen,
    active: false,
    acl: {
      build_status: "idle",
      config_revision: "",
      active_revision: "",
      rules_in_sync: true,
      active_logical_rules: 0,
      active_patterns: 0,
      enabled_rules_in_db: 0,
    },
    traffic: {
      active_connections: 0,
      active_websocket_connections: 0,
      requests_per_sec_avg_5m: 0,
      requests_per_sec_now_5s: 0,
      requests_1m: 0,
      allowed_5m: 0,
      denied_5m: 0,
      decide_duration_us_avg_5m: 0,
      decide_duration_us_p95_5m: 0,
      decide_duration_us_p99_5m: 0,
      decide_duration_us_max_5m: 0,
      inspect_duration_us_avg_5m: 0,
      inspect_duration_us_p95_5m: 0,
      inspect_duration_us_p99_5m: 0,
      policy_duration_us_avg_5m: 0,
      policy_duration_us_p95_5m: 0,
      policy_duration_us_p99_5m: 0,
      unique_users_5m: 0,
      unique_sources_5m: 0,
      access_log_queue_pending: 0,
      buckets_10s: [],
    },
  };
}

export default function InstanceDashboardPage() {
  const instanceId = useParams().id as string;
  const { t } = useTranslation();
  const formatApiError = useApiErrorMessage();
  const [meta, setMeta] = useState<ProxyInstance | null>(null);
  const [row, setRow] = useState<ProxyInstanceRuntimeStatus | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [initialLoadDone, setInitialLoadDone] = useState(false);

  const load = useCallback(async () => {
    try {
      const [st, instances] = await Promise.all([
        proxyApi.getProxyRuntimeStatus(),
        proxyApi.listProxyInstances().catch(() => [] as ProxyInstance[]),
      ]);
      const runtimeRows = st.instances ?? [];
      const dbRows = instances ?? [];
      const found = runtimeRows.find((i) => i.id === instanceId) ?? null;
      setRow(found);
      setMeta(dbRows.find((i) => i.id === instanceId) ?? null);
      setError(null);
    } catch (e) {
      setError(formatApiError(e, t("dashboard.loadFailed")));
    } finally {
      setInitialLoadDone(true);
    }
  }, [formatApiError, instanceId, t]);

  useEffect(() => {
    void load();
    const timer = setInterval(() => void load(), 3000);
    return () => clearInterval(timer);
  }, [load]);

  const overviewInst = useMemo(
    () => row ?? emptyRuntimeStub(instanceId, meta?.listen ?? ""),
    [row, instanceId, meta?.listen],
  );

  const layoutInst = row ?? overviewInst;
  const layoutLoading = !initialLoadDone && !error;

  if (layoutLoading && !error) {
    return (
      <InstanceRuntimeDashboardLayout
        inst={overviewInst}
        loading
        instanceName={meta?.name}
      />
    );
  }

  return (
    <div className="flex flex-col gap-4">
      {error ? <Alert type="error" message={error} showIcon /> : null}
      {meta ? (
        <>
          {row?.proxy_start_error ? (
            <Alert type="warning" message={row.proxy_start_error} showIcon />
          ) : null}
          <InstanceRuntimeDashboardLayout
            inst={layoutInst}
            loading={false}
            instanceName={meta.name}
            showAccessLogLink
          />
          {!row ? (
            <Alert type="info" message={t("instance.runtimePending")} showIcon />
          ) : null}
        </>
      ) : (
        <Alert type="info" message={t("instance.notFound")} showIcon />
      )}
    </div>
  );
}
