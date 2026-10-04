"use client";

import { ApiError } from "@/api/base";
import {
  deleteProxyAccessLogByPeriod,
  listProxyAccessLog,
  listAllProxyInspectRules,
  patchHostSettings,
} from "@/api/proxy";
import { AccessLogDecisionRuleCell } from "@/assets/components/AccessLogDecisionRuleCell";
import { AccessLogRuleNameCell } from "@/assets/components/AccessLogRuleNameCell";
import { AccessLogDeletePeriodModal } from "@/assets/modals/AccessLogDeletePeriodModal";
import { AccessLogDetailModal } from "@/assets/modals/AccessLogDetailModal";
import {
  ACCESS_LOG_EMPTY_FILTERS,
  AccessLogFiltersModal,
  type AccessLogFiltersValues,
} from "@/assets/modals/AccessLogFiltersModal";
import { AccessLogRetentionModal } from "@/assets/modals/AccessLogRetentionModal";
import type { ProxyAccessLogRow } from "@/types/accessLog";
import {
  accessLogAclDecisionLabel,
  isAccessLogRecordId,
  parseAccessLogExtra,
  searchEngineLabel,
} from "@/utils/accessLogExtra";
import { AccessLogActionTag } from "@/utils/accessLogAction";
import type { AccessLogActionCode } from "@/utils/accessLogAction";
import {
  accessLogActionToListParams,
  migrateAccessLogSegmentFromStorage,
  parseAccessLogActionFilter,
} from "@/utils/accessLogFilters";
import {
  ACCESS_LOG_SEGMENT_LABEL_KEYS,
  ACCESS_LOG_SEGMENTS,
  type AccessLogSegment,
  parseAccessLogSegment,
} from "@/utils/accessLogSegment";
import {
  normalizeAccessLogRow,
} from "@/utils/accessLogRow";
import {
  FilterOutlined,
  ReloadOutlined,
  SettingOutlined,
  TableOutlined,
} from "@ant-design/icons";
import {
  App,
  Button,
  Checkbox,
  Dropdown,
  Input,
  Space,
  Table,
  Tabs,
} from "antd";
import type { ColumnsType, TablePaginationConfig } from "antd/es/table";
import Link from "next/link";
import { useApiErrorMessage, useTranslation } from "@/contexts/LocaleContext";
import type { MessageKey } from "@/i18n/translate";
import { useProxyInstanceNameMap } from "@/assets/hooks/useProxyInstanceNameMap";
import { useCallback, useEffect, useMemo, useState } from "react";

const DEFAULT_PAGE_SIZE = 20;
const MAX_PAGE_SIZE = 100;

type ColumnKey =
  | "created_at"
  | "instance"
  | "source_address"
  | "destination_address"
  | "user"
  | "action"
  | "acl_rule"
  | "inspect_rule"
  | "decision_rule"
  | "decide_duration_us"
  | "full_url"
  | "search_engine"
  | "search_query";

const DEFAULT_VISIBLE_COLUMNS: ColumnKey[] = [
  "created_at",
  "instance",
  "source_address",
  "destination_address",
  "user",
  "action",
  "acl_rule",
  "inspect_rule",
  "decision_rule",
];

const COLUMN_LABEL_KEYS: Record<ColumnKey, MessageKey> = {
  created_at: "accessLog.col.created_at",
  instance: "accessLog.col.instance",
  source_address: "accessLog.col.source_address",
  destination_address: "accessLog.col.destination_address",
  user: "accessLog.col.user",
  action: "accessLog.col.action",
  acl_rule: "accessLog.col.acl_rule",
  inspect_rule: "accessLog.col.inspect_rule",
  decision_rule: "accessLog.col.decision_rule",
  decide_duration_us: "accessLog.col.decide_duration_us",
  full_url: "accessLog.col.full_url",
  search_engine: "accessLog.col.search_engine",
  search_query: "accessLog.col.search_query",
};

const ALL_COLUMN_KEYS = Object.keys(COLUMN_LABEL_KEYS) as ColumnKey[];
const VISIBLE_COLUMNS_STORAGE_KEY = "oktopus.manage.access-log.visible-columns";
const FILTERS_STORAGE_KEY = "oktopus.manage.access-log.filters";
const SEGMENT_STORAGE_KEY = "oktopus.manage.access-log.segment";

function loadFiltersFromStorage(): AccessLogFiltersValues {
  if (typeof window === "undefined") {
    return ACCESS_LOG_EMPTY_FILTERS;
  }
  try {
    const raw = localStorage.getItem(FILTERS_STORAGE_KEY);
    if (!raw) {
      return ACCESS_LOG_EMPTY_FILTERS;
    }
    const parsed: unknown = JSON.parse(raw);
    if (!parsed || typeof parsed !== "object") {
      return ACCESS_LOG_EMPTY_FILTERS;
    }
    const p = parsed as Partial<AccessLogFiltersValues> & {
      error_kind?: string;
    };
    const rawAction = (parsed as Record<string, unknown>).action;
    let action = parseAccessLogActionFilter(rawAction);
    if (rawAction === "errors") {
      action = "";
    }
    return {
      instance_id: typeof p.instance_id === "string" ? p.instance_id : "",
      id: typeof p.id === "string" ? p.id : "",
      user: typeof p.user === "string" ? p.user : "",
      source: typeof p.source === "string" ? p.source : "",
      destination: typeof p.destination === "string" ? p.destination : "",
      url: typeof p.url === "string" ? p.url : "",
      search_only: Boolean(p.search_only),
      from: typeof p.from === "string" ? p.from : "",
      to: typeof p.to === "string" ? p.to : "",
      action,
      attack_kind: typeof p.attack_kind === "string" ? p.attack_kind : "",
      policy_anomaly_q:
        typeof p.policy_anomaly_q === "string" ? p.policy_anomaly_q : "",
      decision_rule_ref:
        typeof p.decision_rule_ref === "string" ? p.decision_rule_ref : "",
      inspect_rule_id:
        typeof p.inspect_rule_id === "string" ? p.inspect_rule_id : "",
    };
  } catch {
    return ACCESS_LOG_EMPTY_FILTERS;
  }
}

function loadVisibleColumnsFromStorage(): ColumnKey[] {
  if (typeof window === "undefined") {
    return DEFAULT_VISIBLE_COLUMNS;
  }
  try {
    const raw = localStorage.getItem(VISIBLE_COLUMNS_STORAGE_KEY);
    if (!raw) {
      return DEFAULT_VISIBLE_COLUMNS;
    }
    const parsed: unknown = JSON.parse(raw);
    if (!Array.isArray(parsed)) {
      return DEFAULT_VISIBLE_COLUMNS;
    }
    const valid = parsed.filter(
      (k): k is ColumnKey =>
        typeof k === "string" && ALL_COLUMN_KEYS.includes(k as ColumnKey),
    );
    return valid.length > 0 ? valid : DEFAULT_VISIBLE_COLUMNS;
  } catch {
    return DEFAULT_VISIBLE_COLUMNS;
  }
}

function loadSegmentFromStorage(
  filtersParsed: Record<string, unknown>,
): AccessLogSegment {
  if (typeof window !== "undefined") {
    try {
      const raw = localStorage.getItem(SEGMENT_STORAGE_KEY);
      if (raw) {
        return parseAccessLogSegment(raw);
      }
    } catch {
      /* ignore */
    }
  }
  return migrateAccessLogSegmentFromStorage(filtersParsed) ?? "traffic";
}

function accessLogListParams(
  page: number,
  pageSize: number,
  segment: AccessLogSegment,
  filters: AccessLogFiltersValues,
) {
  const attackParams =
    segment === "attacks"
      ? {
          attack_kind: filters.attack_kind || undefined,
          policy_anomaly_q: filters.policy_anomaly_q || undefined,
        }
      : {};
  return {
    page,
    page_size: pageSize,
    segment,
    id: filters.id || undefined,
    user: filters.user || undefined,
    source: filters.source || undefined,
    destination: filters.destination || undefined,
    url: filters.url || undefined,
    search_only: filters.search_only || undefined,
    from: filters.from || undefined,
    to: filters.to || undefined,
    ...accessLogActionToListParams(parseAccessLogActionFilter(filters.action)),
    ...attackParams,
    decision_rule_ref: filters.decision_rule_ref || undefined,
    inspect_rule_id: filters.inspect_rule_id || undefined,
    instance_id: filters.instance_id || undefined,
  };
}

export default function ManageAccessLogPage() {
  const { message, modal } = App.useApp();
  const { t } = useTranslation();
  const formatApiError = useApiErrorMessage();
  const instanceNames = useProxyInstanceNameMap();
  const columnLabels = useMemo(
    () =>
      Object.fromEntries(
        ALL_COLUMN_KEYS.map((k) => [k, t(COLUMN_LABEL_KEYS[k])]),
      ) as Record<ColumnKey, string>,
    [t],
  );
  const [data, setData] = useState<ProxyAccessLogRow[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(DEFAULT_PAGE_SIZE);
  const [loading, setLoading] = useState(false);

  const [filtersOpen, setFiltersOpen] = useState(false);
  const [appliedFilters, setAppliedFilters] =
    useState<AccessLogFiltersValues>(ACCESS_LOG_EMPTY_FILTERS);
  const [filtersHydrated, setFiltersHydrated] = useState(false);
  const [segment, setSegment] = useState<AccessLogSegment>("traffic");
  const [searchNonce, setSearchNonce] = useState(0);
  const [idSearchDraft, setIdSearchDraft] = useState("");

  const [visibleColumns, setVisibleColumns] = useState<ColumnKey[]>(
    DEFAULT_VISIBLE_COLUMNS,
  );

  useEffect(() => {
    const raw = localStorage.getItem(FILTERS_STORAGE_KEY);
    let filtersParsed: Record<string, unknown> = {};
    if (raw) {
      try {
        const p: unknown = JSON.parse(raw);
        if (p && typeof p === "object") {
          filtersParsed = p as Record<string, unknown>;
        }
      } catch {
        /* ignore */
      }
    }
    const stored = loadFiltersFromStorage();
    setAppliedFilters(stored);
    setIdSearchDraft(stored.id);
    setSegment(loadSegmentFromStorage(filtersParsed));
    setFiltersHydrated(true);
  }, []);

  useEffect(() => {
    setIdSearchDraft(appliedFilters.id);
  }, [appliedFilters.id]);

  useEffect(() => {
    setVisibleColumns(loadVisibleColumnsFromStorage());
  }, []);

  const onVisibleColumnsChange = (keys: ColumnKey[]) => {
    setVisibleColumns(keys);
    try {
      localStorage.setItem(
        VISIBLE_COLUMNS_STORAGE_KEY,
        JSON.stringify(keys),
      );
    } catch {
      /* ignore */
    }
  };

  const [settingsOpen, setSettingsOpen] = useState(false);
  const [settingsSaving, setSettingsSaving] = useState(false);

  const [deleteOpen, setDeleteOpen] = useState(false);
  const [deleteLoading, setDeleteLoading] = useState(false);

  const [detailRow, setDetailRow] = useState<ProxyAccessLogRow | null>(null);
  const [ruleNames, setRuleNames] = useState<Map<string, string>>(
    () => new Map(),
  );

  useEffect(() => {
    void listAllProxyInspectRules()
      .then((inspect) => {
        const m = new Map<string, string>();
        for (const r of inspect) {
          m.set(r.id, r.name);
        }
        setRuleNames(m);
      })
      .catch(() => {
        /* имена правил необязательны для списка */
      });
  }, []);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const res = await listProxyAccessLog(
        accessLogListParams(page, pageSize, segment, appliedFilters),
      );
      setData(res.results.map(normalizeAccessLogRow));
      setTotal(res.count);
    } catch (e) {
      message.error(formatApiError(e, t("accessLog.loadFailed")));
    } finally {
      setLoading(false);
    }
  }, [
    appliedFilters,
    formatApiError,
    message,
    page,
    pageSize,
    searchNonce,
    segment,
    t,
  ]);

  useEffect(() => {
    if (!filtersHydrated) {
      return;
    }
    void load();
  }, [load, filtersHydrated]);

  const applyFilters = (values: AccessLogFiltersValues) => {
    setAppliedFilters(values);
    try {
      localStorage.setItem(FILTERS_STORAGE_KEY, JSON.stringify(values));
    } catch {
      /* ignore */
    }
    setPage(1);
    setSearchNonce((n) => n + 1);
    setFiltersOpen(false);
  };

  const filtersActive =
    appliedFilters.id !== "" ||
    appliedFilters.user !== "" ||
    appliedFilters.source !== "" ||
    appliedFilters.destination !== "" ||
    appliedFilters.url !== "" ||
    appliedFilters.search_only ||
    appliedFilters.from !== "" ||
    appliedFilters.to !== "" ||
    appliedFilters.action !== "" ||
    appliedFilters.attack_kind !== "" ||
    appliedFilters.policy_anomaly_q !== "" ||
    appliedFilters.decision_rule_ref !== "" ||
    appliedFilters.inspect_rule_id !== "";

  const onSegmentChange = (key: string) => {
    const next = parseAccessLogSegment(key);
    setSegment(next);
    setPage(1);
    try {
      localStorage.setItem(SEGMENT_STORAGE_KEY, next);
    } catch {
      /* ignore */
    }
    setSearchNonce((n) => n + 1);
  };

  const refreshList = () => {
    setSearchNonce((n) => n + 1);
  };

  const searchByRecordId = (raw: string) => {
    const id = raw.trim();
    if (id && !isAccessLogRecordId(id)) {
      message.error(t("accessLog.invalidUuid"));
      return;
    }
    applyFilters({ ...appliedFilters, id });
  };

  const confirmDeletePeriod = (range: { from: string; to: string }) => {
    const fromLabel = new Date(range.from).toLocaleString();
    const toLabel = new Date(range.to).toLocaleString();
    modal.confirm({
      title: t("accessLog.deletePeriodConfirm"),
      content: `${fromLabel} — ${toLabel}`,
      okText: t("common.delete"),
      okType: "danger",
      cancelText: t("common.cancel"),
      onOk: async () => {
        setDeleteLoading(true);
        try {
          const res = await deleteProxyAccessLogByPeriod(range);
          message.success(
            t("accessLog.deletedCount", { count: res.deleted }),
          );
          setDeleteOpen(false);
          setPage(1);
          const list = await listProxyAccessLog(
            accessLogListParams(1, pageSize, segment, appliedFilters),
          );
          setData(list.results.map(normalizeAccessLogRow));
          setTotal(list.count);
        } catch (e) {
          message.error(formatApiError(e, t("accessLog.deleteFailed")));
        } finally {
          setDeleteLoading(false);
        }
      },
    });
  };

  const saveRetention = async (body: { access_log_retention_days: number }) => {
    setSettingsSaving(true);
    try {
      await patchHostSettings(body);
      message.success(t("common.saved"));
      setSettingsOpen(false);
    } catch (e) {
      message.error(formatApiError(e, t("common.saveFailed")));
    } finally {
      setSettingsSaving(false);
    }
  };

  const allColumns: ColumnsType<ProxyAccessLogRow> = useMemo(
    () => [
      {
        key: "created_at",
        title: columnLabels.created_at,
        dataIndex: "created_at",
        width: 190,
        render: (v: string) => new Date(v).toLocaleString(),
      },
      {
        key: "instance",
        title: columnLabels.instance,
        dataIndex: "instance_id",
        width: 140,
        ellipsis: true,
        render: (id: string) =>
          instanceNames.get(id) ?? (id ? id.slice(0, 8) : "—"),
      },
      {
        key: "source_address",
        title: columnLabels.source_address,
        dataIndex: "source_address",
        ellipsis: true,
      },
      {
        key: "destination_address",
        title: columnLabels.destination_address,
        dataIndex: "destination_address",
        ellipsis: true,
      },
      {
        key: "user",
        title: columnLabels.user,
        dataIndex: "user",
        width: 140,
        render: (v: string | null) => v ?? "—",
      },
      {
        key: "action",
        title: columnLabels.action,
        dataIndex: "action",
        width: 100,
        render: (_: AccessLogActionCode, row: ProxyAccessLogRow) => (
          <AccessLogActionTag
            action={row.action}
            decisionRuleRef={row.decision_rule_ref}
            deniedBy={row.denied_by}
          />
        ),
      },
      {
        key: "acl_rule",
        title: columnLabels.acl_rule,
        width: 160,
        ellipsis: true,
        render: (_: unknown, row: ProxyAccessLogRow) =>
          accessLogAclDecisionLabel(row.decision_rule_ref, t),
      },
      {
        key: "inspect_rule",
        title: columnLabels.inspect_rule,
        width: 160,
        ellipsis: true,
        render: (_: unknown, row: ProxyAccessLogRow) => (
          <AccessLogRuleNameCell
            ruleId={row.inspect_rule_id}
            ruleNames={ruleNames}
          />
        ),
      },
      {
        key: "decision_rule",
        title: columnLabels.decision_rule,
        width: 180,
        ellipsis: true,
        render: (_: unknown, row: ProxyAccessLogRow) => (
          <AccessLogDecisionRuleCell
            decisionRuleRef={row.decision_rule_ref}
            ruleNames={ruleNames}
          />
        ),
      },
      {
        key: "decide_duration_us",
        title: columnLabels.decide_duration_us,
        dataIndex: "decide_duration_us",
        width: 100,
      },
      {
        key: "full_url",
        title: columnLabels.full_url,
        dataIndex: "full_url",
        ellipsis: true,
      },
      {
        key: "search_engine",
        title: columnLabels.search_engine,
        width: 140,
        render: (_: unknown, row: ProxyAccessLogRow) =>
          searchEngineLabel(parseAccessLogExtra(row.extra).search?.engine, t),
      },
      {
        key: "search_query",
        title: columnLabels.search_query,
        ellipsis: true,
        render: (_: unknown, row: ProxyAccessLogRow) => {
          const q = parseAccessLogExtra(row.extra).search?.query;
          return q ?? "—";
        },
      },
    ],
    [columnLabels, instanceNames, ruleNames, t],
  );

  const tableColumns = useMemo(
    () =>
      allColumns.filter((col) =>
        visibleColumns.includes(col.key as ColumnKey),
      ),
    [allColumns, visibleColumns],
  );

  const onTableChange = (pagination: TablePaginationConfig) => {
    setPage(pagination.current ?? 1);
    const size = pagination.pageSize ?? DEFAULT_PAGE_SIZE;
    setPageSize(Math.min(size, MAX_PAGE_SIZE));
  };

  const columnPicker = (
    <div className="flex flex-col gap-2 rounded-lg border border-white/8 bg-[#141416] p-3 shadow-lg">
      <Checkbox.Group
        className="flex flex-col gap-1"
        value={visibleColumns}
        onChange={(keys) => onVisibleColumnsChange(keys as ColumnKey[])}
        options={ALL_COLUMN_KEYS.map((key) => ({
          label: columnLabels[key],
          value: key,
        }))}
      />
    </div>
  );

  return (
    <div className="flex w-full flex-col gap-4">
      <header className="flex flex-row flex-wrap items-center justify-between gap-3">
        <Link
          href="/manage/reports"
          className="text-[12px] text-zinc-500 transition hover:text-teal-300"
        >
          {t("accessLog.reportsLink")}
        </Link>
        <Space wrap>
          <Button danger onClick={() => setDeleteOpen(true)}>
            {t("accessLog.deletePeriod")}
          </Button>
          <Button
            icon={<SettingOutlined />}
            onClick={() => setSettingsOpen(true)}
          >
            {t("common.settings")}
          </Button>
        </Space>
      </header>

      <AccessLogDeletePeriodModal
        open={deleteOpen}
        loading={deleteLoading}
        onCancel={() => setDeleteOpen(false)}
        onSubmit={(range) => confirmDeletePeriod(range)}
      />

      <AccessLogRetentionModal
        open={settingsOpen}
        saving={settingsSaving}
        onCancel={() => setSettingsOpen(false)}
        onSubmit={(values) => void saveRetention(values)}
      />

      <AccessLogDetailModal
        row={detailRow}
        open={detailRow !== null}
        onClose={() => setDetailRow(null)}
        ruleNames={ruleNames}
        instanceNames={instanceNames}
      />

      <AccessLogFiltersModal
        open={filtersOpen}
        initialValues={appliedFilters}
        onCancel={() => setFiltersOpen(false)}
        onApply={applyFilters}
      />

      <div className="panel-table overflow-hidden rounded-xl border border-white/10 bg-white/[0.05]">
        <div className="px-5 pt-3">
          <Tabs
            activeKey={segment}
            onChange={onSegmentChange}
            tabBarStyle={{ marginBottom: 0 }}
            items={ACCESS_LOG_SEGMENTS.map((key) => ({
              key,
              label: t(ACCESS_LOG_SEGMENT_LABEL_KEYS[key]),
            }))}
          />
        </div>
        <div className="flex flex-row flex-wrap items-center gap-2 border-b border-white/8 px-5 py-4">
          <Space wrap className="flex-1">
            <Input.Search
              allowClear
              placeholder={t("accessLog.recordIdPlaceholder")}
              value={idSearchDraft}
              onChange={(e) => setIdSearchDraft(e.target.value)}
              onSearch={searchByRecordId}
              className="w-full min-w-[220px] max-w-sm"
              enterButton={t("common.search")}
            />
            <Button
              icon={<ReloadOutlined />}
              loading={loading}
              onClick={refreshList}
            >
              {t("common.refresh")}
            </Button>
            <Button
              type={filtersActive ? "primary" : "default"}
              icon={<FilterOutlined />}
              onClick={() => setFiltersOpen(true)}
            >
              {t("common.filters")}
            </Button>
            <Dropdown dropdownRender={() => columnPicker} trigger={["click"]}>
              <Button icon={<TableOutlined />}>{t("common.columns")}</Button>
            </Dropdown>
          </Space>
        </div>
        <Table<ProxyAccessLogRow>
            rowKey="id"
            loading={loading}
            columns={tableColumns}
            dataSource={data}
            pagination={{
              current: page,
              pageSize,
              total,
              showSizeChanger: true,
              pageSizeOptions: ["20", "50", "100"],
              showTotal: (c) => t("common.total", { count: c }),
            }}
            onChange={onTableChange}
            scroll={{ x: true }}
          rowClassName={() => "row-hover"}
          onRow={(record) => ({
            onClick: () => setDetailRow(record),
            style: { cursor: "pointer" },
          })}
        />
      </div>
    </div>
  );
}
