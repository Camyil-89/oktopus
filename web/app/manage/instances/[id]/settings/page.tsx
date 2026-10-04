"use client";

import { ApiError } from "@/api/base";
import * as proxyApi from "@/api/proxy";
import { useApiErrorMessage, useTranslation } from "@/contexts/LocaleContext";
import type { TranslateFn } from "@/i18n/translate";
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
import { PROXY_INSTANCES_CHANGED_EVENT } from "@/assets/modals/CreateProxyInstanceModal";
import { useParams, useRouter } from "next/navigation";
import { useCallback, useEffect, useState } from "react";

function keyBitsOptions(t: TranslateFn) {
  return [
    { value: 2048, label: t("proxy.keyBits2048") },
    { value: 3072, label: t("proxy.keyBits3072") },
    { value: 4096, label: t("proxy.keyBits4096") },
  ];
}

function connectModeOptions(t: TranslateFn) {
  return [
    { value: "tunnel", label: t("proxy.tunnel") },
    { value: "mitm", label: t("proxy.mitm") },
  ] as const;
}

function authBackendOptions(t: TranslateFn) {
  return [
    { value: "ldap", label: "LDAP" },
    { value: "static", label: t("proxy.staticUsers") },
  ];
}

type FormValues = {
  enabled: boolean;
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

function formatValidUntil(iso?: string, localeTag = "ru-RU") {
  if (!iso) return null;
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return null;
  return new Intl.DateTimeFormat(localeTag, { dateStyle: "long" }).format(d);
}

function caStatusText(
  status: ProxyCAStatus | null,
  t: TranslateFn,
  localeTag: string,
) {
  if (!status?.cert_installed) {
    return t("proxy.caNotInstalled");
  }
  const until = formatValidUntil(status.valid_until, localeTag);
  if (until) {
    return t("proxy.caInstalledUntil", { until });
  }
  return t("proxy.caInstalled");
}

export default function InstanceSettingsPage() {
  const params = useParams();
  const router = useRouter();
  const instanceId = params.id as string;
  const { message, modal } = App.useApp();
  const formatApiError = useApiErrorMessage();
  const { t, localeTag } = useTranslation();
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
  const [instanceName, setInstanceName] = useState("");
  const [nameDraft, setNameDraft] = useState("");
  const [renaming, setRenaming] = useState(false);
  const [deleting, setDeleting] = useState(false);

  const loadCAStatus = useCallback(async () => {
    try {
      const st = await proxyApi.getProxyCAStatus(instanceId);
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
        message.error(formatApiError(e, t("proxy.previewFailed")));
      }
    },
    [formatApiError, message, t],
  );

  const applyToForm = useCallback(
    (s: import("@/types/proxy").ProxyInstance) => {
      form.setFieldsValue({
        enabled: s.enabled,
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
      const s = await proxyApi.getProxyInstance(instanceId);
      setInstanceName(s.name);
      setNameDraft(s.name);
      await loadCAStatus();
      await loadForbiddenStatus();
      await loadGatewayStatus();
      applyToForm(s);
    } catch (e) {
      message.error(formatApiError(e, t("proxy.loadFailed")));
    } finally {
      setLoading(false);
    }
  }, [applyToForm, formatApiError, instanceId, loadCAStatus, loadForbiddenStatus, loadGatewayStatus, message, t]);

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
      const updated = await proxyApi.patchProxyInstance(instanceId, body);
      applyToForm(updated);
      message.success(t("proxy.savedApplied"));
    } catch (e) {
      message.error(formatApiError(e, t("proxy.saveFailed")));
    } finally {
      setSaving(false);
    }
  };

  const saveInstanceName = async () => {
    const trimmed = nameDraft.trim();
    if (!trimmed) {
      message.error(t("instance.nameRequired"));
      return;
    }
    if (trimmed === instanceName) {
      return;
    }
    setRenaming(true);
    try {
      const updated = await proxyApi.patchProxyInstance(instanceId, {
        name: trimmed,
      });
      setInstanceName(updated.name);
      setNameDraft(updated.name);
      window.dispatchEvent(new Event(PROXY_INSTANCES_CHANGED_EVENT));
      message.success(t("instance.nameSaved"));
    } catch (e) {
      if (e instanceof ApiError) {
        if (e.message === "instance_name_required") {
          message.error(t("instance.nameRequired"));
          return;
        }
        if (e.message === "instance_name_taken") {
          message.error(t("instance.nameTaken"));
          return;
        }
      }
      message.error(formatApiError(e, t("proxy.saveFailed")));
    } finally {
      setRenaming(false);
    }
  };

  const confirmDeleteInstance = () => {
    modal.confirm({
      title: t("instance.deleteConfirmTitle", {
        name: instanceName || instanceId,
      }),
      content: t("instance.deleteConfirmBody"),
      okText: t("common.delete"),
      cancelText: t("common.cancel"),
      okType: "danger",
      onOk: async () => {
        setDeleting(true);
        try {
          await proxyApi.deleteProxyInstance(instanceId);
          window.dispatchEvent(new Event(PROXY_INSTANCES_CHANGED_EVENT));
          message.success(t("instance.deleted"));
          router.push("/manage");
        } catch (e) {
          message.error(formatApiError(e, t("instance.deleteFailed")));
        } finally {
          setDeleting(false);
        }
      },
    });
  };

  const confirmGenerate = () => {
    modal.confirm({
      title: t("proxy.generateCaTitle"),
      content: t("proxy.generateCaContent"),
      okText: t("common.generate"),
      cancelText: t("common.cancel"),
      okButtonProps: { danger: true },
      onOk: async () => {
        setGenerating(true);
        try {
          const updated = await proxyApi.generateProxyCA(instanceId, {
            key_bits: genKeyBits,
          });
          applyToForm(updated);
          await loadCAStatus();
          message.success(t("proxy.certCreated"));
        } catch (e) {
          message.error(formatApiError(e, t("proxy.certCreateFailed")));
        } finally {
          setGenerating(false);
        }
      },
    });
  };

  const handleForbiddenUpload = async () => {
    const html = forbiddenHtmlFiles[0]?.originFileObj;
    if (!html) {
      message.error(t("proxy.selectHtml"));
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
      message.success(t("proxy.page403Uploaded"));
    } catch (e) {
      message.error(formatApiError(e, t("proxy.pageUploadFailed")));
    } finally {
      setForbiddenUploading(false);
    }
  };

  const handleGatewayUpload = async () => {
    const html = gatewayHtmlFiles[0]?.originFileObj;
    if (!html) {
      message.error(t("proxy.selectHtml"));
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
      message.success(t("proxy.page502Uploaded"));
    } catch (e) {
      message.error(formatApiError(e, t("proxy.pageUploadFailed")));
    } finally {
      setGatewayUploading(false);
    }
  };

  const confirmClearGateway = () => {
    modal.confirm({
      title: t("proxy.reset502Title"),
      okText: t("common.return"),
      cancelText: t("common.cancel"),
      onOk: async () => {
        try {
          const st = await proxyApi.clearProxyGatewayPage();
          setGatewayStatus(st);
          message.success(t("proxy.usingDefaultPage"));
        } catch (e) {
          message.error(formatApiError(e, t("proxy.pageResetFailed")));
        }
      },
    });
  };

  const confirmClearForbidden = () => {
    modal.confirm({
      title: t("proxy.reset403Title"),
      okText: t("common.return"),
      cancelText: t("common.cancel"),
      onOk: async () => {
        try {
          const st = await proxyApi.clearProxyForbiddenPage();
          setForbiddenStatus(st);
          message.success(t("proxy.usingDefaultPage"));
        } catch (e) {
          message.error(formatApiError(e, t("proxy.pageResetFailed")));
        }
      },
    });
  };

  const handleUpload = async () => {
    const cert = certFiles[0]?.originFileObj;
    const key = keyFiles[0]?.originFileObj;
    if (!cert || !key) {
      message.error(t("proxy.selectBothFiles"));
      return;
    }
    setUploading(true);
    try {
      const fd = new FormData();
      fd.append("cert", cert);
      fd.append("key", key);
      const updated = await proxyApi.uploadProxyCA(instanceId, fd);
      applyToForm(updated);
      setCertFiles([]);
      setKeyFiles([]);
      setUploadOpen(false);
      await loadCAStatus();
      message.success(t("proxy.certUploaded"));
    } catch (e) {
      message.error(formatApiError(e, t("proxy.certUploadFailed")));
    } finally {
      setUploading(false);
    }
  };

  return (
    <div className="flex w-full flex-col gap-4">
      <Card title={t("proxy.params")} loading={loading}>
        <Form
          form={form}
          layout="vertical"
          initialValues={{ auth_cache_ttl_minutes: 5 }}
          onFinish={(v) => void handleSave(v)}
          className="flex max-w-3xl flex-col gap-0"
        >
          <Typography.Title level={5}>{t("proxy.network")}</Typography.Title>
          <Form.Item
            name="enabled"
            label={t("proxy.server")}
            valuePropName="checked"
          >
            <Switch checkedChildren={t("common.on")} unCheckedChildren={t("common.off")} />
          </Form.Item>
          <Form.Item
            name="listen"
            label={t("proxy.listenAddress")}
            rules={[{ required: true, message: t("proxy.listenRequired") }]}
          >
            <Input placeholder="127.0.0.1:8080" />
          </Form.Item>
          <Form.Item name="connect_mode" label={t("proxy.connectMode")}>
            <Select
              placeholder={t("proxy.tunnel")}
              options={[...connectModeOptions(t)]}
            />
          </Form.Item>

          <Divider />

          <Typography.Title level={5}>{t("proxy.auth")}</Typography.Title>
          <Form.Item
            name="auth_enabled"
            label={t("proxy.requireAuth")}
            valuePropName="checked"
          >
            <Switch checkedChildren={t("common.yes")} unCheckedChildren={t("common.no")} />
          </Form.Item>

          {authEnabled ? (
            <>
              <Form.Item name="auth_backend" label={t("proxy.authBackend")}>
                <Select
                  placeholder="LDAP"
                  options={[...authBackendOptions(t)]}
                />
              </Form.Item>
              <Form.Item name="auth_realm" label={t("proxy.authRealm")}>
                <Input placeholder="oktopus" />
              </Form.Item>
              <Form.Item
                name="auth_cache_ttl_minutes"
                label={t("proxy.authCacheMin")}
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
                  label={t("proxy.accounts")}
                  rules={[
                    {
                      required: true,
                      message: t("proxy.accountsRequired"),
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
                  <Typography.Title level={5}>{t("proxy.ldapConnection")}</Typography.Title>
                  <Form.Item name="ldap_url" label={t("proxy.ldapServer")}>
                    <Input placeholder="ldap://127.0.0.1:1389" />
                  </Form.Item>

                  <Typography.Title level={5}>
                    {t("proxy.ldapServiceAccount")}
                  </Typography.Title>
                  <Form.Item name="ldap_bind_dn" label={t("proxy.ldapBindDn")}>
                    <Input placeholder="cn=admin,dc=oktopus,dc=dev" />
                  </Form.Item>
                  <Form.Item name="ldap_bind_password" label={t("common.password")}>
                    <Input.Password placeholder={t("proxy.ldapPasswordPlaceholder")} />
                  </Form.Item>
                  <Form.Item name="ldap_base_dn" label={t("proxy.ldapBaseDn")}>
                    <Input placeholder="dc=oktopus,dc=dev" />
                  </Form.Item>
                </>
              ) : null}
            </>
          ) : null}

          <Divider />
          <Button type="primary" htmlType="submit" loading={saving}>
            {t("common.save")}
          </Button>
        </Form>
      </Card>

      <Card title={t("proxy.page403")}>
        <div className="flex max-w-3xl flex-col gap-4">
          <Typography.Text>
            {forbiddenStatus?.using_custom
              ? t("proxy.customHtml")
              : t("proxy.defaultHtml")}
          </Typography.Text>
          <Space wrap>
            <Button onClick={() => setForbiddenUploadOpen(true)}>
              {t("proxy.uploadHtml")}
            </Button>
            <Button
              disabled={!forbiddenStatus?.using_custom}
              onClick={confirmClearForbidden}
            >
              {t("common.default")}
            </Button>
            <Button
              onClick={() => void openErrorPagePreview("forbidden", "default")}
            >
              {t("proxy.previewTemplate")}
            </Button>
            <Button
              disabled={!forbiddenStatus?.using_custom}
              onClick={() => void openErrorPagePreview("forbidden", "custom")}
            >
              {t("proxy.previewUploaded")}
            </Button>
          </Space>
        </div>
      </Card>

      <Card title={t("proxy.page502")}>
        <div className="flex max-w-3xl flex-col gap-4">
          <Typography.Text>
            {gatewayStatus?.using_custom
              ? t("proxy.customHtml")
              : t("proxy.defaultHtml")}
          </Typography.Text>
          <Space wrap>
            <Button onClick={() => setGatewayUploadOpen(true)}>
              {t("proxy.uploadHtml")}
            </Button>
            <Button
              disabled={!gatewayStatus?.using_custom}
              onClick={confirmClearGateway}
            >
              {t("common.default")}
            </Button>
            <Button
              onClick={() => void openErrorPagePreview("gateway", "default")}
            >
              {t("proxy.previewTemplate")}
            </Button>
            <Button
              disabled={!gatewayStatus?.using_custom}
              onClick={() => void openErrorPagePreview("gateway", "custom")}
            >
              {t("proxy.previewUploaded")}
            </Button>
          </Space>
        </div>
      </Card>

      <Card title={t("instance.cardTitle")} loading={loading}>
        <div className="flex max-w-3xl flex-col gap-4">
          <div className="flex flex-col gap-2">
            <Typography.Text className="text-[13px]">
              {t("instance.nameLabel")}
            </Typography.Text>
            <div className="flex flex-col gap-2 sm:flex-row sm:items-start">
              <Input
                value={nameDraft}
                onChange={(e) => setNameDraft(e.target.value)}
                onPressEnter={() => void saveInstanceName()}
                placeholder={t("instance.nameLabel")}
                disabled={loading}
                className="sm:max-w-md"
              />
              <Button
                type="primary"
                loading={renaming}
                disabled={
                  loading ||
                  !nameDraft.trim() ||
                  nameDraft.trim() === instanceName
                }
                onClick={() => void saveInstanceName()}
              >
                {t("instance.saveName")}
              </Button>
            </div>
          </div>
          <Divider className="!my-0" />
          <div className="flex flex-col gap-3">
            <Typography.Text className="text-[13px] font-medium">
              {t("instance.deleteTitle")}
            </Typography.Text>
            <Typography.Text type="secondary">
              {t("instance.deleteConfirmBody")}
            </Typography.Text>
            <Button danger loading={deleting} onClick={confirmDeleteInstance}>
              {t("instance.deleteButton")}
            </Button>
          </div>
        </div>
      </Card>

      <Card title={t("proxy.mitmCa")}>
        <div className="flex max-w-3xl flex-col gap-4">
          <Typography.Text>{caStatusText(caStatus, t, localeTag)}</Typography.Text>

          <Space wrap>
            <Button
              disabled={!caStatus?.cert_installed}
              onClick={() =>
                void proxyApi
                  .downloadProxyCACert(instanceId)
                  .catch(handleDownloadError(message, formatApiError, t("proxy.downloadFailed")))
              }
            >
              {t("proxy.downloadCert")}
            </Button>
            <Button
              disabled={!caStatus?.key_installed}
              onClick={() =>
                void proxyApi
                  .downloadProxyCAKey(instanceId)
                  .catch(handleDownloadError(message, formatApiError, t("proxy.downloadFailed")))
              }
            >
              {t("proxy.downloadKey")}
            </Button>
            <Select
              className="min-w-[140px]"
              value={genKeyBits}
              onChange={setGenKeyBits}
              options={keyBitsOptions(t)}
            />
            <Button danger loading={generating} onClick={confirmGenerate}>
              {t("proxy.generateNew")}
            </Button>
            <Button onClick={() => setUploadOpen(true)}>{t("proxy.uploadCert")}</Button>
          </Space>
        </div>
      </Card>

      <Modal
        title={t("proxy.upload502")}
        open={gatewayUploadOpen}
        onCancel={() => setGatewayUploadOpen(false)}
        onOk={() => void handleGatewayUpload()}
        confirmLoading={gatewayUploading}
        okText={t("common.upload")}
        cancelText={t("common.cancel")}
      >
        <Upload
          beforeUpload={() => false}
          maxCount={1}
          fileList={gatewayHtmlFiles}
          onChange={({ fileList }) => setGatewayHtmlFiles(fileList)}
          accept=".html,.htm,text/html"
        >
          <Button icon={<UploadOutlined />}>{t("common.htmlFile")}</Button>
        </Upload>
      </Modal>

      <Modal
        title={t("proxy.upload403")}
        open={forbiddenUploadOpen}
        onCancel={() => setForbiddenUploadOpen(false)}
        onOk={() => void handleForbiddenUpload()}
        confirmLoading={forbiddenUploading}
        okText={t("common.upload")}
        cancelText={t("common.cancel")}
      >
        <Upload
          beforeUpload={() => false}
          maxCount={1}
          fileList={forbiddenHtmlFiles}
          onChange={({ fileList }) => setForbiddenHtmlFiles(fileList)}
          accept=".html,.htm,text/html"
        >
          <Button icon={<UploadOutlined />}>{t("common.htmlFile")}</Button>
        </Upload>
      </Modal>

      <Modal
        title={t("proxy.uploadCertKey")}
        open={uploadOpen}
        onCancel={() => setUploadOpen(false)}
        onOk={() => void handleUpload()}
        confirmLoading={uploading}
        okText={t("common.upload")}
        cancelText={t("common.cancel")}
      >
        <div className="flex flex-col gap-3">
          <Upload
            beforeUpload={() => false}
            maxCount={1}
            fileList={certFiles}
            onChange={({ fileList }) => setCertFiles(fileList)}
            accept=".crt,.pem"
          >
            <Button icon={<UploadOutlined />}>{t("proxy.certFile")}</Button>
          </Upload>
          <Upload
            beforeUpload={() => false}
            maxCount={1}
            fileList={keyFiles}
            onChange={({ fileList }) => setKeyFiles(fileList)}
            accept=".key,.pem"
          >
            <Button icon={<UploadOutlined />}>{t("proxy.keyFile")}</Button>
          </Upload>
        </div>
      </Modal>
    </div>
  );
}

function handleDownloadError(
  message: { error: (s: string) => void },
  formatApiError: (e: unknown, fallback: string) => string,
  fallback: string,
) {
  return (e: unknown) => {
    message.error(formatApiError(e, fallback));
  };
}
