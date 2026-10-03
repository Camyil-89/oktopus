"use client";

import {
  ProxyACLPatternEditor,
  countPatternLines,
} from "@/assets/components/ProxyACLPatternEditor";
import { scrollableModalProps } from "@/assets/modals/modalConfig";
import { useTranslation } from "@/contexts/LocaleContext";
import type { ProxyACLListSourceMode } from "@/types/proxy";
import { Button, Form, Input, InputNumber, Modal, Select, Spin } from "antd";
import type { FormInstance } from "antd";

export type ProxyACLNamedListFormValues = {
  name: string;
  list_type: "src" | "dstdomain" | "port";
  source_mode: ProxyACLListSourceMode;
  source_url: string;
  poll_interval_minutes: number;
  body: string;
};

type ProxyACLNamedListModalProps = {
  open: boolean;
  mode: "create" | "edit";
  listId?: string | null;
  form: FormInstance<ProxyACLNamedListFormValues>;
  onCancel: () => void;
  onOk: () => void;
  onPollNow?: () => void | Promise<void>;
  okLoading?: boolean;
  pollLoading?: boolean;
  modalLoading?: boolean;
  patternEditorKey?: string;
};

export function ProxyACLNamedListModal({
  open,
  mode,
  listId,
  form,
  onCancel,
  onOk,
  onPollNow,
  okLoading,
  pollLoading,
  modalLoading,
  patternEditorKey,
}: ProxyACLNamedListModalProps) {
  const { t } = useTranslation();
  const sourceMode = Form.useWatch("source_mode", form) ?? "manual";
  const isManual = sourceMode === "manual";
  const showPoll =
    mode === "edit" && !isManual && listId != null && listId !== "" && onPollNow;

  const title =
    mode === "create" ? t("acl.listNew") : t("acl.listEdit");

  return (
    <Modal
      title={title}
      open={open}
      onCancel={onCancel}
      width={720}
      destroyOnHidden
      {...scrollableModalProps}
      footer={
        <div className="flex flex-row justify-end gap-2">
          <Button onClick={onCancel}>{t("common.cancel")}</Button>
          {showPoll ? (
            <Button loading={pollLoading} onClick={() => void onPollNow()}>
              {t("acl.pollNow")}
            </Button>
          ) : null}
          <Button type="primary" loading={okLoading} onClick={onOk}>
            {t("acl.done")}
          </Button>
        </div>
      }
    >
      <Spin spinning={modalLoading ?? false}>
      <Form form={form} layout="vertical" className="flex flex-col gap-0">
        <Form.Item
          name="name"
          label={t("common.name")}
          rules={[{ required: true, message: t("acl.listNameRequired") }]}
        >
          <Input placeholder={t("acl.listNamePlaceholder")} />
        </Form.Item>
        <Form.Item
          name="list_type"
          label={t("common.type")}
          rules={[{ required: true }]}
        >
          <Select
            options={[
              { value: "src", label: "src" },
              { value: "dstdomain", label: "dstdomain" },
              { value: "port", label: "port" },
            ]}
          />
        </Form.Item>
        <Form.Item
          name="source_mode"
          label={t("acl.mode")}
          rules={[{ required: true }]}
        >
          <Select
            options={[
              { value: "manual", label: t("acl.modeManual") },
              { value: "remote", label: t("acl.modeRemote") },
            ]}
          />
        </Form.Item>
        {!isManual ? (
          <>
            <Form.Item
              name="source_url"
              label="URL"
              rules={[
                { required: true, message: t("acl.urlRequired") },
                { type: "url", message: t("acl.urlInvalid") },
              ]}
              extra={t("acl.urlHelp")}
            >
              <Input placeholder="https://example.com/lists/my.txt" />
            </Form.Item>
            <Form.Item
              name="poll_interval_minutes"
              label={t("acl.pollInterval")}
              rules={[{ required: true, message: t("acl.intervalRequired") }]}
            >
              <InputNumber min={1} max={10080} className="w-full" />
            </Form.Item>
          </>
        ) : null}
        <Form.Item
          name="body"
          label={t("acl.values")}
          rules={[
            {
              validator: async (_, value) => {
                if (!isManual) return;
                if (!value || !String(value).trim()) {
                  throw new Error(t("acl.valuesRequired"));
                }
              },
            },
          ]}
          getValueFromEvent={(value: string) => value}
          extra={
            isManual ? t("acl.valuesManualExtra") : t("acl.valuesAuto")
          }
        >
          <ProxyACLPatternEditor
            fitModalBody
            resetKey={patternEditorKey ?? "list-body"}
            placeholder={t("acl.valuesPlaceholder")}
            readOnly={!isManual}
            readOnlyMuted={!isManual}
          />
        </Form.Item>
      </Form>
      </Spin>
    </Modal>
  );
}

export function listBodyLineCount(body: string): number {
  return countPatternLines(body);
}
