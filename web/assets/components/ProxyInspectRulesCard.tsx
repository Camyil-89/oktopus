"use client";

import { ApiError } from "@/api/base";
import * as proxyApi from "@/api/proxy";
import { buildInspectAgentPrompt } from "@/assets/inspect/inspectAgentPrompt";
import {
  INSPECT_LUA_API_REFERENCE,
  INSPECT_LUA_EXAMPLES,
} from "@/assets/inspect/luaExamples";
import { downloadTextFile } from "@/utils/downloadText";
import { ProxyInspectRuleFormFields } from "@/assets/modals/ProxyInspectRuleFormFields";
import { scrollableModalProps } from "@/assets/modals/modalConfig";
import { PanelTable } from "@/assets/components/PanelTable";
import type {
  ProxyInspectCompileStatus,
  ProxyInspectRuleDraft,
} from "@/types/inspect";
import type { RulesSectionUnsaved } from "@/types/rulesUnsaved";
import { ensureRuleId } from "@/utils/uuidv7";
import {
  ArrowDownOutlined,
  ArrowUpOutlined,
  CopyOutlined,
  DeleteOutlined,
  DownloadOutlined,
  EditOutlined,
  PlusOutlined,
} from "@ant-design/icons";
import {
  App,
  Alert,
  Button,
  Card,
  Collapse,
  Form,
  Modal,
  Space,
  Switch,
  Table,
  Tag,
  Typography,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import { useApiErrorMessage, useTranslation } from "@/contexts/LocaleContext";
import type { TranslateFn } from "@/i18n/translate";
import { useCallback, useEffect, useMemo, useState } from "react";

type Row = ProxyInspectRuleDraft & { key: string };

function buildStatusTag(st: ProxyInspectCompileStatus, t: TranslateFn) {
  switch (st.build_status) {
    case "building":
      return <Tag color="processing">{t("inspect.compiling")}</Tag>;
    case "error":
      return <Tag color="error">{t("inspect.error")}</Tag>;
    case "ready":
      return <Tag color="success">{t("inspect.ready")}</Tag>;
    default:
      return <Tag>{t("inspect.pending")}</Tag>;
  }
}

function rowsSnapshot(rows: Row[]) {
  const sorted = [...rows].sort((a, b) => a.sort_order - b.sort_order);
  return JSON.stringify(
    sorted.map(({ key: _k, ...r }) => ({
      id: r.id,
      name: r.name,
      script: r.script,
      action: r.action,
      enabled: r.enabled,
      sort_order: r.sort_order,
    })),
  );
}

function rowsFromSnapshot(snapshot: string): Row[] {
  type SnapRow = Omit<Row, "key"> & { id: string };
  const items = JSON.parse(snapshot) as SnapRow[];
  return items.map((r) => ({
    ...r,
    id: ensureRuleId(r.id),
    key: ensureRuleId(r.id),
  }));
}

type ProxyInspectRulesCardProps = {
  instanceId: string;
  onUnsavedChange?: (state: RulesSectionUnsaved) => void;
};

export function ProxyInspectRulesCard({
  instanceId,
  onUnsavedChange,
}: ProxyInspectRulesCardProps) {
  const { message } = App.useApp();
  const { t } = useTranslation();
  const formatApiError = useApiErrorMessage();
  const [status, setStatus] = useState<ProxyInspectCompileStatus | null>(null);
  const [rows, setRows] = useState<Row[]>([]);
  const [savedSnapshot, setSavedSnapshot] = useState(() => rowsSnapshot([]));
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);
  const [editKey, setEditKey] = useState<string | null>(null);
  const [scriptValidateLoading, setScriptValidateLoading] = useState(false);
  const [form] = Form.useForm<ProxyInspectRuleDraft>();
  const [connectMode, setConnectMode] = useState<"mitm" | "tunnel" | null>(
    null,
  );
  const mitmRequired = connectMode !== null && connectMode !== "mitm";

  const dirty = useMemo(
    () => rowsSnapshot(rows) !== savedSnapshot,
    [rows, savedSnapshot],
  );

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const [st, list, settings] = await Promise.all([
        proxyApi.getProxyInspectStatus(instanceId),
        proxyApi.listProxyInspectRules(instanceId),
        proxyApi.getProxyInstance(instanceId),
      ]);
      setConnectMode(settings.connect_mode);
      setStatus(st);
      const next: Row[] = list.map((r) => ({
        key: r.id,
        id: r.id,
        name: r.name,
        script: r.script,
        action: r.action,
        enabled: r.enabled,
        sort_order: r.sort_order,
      }));
      setRows(next);
      setSavedSnapshot(rowsSnapshot(next));
    } catch (e) {
      message.error(formatApiError(e, t("inspect.loadFailed")));
    } finally {
      setLoading(false);
    }
  }, [message]);

  useEffect(() => {
    void load();
  }, [load]);

  useEffect(() => {
    if (!status || status.build_status !== "building") return;
    const t = window.setInterval(() => void load(), 2000);
    return () => window.clearInterval(t);
  }, [status, load]);

  const openCreate = () => {
    setEditKey(null);
    form.setFieldsValue({
      name: "",
      script: INSPECT_LUA_EXAMPLES[0]?.script ?? "",
      action: 0,
      enabled: true,
    });
    setModalOpen(true);
  };

  const openEdit = (row: Row) => {
    setEditKey(row.key);
    form.setFieldsValue({
      name: row.name,
      script: row.script,
      action: row.action,
      enabled: row.enabled,
    });
    setModalOpen(true);
  };

  const validateModalScript = async (): Promise<boolean> => {
    let script: string;
    try {
      ({ script } = await form.validateFields(["script"]));
    } catch {
      return false;
    }
    setScriptValidateLoading(true);
    try {
      await proxyApi.validateProxyInspectScript(instanceId, { script });
      message.success(t("inspect.scriptOk"));
      return true;
    } catch (e) {
      message.error(
        formatApiError(e, t("inspect.scriptValidateFailed")),
      );
      return false;
    } finally {
      setScriptValidateLoading(false);
    }
  };

  const saveModal = async () => {
    let values: ProxyInspectRuleDraft;
    try {
      values = await form.validateFields();
    } catch {
      return;
    }
    setScriptValidateLoading(true);
    try {
      await proxyApi.validateProxyInspectScript(instanceId, { script: values.script });
    } catch (e) {
      message.error(
        formatApiError(e, t("inspect.fixLua")),
      );
      return;
    } finally {
      setScriptValidateLoading(false);
    }
    if (editKey) {
      setRows((prev) =>
        prev.map((r) =>
          r.key === editKey
            ? {
              ...r,
              id: ensureRuleId(r.id),
              name: values.name,
              script: values.script,
              action: values.action,
              enabled: values.enabled,
            }
            : r,
        ),
      );
    } else {
      const newId = ensureRuleId(undefined);
      setRows((prev) => [
        ...prev,
        {
          key: newId,
          id: newId,
          name: values.name,
          script: values.script,
          action: values.action,
          enabled: values.enabled,
          sort_order: prev.length,
        },
      ]);
    }
    setModalOpen(false);
  };

  const move = (index: number, dir: -1 | 1) => {
    setRows((prev) => {
      const j = index + dir;
      if (j < 0 || j >= prev.length) return prev;
      const copy = [...prev];
      [copy[index], copy[j]] = [copy[j], copy[index]];
      return copy.map((r, i) => ({ ...r, sort_order: i }));
    });
  };

  const remove = (key: string) => {
    setRows((prev) =>
      prev.filter((r) => r.key !== key).map((r, i) => ({ ...r, sort_order: i })),
    );
  };

  const toggleEnabled = (key: string, enabled: boolean) => {
    setRows((prev) =>
      prev.map((r) => (r.key === key ? { ...r, enabled } : r)),
    );
  };

  const agentPromptText = useMemo(() => buildInspectAgentPrompt(), []);

  const copyAgentPrompt = async () => {
    try {
      await navigator.clipboard.writeText(agentPromptText);
      message.success(t("inspect.promptCopied"));
    } catch {
      message.error(t("common.copyFailed"));
    }
  };

  const downloadAgentPrompt = () => {
    const stamp = new Date().toISOString().replace(/[:.]/g, "-").slice(0, 19);
    downloadTextFile(agentPromptText, `oktopus-inspect-agent-prompt-${stamp}.txt`);
  };

  const persist = useCallback(async (): Promise<boolean> => {
    setSaving(true);
    try {
      const payload: ProxyInspectRuleDraft[] = rows.map((r, i) => ({
        id: ensureRuleId(r.id),
        name: r.name,
        script: r.script,
        action: r.action,
        enabled: r.enabled,
        sort_order: i,
      }));
      await proxyApi.syncProxyInspectRules(instanceId, payload);
      message.success(t("inspect.saved"));
      await load();
      return true;
    } catch (e) {
      message.error(formatApiError(e, t("inspect.saveFailed")));
      return false;
    } finally {
      setSaving(false);
    }
  }, [formatApiError, load, message, rows, t]);

  const discardChanges = useCallback(() => {
    setRows(rowsFromSnapshot(savedSnapshot));
  }, [savedSnapshot]);

  useEffect(() => {
    onUnsavedChange?.({
      dirty,
      discard: discardChanges,
      save: persist,
    });
  }, [dirty, discardChanges, onUnsavedChange, persist]);

  const columns: ColumnsType<Row> = [
    { title: "№", width: 48, render: (_, __, i) => i + 1 },
    { title: t("common.name"), dataIndex: "name" },
    {
      title: t("inspect.onMatch"),
      dataIndex: "action",
      width: 160,
      render: (a: 0 | 1) =>
        a === 0 ? (
          <Tag color="error">{t("inspect.deny")}</Tag>
        ) : (
          <Tag color="success">{t("inspect.allow")}</Tag>
        ),
    },
    {
      title: t("inspect.enabledCol"),
      width: 56,
      align: "center",
      render: (_, r) => (
        <Switch
          size="small"
          checked={r.enabled}
          onChange={(checked) => toggleEnabled(r.key, checked)}
        />
      ),
    },
    {
      title: "",
      width: 200,
      render: (_, row, index) => (
        <Space size="small">
          <Button
            type="text"
            size="small"
            icon={<ArrowUpOutlined />}
            disabled={index === 0}
            onClick={() => move(index, -1)}
          />
          <Button
            type="text"
            size="small"
            icon={<ArrowDownOutlined />}
            disabled={index === rows.length - 1}
            onClick={() => move(index, 1)}
          />
          <Button
            type="text"
            size="small"
            icon={<EditOutlined />}
            onClick={() => openEdit(row)}
          />
          <Button
            type="text"
            size="small"
            danger
            icon={<DeleteOutlined />}
            onClick={() => remove(row.key)}
          />
        </Space>
      ),
    },
  ];

  return (
    <div className="relative flex flex-col gap-4">
      {mitmRequired ? (
        <Alert
          type="warning"
          showIcon
          message={t("inspect.mitmOnlyTitle")}
          description={t("inspect.mitmOnlyDesc")}
        />
      ) : null}
      <div
        className={
          mitmRequired
            ? "pointer-events-none flex flex-col gap-4 opacity-45 select-none"
            : "flex flex-col gap-4"
        }
      >
      <Card
        title={t("inspect.cardTitle")}
        extra={
          <Space wrap>
            {status ? buildStatusTag(status, t) : null}
            {dirty ? <Tag color="warning">{t("inspect.draft")}</Tag> : null}
            <Button
              icon={<CopyOutlined />}
              onClick={() => void copyAgentPrompt()}
            >
              {t("inspect.aiPrompt")}
            </Button>
            <Button
              icon={<DownloadOutlined />}
              onClick={downloadAgentPrompt}
            >
              {t("inspect.downloadTxt")}
            </Button>
            <Button icon={<PlusOutlined />} onClick={openCreate}>
              {t("inspect.rule")}
            </Button>
            <Button
              type="primary"
              loading={saving}
              disabled={loading || !dirty}
              onClick={() => void persist()}
            >
              {t("inspect.saveApply")}
            </Button>
          </Space>
        }
      >
        {status?.build_error ? (
          <Typography.Text type="danger" className="mb-3 block">
            {status.build_error}
          </Typography.Text>
        ) : null}
        <PanelTable>
          <Table<Row>
            size="small"
            rowKey="key"
            loading={loading}
            pagination={false}
            tableLayout="fixed"
            scroll={{ x: 720 }}
            columns={columns}
            dataSource={rows}
            rowClassName={(r) =>
              r.enabled
                ? "row-hover"
                : "row-hover [&>td]:opacity-50 [&>td:nth-child(4)]:opacity-100 [&>td:nth-child(5)]:opacity-100"
            }
            locale={{ emptyText: t("inspect.emptyRules") }}
          />
        </PanelTable>
      </Card>

      <Collapse
        items={[
          {
            key: "api",
            label: t("inspect.ctxHelp"),
            children: (
              <pre className="m-0 whitespace-pre-wrap text-sm">{INSPECT_LUA_API_REFERENCE}</pre>
            ),
          },
          {
            key: "examples",
            label: t("inspect.luaExamples"),
            children: (
              <div className="flex flex-col gap-4">
                {INSPECT_LUA_EXAMPLES.map((ex) => (
                  <div key={ex.id} className="flex flex-col gap-1">
                    <Typography.Text strong>{ex.title}</Typography.Text>
                    <Typography.Text type="secondary">{ex.description}</Typography.Text>
                    <pre className="m-0 overflow-x-auto rounded bg-black/5 p-3 text-xs dark:bg-white/5">
                      {ex.script}
                    </pre>
                    <Button
                      size="small"
                      onClick={() => {
                        setEditKey(null);
                        form.setFieldsValue({
                          name: ex.title,
                          script: ex.script,
                          action: 0,
                          enabled: true,
                        });
                        setModalOpen(true);
                      }}
                    >
                      {t("inspect.insertNewRule")}
                    </Button>
                  </div>
                ))}
              </div>
            ),
          },
        ]}
      />

      <Modal
        title={editKey ? t("inspect.editRule") : t("inspect.newRule")}
        open={modalOpen}
        onCancel={() => setModalOpen(false)}
        width={720}
        footer={
          <Space wrap>
            <Button onClick={() => setModalOpen(false)}>{t("common.cancel")}</Button>
            <Button
              loading={scriptValidateLoading}
              onClick={() => void validateModalScript()}
            >
              {t("inspect.validateCode")}
            </Button>
            <Button
              type="primary"
              loading={scriptValidateLoading}
              onClick={() => void saveModal()}
            >
              OK
            </Button>
          </Space>
        }
        {...scrollableModalProps}
      >
        <ProxyInspectRuleFormFields form={form} />
      </Modal>
      </div>
    </div>
  );
}
