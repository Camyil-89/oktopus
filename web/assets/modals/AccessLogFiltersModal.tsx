"use client";

import { scrollableModalProps } from "@/assets/modals/modalConfig";
import { listProxyInspectRules } from "@/api/proxy";
import { Button, Checkbox, Collapse, DatePicker, Form, Input, Modal, Select } from "antd";
import type { Dayjs } from "dayjs";
import dayjs from "dayjs";
import type { AccessLogActionFilter } from "@/utils/accessLogFilters";
import { useEffect, useMemo, useState } from "react";

export type AccessLogFiltersValues = {
  id: string;
  user: string;
  source: string;
  destination: string;
  url: string;
  search_only: boolean;
  from: string;
  to: string;
  action: AccessLogActionFilter;
  decision_rule_ref: string;
  inspect_rule_id: string;
};

export const ACCESS_LOG_EMPTY_FILTERS: AccessLogFiltersValues = {
  id: "",
  user: "",
  source: "",
  destination: "",
  url: "",
  search_only: false,
  from: "",
  to: "",
  action: "",
  decision_rule_ref: "",
  inspect_rule_id: "",
};

type PeriodValue = [Dayjs | null, Dayjs | null] | null;

type FormValues = Omit<AccessLogFiltersValues, "from" | "to"> & {
  period?: PeriodValue;
};

type AccessLogFiltersModalProps = {
  open: boolean;
  initialValues: AccessLogFiltersValues;
  onCancel: () => void;
  onApply: (values: AccessLogFiltersValues) => void;
};

const FORM_ITEM_CLASS = "!mb-2";

function periodFromFilterValues(values: AccessLogFiltersValues): PeriodValue {
  if (!values.from && !values.to) {
    return null;
  }
  const from = values.from ? dayjs(values.from) : null;
  const to = values.to ? dayjs(values.to) : null;
  if (from && !from.isValid()) {
    return null;
  }
  if (to && !to.isValid()) {
    return null;
  }
  return [from, to];
}

function formValuesFromFilters(values: AccessLogFiltersValues): FormValues {
  return {
    ...values,
    period: periodFromFilterValues(values),
  };
}

function collapseKeysForFilters(values: AccessLogFiltersValues): string[] {
  const keys: string[] = [];
  if (values.from || values.to) {
    keys.push("period");
  }
  if (values.action || values.decision_rule_ref || values.inspect_rule_id) {
    keys.push("rules");
  }
  if (
    values.user ||
    values.source ||
    values.destination ||
    values.url ||
    values.search_only
  ) {
    keys.push("request");
  }
  return keys;
}

function filtersFromForm(values: FormValues): AccessLogFiltersValues {
  const period = values.period;
  return {
    id: values.id?.trim() ?? "",
    user: values.user?.trim() ?? "",
    source: values.source?.trim() ?? "",
    destination: values.destination?.trim() ?? "",
    url: values.url?.trim() ?? "",
    search_only: values.search_only ?? false,
    from: period?.[0]?.isValid() ? period[0]!.toDate().toISOString() : "",
    to: period?.[1]?.isValid() ? period[1]!.toDate().toISOString() : "",
    action: values.action ?? "",
    decision_rule_ref: values.decision_rule_ref?.trim() ?? "",
    inspect_rule_id: values.inspect_rule_id ?? "",
  };
}

export function AccessLogFiltersModal({
  open,
  initialValues,
  onCancel,
  onApply,
}: AccessLogFiltersModalProps) {
  const [form] = Form.useForm<FormValues>();
  const [collapseOpen, setCollapseOpen] = useState<string[]>([]);
  const [inspectOptions, setInspectOptions] = useState<
    { value: string; label: string }[]
  >([]);

  useEffect(() => {
    if (open) {
      form.setFieldsValue(formValuesFromFilters(initialValues));
      setCollapseOpen(collapseKeysForFilters(initialValues));
    }
  }, [open, initialValues, form]);

  useEffect(() => {
    if (!open) {
      return;
    }
    let cancelled = false;
    void listProxyInspectRules()
      .then((inspect) => {
        if (cancelled) {
          return;
        }
        setInspectOptions(
          inspect.map((r) => ({ value: r.id, label: r.name || r.id })),
        );
      })
      .catch(() => {
        if (!cancelled) {
          setInspectOptions([]);
        }
      });
    return () => {
      cancelled = true;
    };
  }, [open]);

  const collapseItems = useMemo(
    () => [
      {
        key: "period",
        label: "Период",
        children: (
          <Form.Item name="period" className={FORM_ITEM_CLASS}>
            <DatePicker.RangePicker
              size="small"
              showTime
              allowEmpty={[true, true]}
              className="w-full"
              format="DD.MM.YYYY HH:mm"
            />
          </Form.Item>
        ),
      },
      {
        key: "rules",
        label: "Решение и правила",
        children: (
          <>
            <Form.Item name="action" label="Действие" className={FORM_ITEM_CLASS}>
              <Select
                size="small"
                allowClear
                placeholder="Любое"
                options={[
                  { value: "1", label: "Разрешено" },
                  { value: "0", label: "Запрещено" },
                  { value: "errors", label: "Только с ошибкой" },
                ]}
              />
            </Form.Item>
            <Form.Item
              name="decision_rule_ref"
              label="Правило ACL"
              className={FORM_ITEM_CLASS}
            >
              <Input
                size="small"
                allowClear
                autoComplete="off"
                placeholder="http_access allow all"
              />
            </Form.Item>
            <Form.Item
              name="inspect_rule_id"
              label="Правило инспекции"
              className={FORM_ITEM_CLASS}
            >
              <Select
                size="small"
                allowClear
                showSearch
                optionFilterProp="label"
                placeholder="Любое"
                options={inspectOptions}
              />
            </Form.Item>
          </>
        ),
      },
      {
        key: "request",
        label: "Запрос",
        children: (
          <>
            <Form.Item name="user" label="Пользователь" className={FORM_ITEM_CLASS}>
              <Input size="small" allowClear autoComplete="off" />
            </Form.Item>
            <Form.Item name="source" label="Источник" className={FORM_ITEM_CLASS}>
              <Input size="small" allowClear autoComplete="off" />
            </Form.Item>
            <Form.Item name="destination" label="Назначение" className={FORM_ITEM_CLASS}>
              <Input size="small" allowClear autoComplete="off" />
            </Form.Item>
            <Form.Item name="url" label="URL" className={FORM_ITEM_CLASS}>
              <Input size="small" allowClear autoComplete="off" />
            </Form.Item>
            <Form.Item name="search_only" valuePropName="checked" className="!mb-0">
              <Checkbox className="text-sm">Только поисковые запросы</Checkbox>
            </Form.Item>
          </>
        ),
      },
    ],
    [inspectOptions],
  );

  const close = () => {
    form.resetFields();
    onCancel();
  };

  const submit = () => {
    form.validateFields().then((values) => {
      onApply({
        ...filtersFromForm(values),
        id: initialValues.id,
      });
    });
  };

  const resetFilters = () => {
    form.setFieldsValue(formValuesFromFilters(ACCESS_LOG_EMPTY_FILTERS));
    setCollapseOpen([]);
    onApply({ ...ACCESS_LOG_EMPTY_FILTERS });
  };

  return (
    <Modal
      title="Фильтры"
      open={open}
      width={480}
      onCancel={close}
      footer={
        <div className="flex flex-row flex-wrap items-center justify-between gap-2">
          <Button size="small" onClick={resetFilters}>
            Сбросить
          </Button>
          <div className="flex flex-row gap-2">
            <Button size="small" onClick={close}>
              Отмена
            </Button>
            <Button size="small" type="primary" onClick={submit}>
              Найти
            </Button>
          </div>
        </div>
      }
      {...scrollableModalProps}
    >
      <Form
        form={form}
        layout="vertical"
        size="small"
        requiredMark={false}
        initialValues={initialValues}
      >
        <Collapse
          size="small"
          bordered={false}
          activeKey={collapseOpen}
          onChange={(keys) =>
            setCollapseOpen(Array.isArray(keys) ? keys : [keys])
          }
          items={collapseItems}
          className="bg-transparent"
        />
      </Form>
    </Modal>
  );
}
