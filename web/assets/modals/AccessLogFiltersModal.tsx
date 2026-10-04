"use client";

import { scrollableModalProps } from "@/assets/modals/modalConfig";
import { listProxyInspectRules } from "@/api/proxy";
import { Button, Checkbox, Collapse, DatePicker, Form, Input, Modal, Select } from "antd";
import type { Dayjs } from "dayjs";
import dayjs from "dayjs";
import type { AccessLogActionFilter } from "@/utils/accessLogFilters";
import { ACCESS_LOG_ATTACK_KINDS } from "@/utils/accessLogSegment";
import { useTranslation } from "@/contexts/LocaleContext";
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
  attack_kind: string;
  policy_anomaly_q: string;
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
  attack_kind: "",
  policy_anomaly_q: "",
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
  if (values.attack_kind || values.policy_anomaly_q) {
    keys.push("attacks");
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
    attack_kind: values.attack_kind ?? "",
    policy_anomaly_q: values.policy_anomaly_q?.trim() ?? "",
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
  const { t } = useTranslation();
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
        label: t("accessLog.filtersPeriod"),
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
        label: t("accessLog.filtersDecision"),
        children: (
          <>
            <Form.Item name="action" label={t("common.action")} className={FORM_ITEM_CLASS}>
              <Select
                size="small"
                allowClear
                placeholder={t("common.any")}
                options={[
                  { value: "1", label: t("accessLog.filterAllowed") },
                  { value: "0", label: t("accessLog.filterDenied") },
                ]}
              />
            </Form.Item>
            <Form.Item
              name="decision_rule_ref"
              label={t("accessLog.aclRule")}
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
              label={t("accessLog.inspectRule")}
              className={FORM_ITEM_CLASS}
            >
              <Select
                size="small"
                allowClear
                showSearch
                optionFilterProp="label"
                placeholder={t("common.any")}
                options={inspectOptions}
              />
            </Form.Item>
          </>
        ),
      },
      {
        key: "attacks",
        label: t("accessLog.filtersAttacks"),
        children: (
          <>
            <Form.Item
              name="attack_kind"
              label={t("accessLog.filterAttackKind")}
              className={FORM_ITEM_CLASS}
            >
              <Select
                size="small"
                allowClear
                placeholder={t("accessLog.filterAttackKindAny")}
                options={ACCESS_LOG_ATTACK_KINDS.map((k) => ({
                  value: k.id,
                  label: t(k.labelKey),
                }))}
              />
            </Form.Item>
            <Form.Item
              name="policy_anomaly_q"
              label={t("accessLog.filterPolicyAnomalyQ")}
              className={FORM_ITEM_CLASS}
            >
              <Input
                size="small"
                allowClear
                autoComplete="off"
                placeholder={t("accessLog.filterPolicyAnomalyQPlaceholder")}
              />
            </Form.Item>
          </>
        ),
      },
      {
        key: "request",
        label: t("accessLog.filtersRequest"),
        children: (
          <>
            <Form.Item name="user" label={t("common.user")} className={FORM_ITEM_CLASS}>
              <Input size="small" allowClear autoComplete="off" />
            </Form.Item>
            <Form.Item name="source" label={t("common.source")} className={FORM_ITEM_CLASS}>
              <Input size="small" allowClear autoComplete="off" />
            </Form.Item>
            <Form.Item name="destination" label={t("common.destination")} className={FORM_ITEM_CLASS}>
              <Input size="small" allowClear autoComplete="off" />
            </Form.Item>
            <Form.Item name="url" label="URL" className={FORM_ITEM_CLASS}>
              <Input size="small" allowClear autoComplete="off" />
            </Form.Item>
            <Form.Item name="search_only" valuePropName="checked" className="!mb-0">
              <Checkbox className="text-sm">{t("accessLog.searchOnly")}</Checkbox>
            </Form.Item>
          </>
        ),
      },
    ],
    [inspectOptions, t],
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
      title={t("accessLog.filtersTitle")}
      open={open}
      width={480}
      onCancel={close}
      footer={
        <div className="flex flex-row flex-wrap items-center justify-between gap-2">
          <Button size="small" onClick={resetFilters}>
            {t("accessLog.filtersReset")}
          </Button>
          <div className="flex flex-row gap-2">
            <Button size="small" onClick={close}>
              {t("common.cancel")}
            </Button>
            <Button size="small" type="primary" onClick={submit}>
              {t("common.search")}
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
