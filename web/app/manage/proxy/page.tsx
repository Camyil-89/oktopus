"use client";

import { ApiError } from "@/api/base";
import * as proxyApi from "@/api/proxy";
import type {
  ProxyCAStatus,
  ProxyErrorPagePreviewVariant,
  ProxyForbiddenPageStatus,
  ProxyGatewayPageStatus,
  ProxySettings,
} from "@/types/proxy";
import { UploadOutlined } from "@ant-design/icons";
import {
  App,
  Button,
  Card,
  Divider,
  Form,
  Input,
  InputNumber,
  Modal,
  Select,
  Space,
  Switch,
  Typography,
  Upload,
} from "antd";
import type { UploadFile } from "antd/es/upload";
import { useCallback, useEffect, useState } from "react";

const KEY_BITS_OPTIONS = [
  { value: 2048, label: "2048 бит" },
  { value: 3072, label: "3072 бит" },
  { value: 4096, label: "4096 бит" },
];

const CONNECT_MODE_OPTIONS = [
  { value: "tunnel", label: "Туннель" },
  { value: "mitm", label: "Перехват HTTPS (MITM)" },
] as const;

const AUTH_BACKEND_OPTIONS = [
  { value: "ldap", label: "LDAP" },
  { value: "static", label: "Список login:password" },
];

type FormValues = {
  proxy_enabled: boolean;
  listen: string;
  connect_mode: string;
  auth_enabled: boolean;
  auth_static_users: string;
  auth_realm: string;
  auth_backend: string;
  auth_cache_ttl_minutes: number;
  ldap_url: string;
  ldap_base_dn: string;
  ldap_bind_dn: string;
  ldap_bind_password?: string;
};

function formatValidUntil(iso?: string) {
  if (!iso) return null;
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return null;
  return new Intl.DateTimeFormat("ru-RU", { dateStyle: "long" }).format(d);
}

function caStatusText(status: ProxyCAStatus | null) {
  if (!status?.cert_installed) {
    return "Корневой сертификат не установлен.";
  }
  const until = formatValidUntil(status.valid_until);
  if (until) {
    return `Сертификат установлен, действует до ${until}.`;
  }
  return "Сертификат установлен.";
}

export default function ManageProxyPage() {
  const { message, modal } = App.useApp();
  const [form] = Form.useForm<FormValues>();
  const authEnabled = Form.useWatch("auth_enabled", form);
  const authBackend = Form.useWatch("auth_backend", form);

  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [generating, setGenerating] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [caStatus, setCaStatus] = useState<ProxyCAStatus | null>(null);
  const [genKeyBits, setGenKeyBits] = useState(4096);
  const [uploadOpen, setUploadOpen] = useState(false);
  const [certFiles, setCertFiles] = useState<UploadFile[]>([]);
  const [keyFiles, setKeyFiles] = useState<UploadFile[]>([]);
  const [forbiddenStatus, setForbiddenStatus] =
    useState<ProxyForbiddenPageStatus | null>(null);
  const [forbiddenUploadOpen, setForbiddenUploadOpen] = useState(false);
  const [forbiddenUploading, setForbiddenUploading] = useState(false);
  const [forbiddenHtmlFiles, setForbiddenHtmlFiles] = useState<UploadFile[]>([]);
  const [gatewayStatus, setGatewayStatus] =
    useState<ProxyGatewayPageStatus | null>(null);
  const [gatewayUploadOpen, setGatewayUploadOpen] = useState(false);
  const [gatewayUploading, setGatewayUploading] = useState(false);
  const [gatewayHtmlFiles, setGatewayHtmlFiles] = useState<UploadFile[]>([]);

  const loadCAStatus = useCallback(async () => {
    try {
      const st = await proxyApi.getProxyCAStatus();
      setCaStatus(st);
    } catch {
      setCaStatus(null);
    }
  }, []);

  const loadForbiddenStatus = useCallback(async () => {
    try {
      const st = await proxyApi.getProxyForbiddenPageStatus();
      setForbiddenStatus(st);
    } catch {
      setForbiddenStatus(null);
    }
  }, []);

  const loadGatewayStatus = useCallback(async () => {
    try {
      const st = await proxyApi.getProxyGatewayPageStatus();
      setGatewayStatus(st);
    } catch {
      setGatewayStatus(null);
    }
  }, []);

  const openErrorPagePreview = useCallback(
    async (kind: "forbidden" | "gateway", variant: ProxyErrorPagePreviewVariant) => {
      try {
        await proxyApi.openProxyErrorPagePreview(kind, variant);
      } catch (e) {
        const msg =
          e instanceof ApiError ? e.message : "Не удалось открыть предпросмотр";
        message.error(msg);
      }
    },
    [message],
  );

  const applyToForm = useCallback(
    (s: ProxySettings) => {
      form.setFieldsValue({
        proxy_enabled: s.proxy_enabled,
        listen: s.listen,
        connect_mode: s.connect_mode,
        auth_enabled: s.auth_enabled,
        auth_static_users: s.auth_static_users,
        auth_realm: s.auth_realm,
        auth_backend: s.auth_backend,
        auth_cache_ttl_minutes: s.auth_cache_ttl_minutes,
        ldap_url: s.ldap_url,
        ldap_base_dn: s.ldap_base_dn,
        ldap_bind_dn: s.ldap_bind_dn,
      });
    },
    [form],
  );

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const s = await proxyApi.getProxySettings();
      await loadCAStatus();
      await loadForbiddenStatus();
      await loadGatewayStatus();
      applyToForm(s);
    } catch (e) {
      const msg =
        e instanceof ApiError ? e.message : "Не удалось загрузить настройки";
      message.error(msg);
    } finally {
      setLoading(false);
    }
  }, [applyToForm, loadCAStatus, loadForbiddenStatus, loadGatewayStatus, message]);

  useEffect(() => {
    void load();
  }, [load]);

  const handleSave = async (values: FormValues) => {
    setSaving(true);
    try {
      const body: Record<string, unknown> = { ...values };
      if (!values.ldap_bind_password) {
        delete body.ldap_bind_password;
      }
      const updated = await proxyApi.patchProxySettings(body);
      applyToForm(updated);
      message.success("Настройки сохранены и применены к прокси");
    } catch (e) {
      const msg =
        e instanceof ApiError ? e.message : "Не удалось сохранить настройки";
      message.error(msg);
    } finally {
      setSaving(false);
    }
  };

  const confirmGenerate = () => {
    modal.confirm({
      title: "Сгенерировать новый корневой сертификат?",
      content:
        "Текущий сертификат и ключ будут удалены. Все клиенты должны будут установить новый корневой CA.",
      okText: "Сгенерировать",
      cancelText: "Отмена",
      okButtonProps: { danger: true },
      onOk: async () => {
        setGenerating(true);
        try {
          const updated = await proxyApi.generateProxyCA({
            key_bits: genKeyBits,
          });
          applyToForm(updated);
          await loadCAStatus();
          message.success("Новый сертификат создан");
        } catch (e) {
          const msg =
            e instanceof ApiError
              ? e.message
              : "Не удалось создать сертификат";
          message.error(msg);
        } finally {
          setGenerating(false);
        }
      },
    });
  };

  const handleForbiddenUpload = async () => {
    const html = forbiddenHtmlFiles[0]?.originFileObj;
    if (!html) {
      message.error("Выберите HTML-файл");
      return;
    }
    setForbiddenUploading(true);
    try {
      const fd = new FormData();
      fd.append("html", html);
      const st = await proxyApi.uploadProxyForbiddenPage(fd);
      setForbiddenStatus(st);
      setForbiddenHtmlFiles([]);
      setForbiddenUploadOpen(false);
      message.success("Страница 403 загружена");
    } catch (e) {
      const msg =
        e instanceof ApiError ? e.message : "Не удалось загрузить страницу";
      message.error(msg);
    } finally {
      setForbiddenUploading(false);
    }
  };

  const handleGatewayUpload = async () => {
    const html = gatewayHtmlFiles[0]?.originFileObj;
    if (!html) {
      message.error("Выберите HTML-файл");
      return;
    }
    setGatewayUploading(true);
    try {
      const fd = new FormData();
      fd.append("html", html);
      const st = await proxyApi.uploadProxyGatewayPage(fd);
      setGatewayStatus(st);
      setGatewayHtmlFiles([]);
      setGatewayUploadOpen(false);
      message.success("Страница 502 загружена");
    } catch (e) {
      const msg =
        e instanceof ApiError ? e.message : "Не удалось загрузить страницу";
      message.error(msg);
    } finally {
      setGatewayUploading(false);
    }
  };

  const confirmClearGateway = () => {
    modal.confirm({
      title: "Вернуть страницу 502 по умолчанию?",
      okText: "Вернуть",
      cancelText: "Отмена",
      onOk: async () => {
        try {
          const st = await proxyApi.clearProxyGatewayPage();
          setGatewayStatus(st);
          message.success("Используется страница по умолчанию");
        } catch (e) {
          const msg =
            e instanceof ApiError ? e.message : "Не удалось сбросить страницу";
          message.error(msg);
        }
      },
    });
  };

  const confirmClearForbidden = () => {
    modal.confirm({
      title: "Вернуть страницу 403 по умолчанию?",
      okText: "Вернуть",
      cancelText: "Отмена",
      onOk: async () => {
        try {
          const st = await proxyApi.clearProxyForbiddenPage();
          setForbiddenStatus(st);
          message.success("Используется страница по умолчанию");
        } catch (e) {
          const msg =
            e instanceof ApiError ? e.message : "Не удалось сбросить страницу";
          message.error(msg);
        }
      },
    });
  };

  const handleUpload = async () => {
    const cert = certFiles[0]?.originFileObj;
    const key = keyFiles[0]?.originFileObj;
    if (!cert || !key) {
      message.error("Выберите оба файла");
      return;
    }
    setUploading(true);
    try {
      const fd = new FormData();
      fd.append("cert", cert);
      fd.append("key", key);
      const updated = await proxyApi.uploadProxyCA(fd);
      applyToForm(updated);
      setCertFiles([]);
      setKeyFiles([]);
      setUploadOpen(false);
      await loadCAStatus();
      message.success("Сертификат загружен");
    } catch (e) {
      const msg =
        e instanceof ApiError ? e.message : "Не удалось загрузить сертификат";
      message.error(msg);
    } finally {
      setUploading(false);
    }
  };

  return (
    <div className="flex w-full flex-col gap-4">
      <Card title="Параметры" loading={loading}>
        <Form
          form={form}
          layout="vertical"
          initialValues={{ auth_cache_ttl_minutes: 5 }}
          onFinish={(v) => void handleSave(v)}
          className="flex max-w-3xl flex-col gap-0"
        >
          <Typography.Title level={5}>Сеть</Typography.Title>
          <Form.Item
            name="proxy_enabled"
            label="Прокси-сервер"
            valuePropName="checked"
          >
            <Switch checkedChildren="Вкл" unCheckedChildren="Выкл" />
          </Form.Item>
          <Form.Item
            name="listen"
            label="Адрес прослушивания"
            rules={[{ required: true, message: "Укажите адрес" }]}
          >
            <Input placeholder="127.0.0.1:8080" />
          </Form.Item>
          <Form.Item name="connect_mode" label="Режим HTTPS (CONNECT)">
            <Select
              placeholder="Туннель"
              options={[...CONNECT_MODE_OPTIONS]}
            />
          </Form.Item>

          <Divider />

          <Typography.Title level={5}>Авторизация</Typography.Title>
          <Form.Item
            name="auth_enabled"
            label="Требовать логин и пароль"
            valuePropName="checked"
          >
            <Switch checkedChildren="Да" unCheckedChildren="Нет" />
          </Form.Item>

          {authEnabled ? (
            <>
              <Form.Item name="auth_backend" label="Источник учётных записей">
                <Select
                  placeholder="LDAP"
                  options={[...AUTH_BACKEND_OPTIONS]}
                />
              </Form.Item>
              <Form.Item name="auth_realm" label="Имя realm">
                <Input placeholder="oktopus" />
              </Form.Item>
              <Form.Item
                name="auth_cache_ttl_minutes"
                label="Кеш успешного входа, мин"
              >
                <InputNumber
                  className="w-full"
                  min={0}
                  placeholder="5"
                />
              </Form.Item>

              {authBackend === "static" ? (
                <Form.Item
                  name="auth_static_users"
                  label="Учётные записи"
                  rules={[
                    {
                      required: true,
                      message: "Укажите хотя бы одну строку login:password",
                    },
                  ]}
                >
                  <Input.TextArea
                    rows={5}
                    placeholder={"user:password\nuser2:password2"}
                  />
                </Form.Item>
              ) : null}

              {authBackend === "ldap" ? (
                <>
                  <Divider />
                  <Typography.Title level={5}>LDAP — подключение</Typography.Title>
                  <Form.Item name="ldap_url" label="Адрес сервера">
                    <Input placeholder="ldap://127.0.0.1:1389" />
                  </Form.Item>

                  <Typography.Title level={5}>
                    LDAP — сервисная учётка
                  </Typography.Title>
                  <Form.Item name="ldap_bind_dn" label="DN сервисной учётки">
                    <Input placeholder="cn=admin,dc=oktopus,dc=dev" />
                  </Form.Item>
                  <Form.Item name="ldap_bind_password" label="Пароль">
                    <Input.Password placeholder="Пусто — не менять" />
                  </Form.Item>
                  <Form.Item name="ldap_base_dn" label="Корень каталога (Base DN)">
                    <Input placeholder="dc=oktopus,dc=dev" />
                  </Form.Item>
                </>
              ) : null}
            </>
          ) : null}

          <Divider />
          <Button type="primary" htmlType="submit" loading={saving}>
            Сохранить
          </Button>
        </Form>
      </Card>

      <Card title="Страница 403 (ACL)">
        <div className="flex max-w-3xl flex-col gap-4">
          <Typography.Text>
            {forbiddenStatus?.using_custom
              ? "Используется загруженная HTML-страница."
              : "Используется страница по умолчанию из config."}
          </Typography.Text>
          <Space wrap>
            <Button onClick={() => setForbiddenUploadOpen(true)}>
              Загрузить HTML…
            </Button>
            <Button
              disabled={!forbiddenStatus?.using_custom}
              onClick={confirmClearForbidden}
            >
              По умолчанию
            </Button>
            <Button
              onClick={() => void openErrorPagePreview("forbidden", "default")}
            >
              Предпросмотр: шаблон
            </Button>
            <Button
              disabled={!forbiddenStatus?.using_custom}
              onClick={() => void openErrorPagePreview("forbidden", "custom")}
            >
              Предпросмотр: загруженная
            </Button>
          </Space>
        </div>
      </Card>

      <Card title="Страница 502 (ошибка шлюза)">
        <div className="flex max-w-3xl flex-col gap-4">
          <Typography.Text>
            {gatewayStatus?.using_custom
              ? "Используется загруженная HTML-страница."
              : "Используется страница по умолчанию из config."}
          </Typography.Text>
          <Space wrap>
            <Button onClick={() => setGatewayUploadOpen(true)}>
              Загрузить HTML…
            </Button>
            <Button
              disabled={!gatewayStatus?.using_custom}
              onClick={confirmClearGateway}
            >
              По умолчанию
            </Button>
            <Button
              onClick={() => void openErrorPagePreview("gateway", "default")}
            >
              Предпросмотр: шаблон
            </Button>
            <Button
              disabled={!gatewayStatus?.using_custom}
              onClick={() => void openErrorPagePreview("gateway", "custom")}
            >
              Предпросмотр: загруженная
            </Button>
          </Space>
        </div>
      </Card>

      <Card title="Корневой сертификат для MITM">
        <div className="flex max-w-3xl flex-col gap-4">
          <Typography.Text>{caStatusText(caStatus)}</Typography.Text>

          <Space wrap>
            <Button
              disabled={!caStatus?.cert_installed}
              onClick={() => void proxyApi.downloadProxyCACert().catch(handleDownloadError(message))}
            >
              Скачать сертификат
            </Button>
            <Button
              disabled={!caStatus?.key_installed}
              onClick={() => void proxyApi.downloadProxyCAKey().catch(handleDownloadError(message))}
            >
              Скачать ключ
            </Button>
            <Select
              className="min-w-[140px]"
              value={genKeyBits}
              onChange={setGenKeyBits}
              options={KEY_BITS_OPTIONS}
            />
            <Button danger loading={generating} onClick={confirmGenerate}>
              Сгенерировать новый
            </Button>
            <Button onClick={() => setUploadOpen(true)}>Загрузить…</Button>
          </Space>
        </div>
      </Card>

      <Modal
        title="Загрузить страницу 502"
        open={gatewayUploadOpen}
        onCancel={() => setGatewayUploadOpen(false)}
        onOk={() => void handleGatewayUpload()}
        confirmLoading={gatewayUploading}
        okText="Загрузить"
        cancelText="Отмена"
      >
        <Upload
          beforeUpload={() => false}
          maxCount={1}
          fileList={gatewayHtmlFiles}
          onChange={({ fileList }) => setGatewayHtmlFiles(fileList)}
          accept=".html,.htm,text/html"
        >
          <Button icon={<UploadOutlined />}>HTML-файл</Button>
        </Upload>
      </Modal>

      <Modal
        title="Загрузить страницу 403"
        open={forbiddenUploadOpen}
        onCancel={() => setForbiddenUploadOpen(false)}
        onOk={() => void handleForbiddenUpload()}
        confirmLoading={forbiddenUploading}
        okText="Загрузить"
        cancelText="Отмена"
      >
        <Upload
          beforeUpload={() => false}
          maxCount={1}
          fileList={forbiddenHtmlFiles}
          onChange={({ fileList }) => setForbiddenHtmlFiles(fileList)}
          accept=".html,.htm,text/html"
        >
          <Button icon={<UploadOutlined />}>HTML-файл</Button>
        </Upload>
      </Modal>

      <Modal
        title="Загрузить сертификат и ключ"
        open={uploadOpen}
        onCancel={() => setUploadOpen(false)}
        onOk={() => void handleUpload()}
        confirmLoading={uploading}
        okText="Загрузить"
        cancelText="Отмена"
      >
        <div className="flex flex-col gap-3">
          <Upload
            beforeUpload={() => false}
            maxCount={1}
            fileList={certFiles}
            onChange={({ fileList }) => setCertFiles(fileList)}
            accept=".crt,.pem"
          >
            <Button icon={<UploadOutlined />}>Файл сертификата (.crt)</Button>
          </Upload>
          <Upload
            beforeUpload={() => false}
            maxCount={1}
            fileList={keyFiles}
            onChange={({ fileList }) => setKeyFiles(fileList)}
            accept=".key,.pem"
          >
            <Button icon={<UploadOutlined />}>Файл ключа (.key)</Button>
          </Upload>
        </div>
      </Modal>
    </div>
  );
}

function handleDownloadError(message: { error: (s: string) => void }) {
  return (e: unknown) => {
    const msg = e instanceof ApiError ? e.message : "Не удалось скачать файл";
    message.error(msg);
  };
}
