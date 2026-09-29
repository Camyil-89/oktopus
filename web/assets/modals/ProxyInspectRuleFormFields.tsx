"use client";

import { MODAL_TEMPLATE_FIELD_HEIGHT_CSS } from "@/assets/modals/modalConfig";
import type { ProxyInspectRuleDraft } from "@/types/inspect";
import { Form, Input, Select, Switch, Tabs } from "antd";
import type { FormInstance } from "antd";

export type ProxyInspectRuleFormValues = ProxyInspectRuleDraft;

type ProxyInspectRuleFormFieldsProps = {
  form: FormInstance<ProxyInspectRuleFormValues>;
  readOnly?: boolean;
};

export function proxyInspectRuleToFormValues(
  rule: Pick<
    ProxyInspectRuleDraft,
    "name" | "script" | "action" | "enabled" | "sort_order"
  >,
): ProxyInspectRuleFormValues {
  return {
    name: rule.name,
    script: rule.script,
    action: rule.action,
    enabled: rule.enabled,
    sort_order: rule.sort_order,
  };
}

export function ProxyInspectRuleFormFields({
  form,
  readOnly = false,
}: ProxyInspectRuleFormFieldsProps) {
  return (
    <Form form={form} layout="vertical" className="flex flex-col gap-0">
      <Tabs
        items={[
          {
            key: "settings",
            label: "Настройки",
            children: (
              <div className="flex flex-col gap-0 pt-2">
                <Form.Item
                  name="name"
                  label="Имя"
                  rules={readOnly ? undefined : [{ required: true }]}
                >
                  <Input readOnly={readOnly} />
                </Form.Item>
                <Form.Item
                  name="action"
                  label="Если скрипт вернул совпадение"
                  rules={readOnly ? undefined : [{ required: true }]}
                >
                  <Select
                    disabled={readOnly}
                    options={[
                      { value: 0, label: "Запретить запрос" },
                      { value: 1, label: "Разрешить запрос" },
                    ]}
                  />
                </Form.Item>
                <Form.Item name="enabled" label="Включено" valuePropName="checked">
                  <Switch disabled={readOnly} />
                </Form.Item>
              </div>
            ),
          },
          {
            key: "template",
            label: "Шаблон",
            forceRender: true,
            children: (
              <div className="flex flex-col gap-0 pt-2">
                <Form.Item
                  className="mb-0"
                  name="script"
                  label="Lua: function inspect(ctx)"
                  rules={readOnly ? undefined : [{ required: true }]}
                >
                  <Input.TextArea
                    className="font-mono text-xs"
                    spellCheck={false}
                    readOnly={readOnly}
                    style={{
                      height: MODAL_TEMPLATE_FIELD_HEIGHT_CSS,
                      resize: "none",
                    }}
                  />
                </Form.Item>
              </div>
            ),
          },
        ]}
      />
    </Form>
  );
}
