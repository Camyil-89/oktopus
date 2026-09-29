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
import { useCallback, useEffect, useMemo, useState } from "react";

type Row = ProxyInspectRuleDraft & { key: string };

function buildStatusTag(st: ProxyInspectCompileStatus) {
  switch (st.build_status) {
    case "building":
      return <Tag color="processing">Компилируется</Tag>;
    case "error":
      return <Tag color="error">Ошибка</Tag>;
    case "ready":
      return <Tag color="success">Готово</Tag>;
    default:
      return <Tag>Ожидание</Tag>;
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

export function ProxyInspectRulesCard() {
  const { message } = App.useApp();
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
        proxyApi.getProxyInspectStatus(),
        proxyApi.listProxyInspectRules(),
        proxyApi.getProxySettings(),
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
      message.error(e instanceof ApiError ? e.message : "Не удалось загрузить");
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

  const openEdit = async (row: Row) => {
    if (!row.id) return;
    try {
      const full = await proxyApi.getProxyInspectRule(row.id);
      setEditKey(row.key);
      form.setFieldsValue({
        name: full.name,
        script: full.script,
        action: full.action,
        enabled: full.enabled,
      });
      setModalOpen(true);
    } catch {
      message.error("Не удалось загрузить скрипт");
    }
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
      await proxyApi.validateProxyInspectScript({ script });
      message.success("Скрипт корректен");
      return true;
    } catch (e) {
      message.error(
        e instanceof ApiError ? e.message : "Ошибка проверки скрипта",
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
      await proxyApi.validateProxyInspectScript({ script: values.script });
    } catch (e) {
      message.error(
        e instanceof ApiError ? e.message : "Исправьте ошибки в Lua",
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
      message.success("Инструкция скопирована");
    } catch {
      message.error("Не удалось скопировать");
    }
  };

  const downloadAgentPrompt = () => {
    const stamp = new Date().toISOString().replace(/[:.]/g, "-").slice(0, 19);
    downloadTextFile(agentPromptText, `oktopus-inspect-agent-prompt-${stamp}.txt`);
  };

  const persist = async () => {
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
      await proxyApi.syncProxyInspectRules(payload);
      message.success("Правила инспекции сохранены");
      await load();
    } catch (e) {
      message.error(e instanceof ApiError ? e.message : "Ошибка сохранения");
    } finally {
      setSaving(false);
    }
  };

  const columns: ColumnsType<Row> = [
    { title: "№", width: 48, render: (_, __, i) => i + 1 },
    { title: "Имя", dataIndex: "name" },
    {
      title: "При совпадении",
      dataIndex: "action",
      width: 160,
      render: (a: 0 | 1) =>
        a === 0 ? (
          <Tag color="error">Запретить</Tag>
        ) : (
          <Tag color="success">Разрешить</Tag>
        ),
    },
    {
      title: "Вкл",
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
            onClick={() => void openEdit(row)}
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
          message="Инспекция доступна только в режиме MITM"
          description="Сейчас включён tunnel: исходящий HTTPS не расшифровывается, Lua-правила не выполняются. Переключите connect_mode на MITM в настройках прокси и установите CA на клиентах."
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
        title="Инспекция исходящих запросов"
        extra={
          <Space wrap>
            {status ? buildStatusTag(status) : null}
            {dirty ? <Tag color="warning">Черновик</Tag> : null}
            <Button
              icon={<CopyOutlined />}
              onClick={() => void copyAgentPrompt()}
            >
              Инструкция для ИИ
            </Button>
            <Button
              icon={<DownloadOutlined />}
              onClick={downloadAgentPrompt}
            >
              Скачать .txt
            </Button>
            <Button icon={<PlusOutlined />} onClick={openCreate}>
              Правило
            </Button>
            <Button
              type="primary"
              loading={saving}
              disabled={loading || !dirty}
              onClick={() => void persist()}
            >
              Сохранить и применить
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
            locale={{ emptyText: "Нет правил — нажмите «Правило»" }}
          />
        </PanelTable>
      </Card>

      <Collapse
        items={[
          {
            key: "api",
            label: "Справка по ctx и возврату",
            children: (
              <pre className="m-0 whitespace-pre-wrap text-sm">{INSPECT_LUA_API_REFERENCE}</pre>
            ),
          },
          {
            key: "examples",
            label: "Примеры Lua (быстрые)",
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
                      Вставить в новое правило
                    </Button>
                  </div>
                ))}
              </div>
            ),
          },
        ]}
      />

      <Modal
        title={editKey ? "Редактирование" : "Новое правило"}
        open={modalOpen}
        onCancel={() => setModalOpen(false)}
        width={720}
        footer={
          <Space wrap>
            <Button onClick={() => setModalOpen(false)}>Отмена</Button>
            <Button
              loading={scriptValidateLoading}
              onClick={() => void validateModalScript()}
            >
              Проверить код
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
