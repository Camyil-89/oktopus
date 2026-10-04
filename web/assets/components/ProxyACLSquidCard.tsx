"use client";

import { ApiError } from "@/api/base";
import * as proxyApi from "@/api/proxy";
import { PanelTable } from "@/assets/components/PanelTable";
import { ProxyACLPolicyEditor } from "@/assets/components/ProxyACLPolicyEditor";
import {
  ProxyACLNamedListModal,
  type ProxyACLNamedListFormValues,
} from "@/assets/modals/ProxyACLNamedListModal";
import type {
  ProxyACLCompileStatus,
  ProxyACLNamedListDraft,
  ProxyACLNamedListSummary,
  ProxyACLPolicyDiagnostic,
} from "@/types/proxy";
import { ensureRuleId } from "@/utils/uuidv7";
import { DeleteOutlined, PlusOutlined } from "@ant-design/icons";
import {
  App,
  Button,
  Card,
  Form,
  Modal,
  Table,
  Tag,
  Typography,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import { useApiErrorMessage, useTranslation } from "@/contexts/LocaleContext";
import type { TranslateFn } from "@/i18n/translate";
import type { RulesSectionUnsaved } from "@/types/rulesUnsaved";
import { useCallback, useEffect, useMemo, useState } from "react";

const POLICY_PLACEHOLDER = `# acl и http_access (как в Squid)
acl localnet src 10.0.0.0/8
acl all all

http_access deny localnet
http_access allow all
`;

/** Строка таблицы; body только если загружен/редактирован локально. */
type ListRow = ProxyACLNamedListSummary & {
  key: string;
  body?: string;
};

function mapSummaryToRows(lists: ProxyACLNamedListSummary[]): ListRow[] {
  return lists.map((l) => ({
    key: l.id,
    id: l.id,
    name: l.name,
    list_type: l.list_type,
    source_mode: l.source_mode ?? "manual",
    source_url: l.source_url ?? "",
    poll_interval_minutes: l.poll_interval_minutes ?? 60,
    body_line_count: l.body_line_count ?? 0,
    body_preview: l.body_preview ?? "",
    created_at: l.created_at,
    updated_at: l.updated_at,
  }));
}

function summaryFromDetail(
  list: import("@/types/proxy").ProxyACLNamedList,
): ProxyACLNamedListSummary {
  return {
    id: list.id,
    name: list.name,
    list_type: list.list_type,
    source_mode: list.source_mode,
    source_url: list.source_url,
    poll_interval_minutes: list.poll_interval_minutes,
    body_line_count: list.body_line_count,
    body_preview: list.body_preview,
    created_at: list.created_at,
    updated_at: list.updated_at,
  };
}

async function rowsToSyncDrafts(rows: ListRow[]): Promise<ProxyACLNamedListDraft[]> {
  return Promise.all(
    rows.map(async (r) => {
      if (r.source_mode === "remote") {
        return {
          id: r.id,
          name: r.name.trim(),
          list_type: r.list_type,
          body: "",
          source_mode: r.source_mode,
          source_url: r.source_url,
          poll_interval_minutes: r.poll_interval_minutes,
        };
      }
      let body = r.body;
      if (body === undefined) {
        const full = await proxyApi.getProxyACLNamedList(r.id);
        body = full.body;
      }
      return {
        id: r.id,
        name: r.name.trim(),
        list_type: r.list_type,
        body,
        source_mode: r.source_mode,
        source_url: r.source_url,
        poll_interval_minutes: r.poll_interval_minutes,
      };
    }),
  );
}

function compileStatusTag(st: ProxyACLCompileStatus | null, t: TranslateFn) {
  if (!st) return null;
  if (st.build_status === "error") {
    return <Tag color="error">{t("acl.buildError")}</Tag>;
  }
  if (st.build_status === "building") {
    return <Tag color="processing">{t("acl.building")}</Tag>;
  }
  if (!st.rules_in_sync) {
    return <Tag color="warning">{t("acl.publishing")}</Tag>;
  }
  return <Tag color="success">{t("acl.published")}</Tag>;
}

type ProxyACLSquidCardProps = {
  onUnsavedChange?: (state: RulesSectionUnsaved) => void;
};

export function ProxyACLSquidCard({
  onUnsavedChange,
}: ProxyACLSquidCardProps = {}) {
  const { message } = App.useApp();
  const { t } = useTranslation();
  const formatApiError = useApiErrorMessage();
  const [loading, setLoading] = useState(true);
  const [policyText, setPolicyText] = useState("");
  const [savedPolicy, setSavedPolicy] = useState("");
  const [listRows, setListRows] = useState<ListRow[]>([]);
  const [aclStatus, setAclStatus] = useState<ProxyACLCompileStatus | null>(
    null,
  );
  const [savingPolicy, setSavingPolicy] = useState(false);
  const [validatingPolicy, setValidatingPolicy] = useState(false);
  const [savingLists, setSavingLists] = useState(false);
  const [pollingList, setPollingList] = useState(false);
  const [listModalLoading, setListModalLoading] = useState(false);

  const [listModalOpen, setListModalOpen] = useState(false);
  const [listModalMode, setListModalMode] = useState<"create" | "edit">(
    "create",
  );
  const [editingListKey, setEditingListKey] = useState<string | null>(null);
  const [listForm] = Form.useForm<ProxyACLNamedListFormValues>();
  const [patternEditorKey, setPatternEditorKey] = useState("list-new");
  const [policyDiagnostics, setPolicyDiagnostics] = useState<
    ProxyACLPolicyDiagnostic[]
  >([]);

  const listNames = useMemo(
    () =>
      listRows
        .map((r) => r.name.trim())
        .filter((n) => n.length > 0),
    [listRows],
  );

  const load = useCallback(async () => {
    try {
      const [pol, lists, st] = await Promise.all([
        proxyApi.getProxyACLPolicy(),
        proxyApi.listProxyACLNamedLists(),
        proxyApi.getProxyACLStatus(),
      ]);
      const text = pol.config_text?.trim()
        ? pol.config_text
        : POLICY_PLACEHOLDER;
      setPolicyText(text);
      setSavedPolicy(text);
      const rows = mapSummaryToRows(lists);
      setListRows(rows);
      setAclStatus(st);
    } catch (e) {
      message.error(formatApiError(e, t("acl.loadFailed")));
    } finally {
      setLoading(false);
    }
  }, [formatApiError, message, t]);

  useEffect(() => {
    void load();
    const t = setInterval(async () => {
      try {
        const st = await proxyApi.getProxyACLStatus();
        setAclStatus(st);
        if (st.compile_diagnostics?.length) {
          setPolicyDiagnostics(st.compile_diagnostics);
        }
      } catch {
        /* ignore */
      }
    }, 2000);
    return () => clearInterval(t);
  }, [load]);

  useEffect(() => {
    if (loading) return;
    const timer = setTimeout(() => {
      void proxyApi
        .validateProxyACLPolicy(policyText)
        .then((res) => {
          setPolicyDiagnostics(res.diagnostics ?? []);
        })
        .catch(() => {
          /* ignore validate errors */
        });
    }, 400);
    return () => clearTimeout(timer);
  }, [policyText, loading]);

  const policyDirty = useMemo(
    () => policyText !== savedPolicy,
    [policyText, savedPolicy],
  );
  const listMetas = useMemo(
    () =>
      listRows
        .filter((r) => r.name.trim() !== "")
        .map((r) => ({
          name: r.name.trim(),
          list_type: r.list_type,
        })),
    [listRows],
  );

  const persistLists = useCallback(
    async (rows: ListRow[]) => {
      setSavingLists(true);
      try {
        const drafts = await rowsToSyncDrafts(rows);
        const out = await proxyApi.syncProxyACLNamedLists(drafts);
        const mapped = mapSummaryToRows(out);
        setListRows(mapped);
        setAclStatus(await proxyApi.getProxyACLStatus());
        return mapped;
      } catch (e) {
        message.error(formatApiError(e, t("acl.listSaveError")));
        throw e;
      } finally {
        setSavingLists(false);
      }
    },
    [formatApiError, message, t],
  );

  const verifyPolicy = async () => {
    setValidatingPolicy(true);
    try {
      const res = await proxyApi.validateProxyACLPolicy(policyText);
      setPolicyDiagnostics(res.diagnostics ?? []);
      if (res.ok) {
        message.success(t("acl.verifyOk"));
      } else {
        message.warning(t("acl.verifyErrors"));
      }
    } catch (e) {
      message.error(formatApiError(e, t("acl.verifyFailed")));
    } finally {
      setValidatingPolicy(false);
    }
  };

  const savePolicy = useCallback(async (): Promise<boolean> => {
    setSavingPolicy(true);
    try {
      const pol = await proxyApi.putProxyACLPolicy(policyText);
      setSavedPolicy(pol.config_text);
      setPolicyText(pol.config_text);
      message.success(t("acl.configSaved"));
      setAclStatus(await proxyApi.getProxyACLStatus());
      return true;
    } catch (e) {
      if (e instanceof ApiError) {
        message.error(e.message);
        if (Array.isArray(e.diagnostics)) {
          setPolicyDiagnostics(e.diagnostics as ProxyACLPolicyDiagnostic[]);
        }
      } else {
        message.error(t("acl.saveError"));
      }
      return false;
    } finally {
      setSavingPolicy(false);
    }
  }, [formatApiError, message, policyText, t]);

  const discardPolicy = useCallback(() => {
    setPolicyText(savedPolicy);
  }, [savedPolicy]);

  useEffect(() => {
    onUnsavedChange?.({
      dirty: policyDirty,
      discard: discardPolicy,
      save: savePolicy,
    });
  }, [onUnsavedChange, policyDirty, discardPolicy, savePolicy]);

  const openCreateList = () => {
    setListModalMode("create");
    setEditingListKey(null);
    setListModalLoading(false);
    setPatternEditorKey(`list-new-${Date.now()}`);
    listForm.setFieldsValue({
      name: "",
      list_type: "src",
      source_mode: "manual",
      source_url: "",
      poll_interval_minutes: 60,
      body: "",
    });
    setListModalOpen(true);
  };

  const openEditList = (row: ListRow) => {
    setListModalMode("edit");
    setEditingListKey(row.key);
    setPatternEditorKey(`list-${row.key}`);
    setListModalOpen(true);
    setListModalLoading(true);
    listForm.setFieldsValue({
      name: row.name,
      list_type: row.list_type,
      source_mode: row.source_mode,
      source_url: row.source_url,
      poll_interval_minutes: row.poll_interval_minutes,
      body: "",
    });
    void proxyApi
      .getProxyACLNamedList(row.id)
      .then((full) => {
        listForm.setFieldsValue({
          name: full.name,
          list_type: full.list_type,
          source_mode: full.source_mode,
          source_url: full.source_url,
          poll_interval_minutes: full.poll_interval_minutes,
          body: full.body,
        });
        setPatternEditorKey(`list-${row.key}-${Date.now()}`);
      })
      .catch((e) => {
        message.error(formatApiError(e, t("acl.listLoadFailed")));
        setListModalOpen(false);
      })
      .finally(() => setListModalLoading(false));
  };

  const forcePollList = async () => {
    if (!editingListKey) return;
    try {
      await listForm.validateFields([
        "source_mode",
        "source_url",
        "poll_interval_minutes",
      ]);
    } catch {
      return;
    }
    const url = String(listForm.getFieldValue("source_url") ?? "").trim();
    setPollingList(true);
    try {
      const updated = await proxyApi.pollProxyACLNamedList(editingListKey, {
        source_url: url,
      });
      listForm.setFieldsValue({ body: updated.body });
      setPatternEditorKey(`list-${editingListKey}-${Date.now()}`);
      const summary = summaryFromDetail(updated);
      setListRows((rows) =>
        rows.map((r) =>
          r.key === editingListKey ? { ...r, ...summary } : r,
        ),
      );
      setAclStatus(await proxyApi.getProxyACLStatus());
      message.success(t("acl.listUpdated"));
    } catch (e) {
      message.error(formatApiError(e, t("acl.pollFailed")));
    } finally {
      setPollingList(false);
    }
  };

  const submitListModal = async () => {
    const v = await listForm.validateFields();
    const body = v.body.trim();
    const sourceMode = v.source_mode;
    let nextRows: ListRow[];
    if (listModalMode === "edit" && editingListKey) {
      const existing = listRows.find((r) => r.key === editingListKey);
      nextRows = listRows.map((r) =>
        r.key === editingListKey
          ? {
              ...r,
              name: v.name.trim(),
              list_type: v.list_type,
              body,
              source_mode: sourceMode,
              source_url: v.source_url?.trim() ?? "",
              poll_interval_minutes: v.poll_interval_minutes ?? 60,
              id: existing?.id ?? r.id,
            }
          : r,
      );
    } else {
      const id = ensureRuleId(undefined);
      const previewLine =
        body.split("\n").find((l) => l.trim())?.trim() ?? "";
      nextRows = [
        ...listRows,
        {
          key: id,
          id,
          name: v.name.trim(),
          list_type: v.list_type,
          body,
          source_mode: sourceMode,
          source_url: v.source_url?.trim() ?? "",
          poll_interval_minutes: v.poll_interval_minutes ?? 60,
          body_line_count: body ? body.split("\n").length : 0,
          body_preview: previewLine.slice(0, 48),
          created_at: new Date().toISOString(),
          updated_at: new Date().toISOString(),
        },
      ];
    }
    try {
      await persistLists(nextRows);
      message.success(t("acl.listSaved"));
      setListModalOpen(false);
    } catch {
      /* ошибка уже в message */
    }
  };

  const deleteList = (key: string) => {
    const row = listRows.find((r) => r.key === key);
    Modal.confirm({
      title: t("acl.deleteListTitle"),
      content: row?.name ? `«${row.name}»` : undefined,
      okText: t("common.delete"),
      okType: "danger",
      cancelText: t("common.cancel"),
      onOk: async () => {
        const nextRows = listRows.filter((r) => r.key !== key);
        await persistLists(nextRows);
        message.success(t("acl.listDeleted"));
      },
    });
  };

  const listColumns: ColumnsType<ListRow> = [
    {
      title: t("common.name"),
      dataIndex: "name",
      key: "name",
      render: (name: string) => name || "—",
    },
    {
      title: t("common.type"),
      dataIndex: "list_type",
      key: "list_type",
      width: 100,
    },
    {
      title: t("acl.lines"),
      key: "lines",
      width: 90,
      render: (_, row) => row.body_line_count,
    },
    {
      title: t("acl.preview"),
      key: "preview",
      ellipsis: true,
      render: (_, row) => row.body_preview || "—",
    },
    {
      title: "",
      key: "actions",
      width: 56,
      render: (_, row) => (
        <Button
          type="text"
          danger
          size="small"
          icon={<DeleteOutlined />}
          onClick={(e) => {
            e.stopPropagation();
            deleteList(row.key);
          }}
        />
      ),
    },
  ];

  return (
    <div className="flex w-full flex-col gap-4">
      <Card
        loading={loading}
        title={t("acl.cardTitle")}
        extra={compileStatusTag(aclStatus, t)}
      >
        <div className="flex flex-col gap-3">
          <ProxyACLPolicyEditor
            value={policyText}
            onChange={setPolicyText}
            listNames={listNames}
            listMetas={listMetas}
            diagnostics={policyDiagnostics}
            resetKey={savedPolicy.slice(0, 32)}
          />
          {policyDiagnostics.length > 0 ? (
            <ul className="m-0 flex list-none flex-col gap-1 p-0 text-sm">
              {policyDiagnostics.map((d, i) => (
                <li key={`${d.line}-${d.column}-${d.code}-${i}`}>
                  <Typography.Text type="danger">
                    {d.line > 0
                      ? `${d.line}:${d.column} `
                      : d.code === "list"
                        ? "list "
                        : ""}
                    {d.message}
                  </Typography.Text>
                </li>
              ))}
            </ul>
          ) : null}
          <div className="flex flex-row flex-wrap gap-2">
            <Button
              loading={validatingPolicy}
              onClick={() => void verifyPolicy()}
            >
              {t("common.check")}
            </Button>
            <Button
              type="primary"
              loading={savingPolicy}
              disabled={!policyDirty}
              onClick={() => void savePolicy()}
            >
              {t("acl.saveConfig")}
            </Button>
            {aclStatus?.build_error ? (
              <Typography.Text type="danger">
                {aclStatus.build_error}
              </Typography.Text>
            ) : null}
          </div>
        </div>
      </Card>

      <Card
        loading={loading}
        title={t("acl.listsTitle")}
        extra={
          <Button icon={<PlusOutlined />} onClick={openCreateList}>
            {t("common.add")}
          </Button>
        }
      >
        <div className="flex flex-col gap-3">
          <PanelTable>
            <Table<ListRow>
              rowKey="key"
              columns={listColumns}
              dataSource={listRows}
              pagination={false}
              size="small"
              locale={{ emptyText: t("acl.emptyLists") }}
              onRow={(row) => ({
                onClick: () => openEditList(row),
                className: "cursor-pointer",
              })}
            />
          </PanelTable>
        </div>
      </Card>

      <ProxyACLNamedListModal
        open={listModalOpen}
        mode={listModalMode}
        listId={editingListKey}
        form={listForm}
        patternEditorKey={patternEditorKey}
        okLoading={savingLists}
        pollLoading={pollingList}
        modalLoading={listModalLoading}
        onPollNow={forcePollList}
        onCancel={() => setListModalOpen(false)}
        onOk={() => void submitListModal()}
      />
    </div>
  );
}
