"use client";

import {
  ProxyACLPatternEditor,
  countPatternLines,
} from "@/assets/components/ProxyACLPatternEditor";
import { scrollableModalProps } from "@/assets/modals/modalConfig";
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

function modalTitle(mode: "create" | "edit"): string {
  return mode === "create" ? "Новый list" : "Редактирование list";
}

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
  const sourceMode = Form.useWatch("source_mode", form) ?? "manual";
  const isManual = sourceMode === "manual";
  const showPoll =
    mode === "edit" && !isManual && listId != null && listId !== "" && onPollNow;

  return (
    <Modal
      title={modalTitle(mode)}
      open={open}
      onCancel={onCancel}
      width={720}
      destroyOnHidden
      {...scrollableModalProps}
      footer={
        <div className="flex flex-row justify-end gap-2">
          <Button onClick={onCancel}>Отмена</Button>
          {showPoll ? (
            <Button loading={pollLoading} onClick={() => void onPollNow()}>
              Опросить
            </Button>
          ) : null}
          <Button type="primary" loading={okLoading} onClick={onOk}>
            Готово
          </Button>
        </div>
      }
    >
      <Spin spinning={modalLoading ?? false}>
      <Form form={form} layout="vertical" className="flex flex-col gap-0">
        <Form.Item
          name="name"
          label="Имя"
          rules={[{ required: true, message: "Укажите имя" }]}
        >
          <Input placeholder="имя acl в http_access" />
        </Form.Item>
        <Form.Item
          name="list_type"
          label="Тип"
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
          label="Режим"
          rules={[{ required: true }]}
        >
          <Select
            options={[
              { value: "manual", label: "Вручную" },
              { value: "remote", label: "Опрос URL" },
            ]}
          />
        </Form.Item>
        {!isManual ? (
          <>
            <Form.Item
              name="source_url"
              label="URL"
              rules={[
                { required: true, message: "Укажите URL" },
                { type: "url", message: "Некорректный URL" },
              ]}
              extra="Ответ — plain text, одно значение на строку."
            >
              <Input placeholder="https://example.com/lists/my.txt" />
            </Form.Item>
            <Form.Item
              name="poll_interval_minutes"
              label="Интервал опроса (мин)"
              rules={[{ required: true, message: "Укажите интервал" }]}
            >
              <InputNumber min={1} max={10080} className="w-full" />
            </Form.Item>
          </>
        ) : null}
        <Form.Item
          name="body"
          label="Значения"
          rules={[
            {
              validator: async (_, value) => {
                if (!isManual) return;
                if (!value || !String(value).trim()) {
                  throw new Error("Добавьте хотя бы одну строку");
                }
              },
            },
          ]}
          getValueFromEvent={(value: string) => value}
          extra={
            isManual
              ? "Plain text: одно значение на строку."
              : "Заполняется автоматически при опросе URL."
          }
        >
          <ProxyACLPatternEditor
            fitModalBody
            resetKey={patternEditorKey ?? "list-body"}
            placeholder="по одному значению на строку"
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
