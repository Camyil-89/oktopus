"use client";

import { ApiError } from "@/api/base";
import {
  runProxyAccessLogReport,
  runProxyAccessLogReportTable,
} from "@/api/proxy";
import {
  ACCESS_LOG_REPORT_EXAMPLE,
  buildAccessLogReportAiPrompt,
  defaultSummaryReportSpec,
} from "@/assets/accessLog/reportSpecReference";
import {
  AccessLogReportWidgetGrid,
  type ReportTableFetchFn,
  type ReportWidgetQueryMeta,
} from "@/assets/components/accessLog/AccessLogReportWidgets";
import type { ProxyRuleKind } from "@/assets/hooks/useProxyRuleNameMap";
import { useProxyRuleNameMap } from "@/assets/hooks/useProxyRuleNameMap";
import { ProxyInspectRuleViewModal } from "@/assets/modals/ProxyInspectRuleViewModal";
import type {
  AccessLogReportSpec,
  AccessLogReportWidgetData,
  AccessLogReportWidgetResult,
  AccessLogReportWidgetSpec,
} from "@/types/accessLogReport";
import { parseAccessLogReportSpec } from "@/utils/accessLogReportValidate";
import { downloadTextFile } from "@/utils/downloadText";
import { CopyOutlined, DownloadOutlined, ReloadOutlined } from "@ant-design/icons";
import { App, Button, DatePicker, Input, Tabs } from "antd";
import type { Dayjs } from "dayjs";
import dayjs from "dayjs";
import Link from "next/link";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";

function defaultRange(): [Dayjs, Dayjs] {
  const to = dayjs();
  const from = to.subtract(30, "day");
  return [from, to];
}

function rangeToSpecTime(range: [Dayjs, Dayjs]): { from: string; to: string } {
  return {
    from: range[0].toDate().toISOString(),
    to: range[1].toDate().toISOString(),
  };
}

const CUSTOM_SPEC_DEFAULT = JSON.stringify(ACCESS_LOG_REPORT_EXAMPLE, null, 2);

function buildWidgetQueryMeta(
  widgets: AccessLogReportWidgetSpec[],
): Record<string, ReportWidgetQueryMeta> {
  const m: Record<string, ReportWidgetQueryMeta> = {};
  for (const w of widgets) {
    m[w.id] = {
      groupBy: w.query.group_by,
      groupByCols: w.query.group_by_cols,
      searchColumns: w.query.search_columns,
    };
  }
  return m;
}

export default function ManageAccessLogReportsPage() {
  const { message } = App.useApp();
  const { ruleNames, ruleKinds } = useProxyRuleNameMap();
  const [viewRule, setViewRule] = useState<{
    id: string;
    kind: ProxyRuleKind;
  } | null>(null);
  const [activeTab, setActiveTab] = useState("summary");

  const [summaryRange, setSummaryRange] = useState<[Dayjs, Dayjs]>(() =>
    defaultRange(),
  );
  const [summaryWidgets, setSummaryWidgets] = useState<
    AccessLogReportWidgetResult[]
  >([]);
  const [summaryLoading, setSummaryLoading] = useState(false);
  const summaryLoadedRef = useRef(false);

  const [customText, setCustomText] = useState(CUSTOM_SPEC_DEFAULT);
  const [customWidgets, setCustomWidgets] = useState<
    AccessLogReportWidgetResult[]
  >([]);
  const [customLoading, setCustomLoading] = useState(false);
  const [aiPromptText, setAiPromptText] = useState<string | null>(null);
  const [aiPromptPreparing, setAiPromptPreparing] = useState(true);
  const [customAppliedSpec, setCustomAppliedSpec] =
    useState<AccessLogReportSpec | null>(null);

  useEffect(() => {
    let cancelled = false;
    setAiPromptPreparing(true);
    setAiPromptText(null);
    void buildAccessLogReportAiPrompt()
      .then((text) => {
        if (!cancelled) {
          setAiPromptText(text);
        }
      })
      .catch((e) => {
        if (!cancelled) {
          setAiPromptText(null);
          const msg =
            e instanceof ApiError
              ? e.message
              : "Не удалось подготовить промпт";
          message.error(msg);
        }
      })
      .finally(() => {
        if (!cancelled) {
          setAiPromptPreparing(false);
        }
      });
    return () => {
      cancelled = true;
    };
  }, [message]);

  const summarySpec = useMemo((): AccessLogReportSpec => {
    const t = rangeToSpecTime(summaryRange);
    return defaultSummaryReportSpec(t.from, t.to);
  }, [summaryRange]);

  const runSpec = useCallback(
    async (spec: AccessLogReportSpec) => {
      const res = await runProxyAccessLogReport(spec);
      return res.widgets;
    },
    [],
  );

  const loadSummary = useCallback(async () => {
    setSummaryLoading(true);
    try {
      const widgets = await runSpec(summarySpec);
      setSummaryWidgets(widgets);
      summaryLoadedRef.current = true;
    } catch (e) {
      const msg =
        e instanceof ApiError ? e.message : "Не удалось загрузить сводку";
      message.error(msg);
    } finally {
      setSummaryLoading(false);
    }
  }, [message, runSpec, summarySpec]);

  useEffect(() => {
    if (activeTab === "summary" && !summaryLoadedRef.current) {
      void loadSummary();
    }
  }, [activeTab, loadSummary]);

  const onSummaryTab = (key: string) => {
    setActiveTab(key);
  };

  const runCustom = async () => {
    setCustomLoading(true);
    try {
      const spec = parseAccessLogReportSpec(customText);
      const widgets = await runSpec(spec);
      setCustomAppliedSpec(spec);
      setCustomWidgets(widgets);
    } catch (e) {
      const msg =
        e instanceof ApiError
          ? e.message
          : e instanceof Error
            ? e.message
            : "Не удалось выполнить отчёт";
      message.error(msg);
    } finally {
      setCustomLoading(false);
    }
  };

  const copyPrompt = async () => {
    if (!aiPromptText) {
      return;
    }
    try {
      await navigator.clipboard.writeText(aiPromptText);
      message.success("Промпт скопирован");
    } catch {
      message.error("Не удалось скопировать");
    }
  };

  const downloadPrompt = () => {
    if (!aiPromptText) {
      return;
    }
    const stamp = new Date().toISOString().replace(/[:.]/g, "-").slice(0, 19);
    downloadTextFile(aiPromptText, `oktopus-access-log-report-prompt-${stamp}.txt`);
  };

  const ruleLabelContext = useMemo(
    () => ({
      ruleNames,
      ruleKinds,
      onOpenRule: (id: string, kind: ProxyRuleKind) =>
        setViewRule({ id, kind }),
    }),
    [ruleNames, ruleKinds],
  );

  const summaryWidgetQueryMeta = useMemo(
    () => buildWidgetQueryMeta(summarySpec.widgets),
    [summarySpec.widgets],
  );

  const widgetMetrics = useMemo(() => {
    const m: Record<string, string> = {};
    for (const w of summarySpec.widgets) {
      m[w.id] = w.query.metric ?? "count";
    }
    return m;
  }, [summarySpec.widgets]);

  const customWidgetMetrics = useMemo(() => {
    try {
      const spec = parseAccessLogReportSpec(customText);
      const m: Record<string, string> = {};
      for (const w of spec.widgets) {
        m[w.id] = w.query.metric ?? "count";
      }
      return m;
    } catch {
      return {};
    }
  }, [customText]);

  const customWidgetQueryMeta = useMemo(
    () =>
      customAppliedSpec
        ? buildWidgetQueryMeta(customAppliedSpec.widgets)
        : {},
    [customAppliedSpec],
  );

  const makeTableFetcher = useCallback(
    (spec: AccessLogReportSpec | null): ReportTableFetchFn | undefined => {
      if (!spec) {
        return undefined;
      }
      return async (widgetId, params) => {
        const widget = spec.widgets.find((w) => w.id === widgetId);
        if (!widget || widget.type !== "table") {
          throw new Error("виджет не найден");
        }
        const res = await runProxyAccessLogReportTable({
          version: spec.version,
          time: spec.time,
          filters: spec.filters,
          widget,
          page: params.page,
          page_size: params.page_size,
          search: params.search,
        });
        return res.data;
      };
    },
    [],
  );

  const summaryTableFetcher = useMemo(
    () => makeTableFetcher(summarySpec),
    [makeTableFetcher, summarySpec],
  );

  const customTableFetcher = useMemo(
    () => makeTableFetcher(customAppliedSpec),
    [customAppliedSpec, makeTableFetcher],
  );

  const patchSummaryWidgetData = useCallback(
    (widgetId: string, data: AccessLogReportWidgetData) => {
      setSummaryWidgets((prev) =>
        prev.map((w) => (w.id === widgetId ? { ...w, data } : w)),
      );
    },
    [],
  );

  const patchCustomWidgetData = useCallback(
    (widgetId: string, data: AccessLogReportWidgetData) => {
      setCustomWidgets((prev) =>
        prev.map((w) => (w.id === widgetId ? { ...w, data } : w)),
      );
    },
    [],
  );

  return (
    <div className="flex w-full flex-col gap-4">
      <header className="flex flex-row flex-wrap items-center justify-between gap-3">
        <Link
          href="/manage/access-log"
          className="text-[12px] text-zinc-500 transition hover:text-teal-300"
        >
          ← Журнал доступа
        </Link>
      </header>

      <Tabs
        activeKey={activeTab}
        onChange={onSummaryTab}
        items={[
          {
            key: "summary",
            label: "Сводка",
            children: (
              <div className="flex flex-col gap-4">
                <div className="flex flex-row flex-wrap items-center gap-2">
                  <DatePicker.RangePicker
                    showTime
                    value={summaryRange}
                    onChange={(vals) => {
                      if (vals?.[0] && vals[1]) {
                        setSummaryRange([vals[0], vals[1]]);
                        summaryLoadedRef.current = false;
                      }
                    }}
                  />
                  <Button
                    icon={<ReloadOutlined />}
                    loading={summaryLoading}
                    onClick={() => void loadSummary()}
                  >
                    Обновить
                  </Button>
                </div>
                <AccessLogReportWidgetGrid
                  widgets={summaryWidgets}
                  loading={summaryLoading}
                  widgetMetrics={widgetMetrics}
                  widgetQueryMeta={summaryWidgetQueryMeta}
                  rules={ruleLabelContext}
                  onFetchTable={summaryTableFetcher}
                  onTableWidgetData={patchSummaryWidgetData}
                />
              </div>
            ),
          },
          {
            key: "custom",
            label: "Свой отчёт",
            children: (
              <div className="flex flex-col gap-4">
                <div className="flex flex-row flex-wrap gap-2">
                  <Button
                    type="primary"
                    loading={customLoading}
                    onClick={() => void runCustom()}
                  >
                    Выполнить
                  </Button>
                  <Button
                    icon={<CopyOutlined />}
                    loading={aiPromptPreparing}
                    disabled={!aiPromptText}
                    onClick={() => void copyPrompt()}
                  >
                    Промпт для ИИ
                  </Button>
                  <Button
                    icon={<DownloadOutlined />}
                    loading={aiPromptPreparing}
                    disabled={!aiPromptText}
                    onClick={downloadPrompt}
                  >
                    Скачать .txt
                  </Button>
                </div>
                <Input.TextArea
                  value={customText}
                  onChange={(e) => setCustomText(e.target.value)}
                  rows={16}
                  className="font-mono text-[12px]"
                  spellCheck={false}
                />
                <AccessLogReportWidgetGrid
                  widgets={customWidgets}
                  loading={customLoading}
                  widgetMetrics={customWidgetMetrics}
                  widgetQueryMeta={customWidgetQueryMeta}
                  rules={ruleLabelContext}
                  onFetchTable={customTableFetcher}
                  onTableWidgetData={patchCustomWidgetData}
                />
              </div>
            ),
          },
        ]}
      />

      <ProxyInspectRuleViewModal
        open={viewRule?.kind === "inspect"}
        ruleId={viewRule?.kind === "inspect" ? viewRule.id : null}
        onClose={() => setViewRule(null)}
      />
    </div>
  );
}
