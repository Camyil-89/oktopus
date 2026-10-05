"use client";

import {
  getReportsDashboard,
  patchReportsDashboard,
  runProxyAccessLogReport,
  runProxyAccessLogReportTable,
} from "@/api/proxy";
import {
  AccessLogReportWidgetGrid,
  type ReportTableFetchFn,
  type ReportWidgetQueryMeta,
} from "@/assets/components/accessLog/AccessLogReportWidgets";
import {
  buildAccessLogReportAiPrompt,
  defaultSummaryReportSpec,
} from "@/assets/accessLog/reportSpecReference";
import type { ProxyRuleKind } from "@/assets/hooks/useProxyRuleNameMap";
import { useProxyRuleNameMap } from "@/assets/hooks/useProxyRuleNameMap";
import { ReportDashboardAddTabModal } from "@/assets/modals/ReportDashboardAddTabModal";
import { ReportDashboardDeleteTabModal } from "@/assets/modals/ReportDashboardDeleteTabModal";
import { ReportDashboardEditModal } from "@/assets/modals/ReportDashboardEditModal";
import { ProxyInspectRuleViewModal } from "@/assets/modals/ProxyInspectRuleViewModal";
import type {
  AccessLogReportSpec,
  AccessLogReportWidgetData,
  AccessLogReportWidgetResult,
  AccessLogReportWidgetSpec,
} from "@/types/accessLogReport";
import type {
  ReportsDashboardDocument,
  ReportsDashboardTab,
} from "@/types/reportsDashboard";
import { applyAutoTimeBucketsToSpec } from "@/utils/accessLogReportBucket";
import {
  createDashboardTab,
  normalizeReportsDashboard,
} from "@/utils/reportsDashboard";
import {
  CopyOutlined,
  EditOutlined,
  PlusOutlined,
  ReloadOutlined,
} from "@ant-design/icons";
import { App, Button, DatePicker, Tabs } from "antd";
import type { Dayjs } from "dayjs";
import dayjs from "dayjs";
import Link from "next/link";
import { useApiErrorMessage, useTranslation } from "@/contexts/LocaleContext";
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

function buildWidgetQueryMeta(
  widgets: AccessLogReportWidgetSpec[],
): Record<string, ReportWidgetQueryMeta> {
  const m: Record<string, ReportWidgetQueryMeta> = {};
  for (const w of widgets) {
    m[w.id] = {
      groupBy: w.query.group_by,
      groupByCols: w.query.group_by_cols,
      searchColumns: w.query.search_columns,
      groupByTime: w.query.group_by_time,
    };
  }
  return m;
}

function tabToSpec(
  tab: ReportsDashboardTab,
  range: [Dayjs, Dayjs],
): AccessLogReportSpec {
  const time = rangeToSpecTime(range);
  return {
    version: 1,
    time,
    filters: tab.filters ?? {},
    widgets: tab.widgets,
  };
}

type TabRunState = {
  widgets: AccessLogReportWidgetResult[];
  appliedSpec: AccessLogReportSpec | null;
  loaded: boolean;
};

export default function ManageAccessLogReportsPage() {
  const { message } = App.useApp();
  const { t } = useTranslation();
  const formatApiError = useApiErrorMessage();
  const { ruleNames, ruleKinds } = useProxyRuleNameMap();
  const [viewRule, setViewRule] = useState<{
    id: string;
    kind: ProxyRuleKind;
  } | null>(null);

  const [activeTab, setActiveTab] = useState("summary");
  const [dashboard, setDashboard] = useState<ReportsDashboardDocument>(() =>
    normalizeReportsDashboard(null),
  );
  const [dashboardLoading, setDashboardLoading] = useState(true);
  const saveTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const dashboardRef = useRef(dashboard);
  dashboardRef.current = dashboard;

  const [summaryRange, setSummaryRange] = useState<[Dayjs, Dayjs]>(() =>
    defaultRange(),
  );
  const [summaryWidgets, setSummaryWidgets] = useState<
    AccessLogReportWidgetResult[]
  >([]);
  const [summaryLoading, setSummaryLoading] = useState(false);
  const summaryLoadedRef = useRef(false);

  const [tabRanges, setTabRanges] = useState<Record<string, [Dayjs, Dayjs]>>({});
  const [tabRuns, setTabRuns] = useState<Record<string, TabRunState>>({});
  const [tabLoadingId, setTabLoadingId] = useState<string | null>(null);

  const [addTabOpen, setAddTabOpen] = useState(false);
  const [deleteTabId, setDeleteTabId] = useState<string | null>(null);
  const [dashboardEditTabId, setDashboardEditTabId] = useState<string | null>(
    null,
  );
  const [aiPromptText, setAiPromptText] = useState<string | null>(null);
  const [aiPromptPreparing, setAiPromptPreparing] = useState(true);

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
          message.error(formatApiError(e, t("reports.promptFailed")));
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
  }, [formatApiError, message, t]);

  const persistDashboard = useCallback(
    (doc: ReportsDashboardDocument) => {
      if (saveTimerRef.current) {
        clearTimeout(saveTimerRef.current);
      }
      saveTimerRef.current = setTimeout(() => {
        void patchReportsDashboard(doc).catch((e) => {
          message.error(formatApiError(e, t("reports.dashboardSaveFailed")));
        });
      }, 400);
    },
    [formatApiError, message, t],
  );

  const updateDashboard = useCallback(
    (updater: (prev: ReportsDashboardDocument) => ReportsDashboardDocument) => {
      setDashboard((prev) => {
        const next = updater(prev);
        persistDashboard(next);
        return next;
      });
    },
    [persistDashboard],
  );

  useEffect(() => {
    let cancelled = false;
    setDashboardLoading(true);
    void getReportsDashboard()
      .then((doc) => {
        if (!cancelled) {
          setDashboard(normalizeReportsDashboard(doc));
        }
      })
      .catch((e) => {
        if (!cancelled) {
          message.error(formatApiError(e, t("reports.dashboardLoadFailed")));
        }
      })
      .finally(() => {
        if (!cancelled) {
          setDashboardLoading(false);
        }
      });
    return () => {
      cancelled = true;
      if (saveTimerRef.current) {
        clearTimeout(saveTimerRef.current);
      }
    };
  }, [formatApiError, message, t]);

  const summarySpec = useMemo((): AccessLogReportSpec => {
    const range = rangeToSpecTime(summaryRange);
    return applyAutoTimeBucketsToSpec(
      defaultSummaryReportSpec(range.from, range.to),
    );
  }, [summaryRange]);

  const runSpec = useCallback(async (spec: AccessLogReportSpec) => {
    const res = await runProxyAccessLogReport(applyAutoTimeBucketsToSpec(spec));
    return res.widgets;
  }, []);

  const loadSummary = useCallback(async () => {
    setSummaryLoading(true);
    try {
      const widgets = await runSpec(summarySpec);
      setSummaryWidgets(widgets);
      summaryLoadedRef.current = true;
    } catch (e) {
      message.error(formatApiError(e, t("reports.summaryLoadFailed")));
    } finally {
      setSummaryLoading(false);
    }
  }, [formatApiError, message, runSpec, summarySpec, t]);

  useEffect(() => {
    if (activeTab === "summary" && !summaryLoadedRef.current) {
      void loadSummary();
    }
  }, [activeTab, loadSummary]);

  const getTabRange = useCallback(
    (tabId: string): [Dayjs, Dayjs] => tabRanges[tabId] ?? defaultRange(),
    [tabRanges],
  );

  const runTabReport = useCallback(
    async (tabId: string, doc: ReportsDashboardDocument) => {
      const tab = doc.tabs.find((x) => x.id === tabId);
      if (!tab || tab.widgets.length === 0) {
        setTabRuns((prev) => ({
          ...prev,
          [tabId]: { widgets: [], appliedSpec: null, loaded: true },
        }));
        return;
      }
      const range = getTabRange(tabId);
      const spec = applyAutoTimeBucketsToSpec(tabToSpec(tab, range));
      setTabLoadingId(tabId);
      try {
        const widgets = await runSpec(spec);
        setTabRuns((prev) => ({
          ...prev,
          [tabId]: { widgets, appliedSpec: spec, loaded: true },
        }));
      } catch (e) {
        message.error(formatApiError(e, t("reports.runFailed")));
      } finally {
        setTabLoadingId(null);
      }
    },
    [formatApiError, getTabRange, message, runSpec, t],
  );

  useEffect(() => {
    if (activeTab === "summary") {
      return;
    }
    if (!dashboard.tabs.some((x) => x.id === activeTab)) {
      return;
    }
    const state = tabRuns[activeTab];
    if (!state?.loaded) {
      void runTabReport(activeTab, dashboardRef.current);
    }
  }, [activeTab, dashboard.tabs, runTabReport, tabRuns]);

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

  const makeTableFetcher = useCallback(
    (spec: AccessLogReportSpec | null): ReportTableFetchFn | undefined => {
      if (!spec) {
        return undefined;
      }
      return async (widgetId, params) => {
        const widget = spec.widgets.find((w) => w.id === widgetId);
        if (!widget || widget.type !== "table") {
          throw new Error(t("reports.widgetNotFound"));
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
    [t],
  );

  const summaryTableFetcher = useMemo(
    () => makeTableFetcher(summarySpec),
    [makeTableFetcher, summarySpec],
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
    (tabId: string, widgetId: string, data: AccessLogReportWidgetData) => {
      setTabRuns((prev) => {
        const cur = prev[tabId];
        if (!cur) {
          return prev;
        }
        return {
          ...prev,
          [tabId]: {
            ...cur,
            widgets: cur.widgets.map((w) =>
              w.id === widgetId ? { ...w, data } : w,
            ),
          },
        };
      });
    },
    [],
  );

  const confirmAddTab = (title: string) => {
    const tab = createDashboardTab(title);
    updateDashboard((prev) => ({
      ...prev,
      tabs: [...prev.tabs, tab],
    }));
    setTabRanges((prev) => ({ ...prev, [tab.id]: defaultRange() }));
    setAddTabOpen(false);
    setActiveTab(tab.id);
  };

  const removeTab = (tabId: string) => {
    updateDashboard((prev) => ({
      ...prev,
      tabs: prev.tabs.filter((x) => x.id !== tabId),
    }));
    setTabRanges((prev) => {
      const next = { ...prev };
      delete next[tabId];
      return next;
    });
    setTabRuns((prev) => {
      const next = { ...prev };
      delete next[tabId];
      return next;
    });
    if (activeTab === tabId) {
      setActiveTab("summary");
    }
  };

  const confirmRemoveTab = (tabId: string) => {
    setDeleteTabId(tabId);
  };

  const deleteTabTitle =
    deleteTabId != null
      ? dashboard.tabs.find((x) => x.id === deleteTabId)?.title ?? ""
      : "";

  const confirmDeleteTab = () => {
    if (deleteTabId == null) {
      return;
    }
    removeTab(deleteTabId);
    setDeleteTabId(null);
  };

  const applyTabWidgetsEdit = (
    tabId: string,
    widgets: AccessLogReportWidgetSpec[],
  ) => {
    setDashboardEditTabId(null);
    setTabRuns((prev) => ({
      ...prev,
      [tabId]: {
        widgets: [],
        appliedSpec: null,
        loaded: false,
      },
    }));
    setDashboard((prev) => {
      const next: ReportsDashboardDocument = {
        ...prev,
        tabs: prev.tabs.map((tab) =>
          tab.id === tabId ? { ...tab, widgets } : tab,
        ),
      };
      persistDashboard(next);
      void runTabReport(tabId, next);
      return next;
    });
  };

  const dashboardEditTab =
    dashboardEditTabId != null
      ? dashboard.tabs.find((t) => t.id === dashboardEditTabId) ?? null
      : null;

  const copyAiPrompt = async () => {
    if (!aiPromptText) {
      return;
    }
    try {
      await navigator.clipboard.writeText(aiPromptText);
      message.success(t("reports.promptCopied"));
    } catch {
      message.error(t("common.copyFailed"));
    }
  };

  const removeWidget = (tabId: string, widgetId: string) => {
    setDashboard((prev) => {
      const next: ReportsDashboardDocument = {
        ...prev,
        tabs: prev.tabs.map((tab) =>
          tab.id === tabId
            ? {
                ...tab,
                widgets: tab.widgets.filter((w) => w.id !== widgetId),
              }
            : tab,
        ),
      };
      persistDashboard(next);
      void runTabReport(tabId, next);
      return next;
    });
  };

  const customTabItems = dashboard.tabs.map((tab) => {
    const range = getTabRange(tab.id);
    const run = tabRuns[tab.id];
    const spec =
      run?.appliedSpec ??
      applyAutoTimeBucketsToSpec(tabToSpec(tab, range));
    const loading = tabLoadingId === tab.id;
    const customMetrics: Record<string, string> = {};
    for (const w of tab.widgets) {
      customMetrics[w.id] = w.query.metric ?? "count";
    }
    return {
      key: tab.id,
      label: tab.title,
      closable: true,
      children: (
        <div className="flex flex-col gap-4">
          <div className="flex flex-row flex-wrap items-center gap-2">
            <DatePicker.RangePicker
              showTime
              value={range}
              onChange={(vals) => {
                const from = vals?.[0];
                const to = vals?.[1];
                if (!from || !to) {
                  return;
                }
                setTabRanges((prev) => ({
                  ...prev,
                  [tab.id]: [from, to],
                }));
                setTabRuns((prev) => ({
                  ...prev,
                  [tab.id]: {
                    widgets: [],
                    appliedSpec: null,
                    loaded: false,
                  },
                }));
              }}
            />
            <Button
              icon={<ReloadOutlined />}
              loading={loading}
              onClick={() => void runTabReport(tab.id, dashboardRef.current)}
            >
              {t("common.refresh")}
            </Button>
            <Button
              icon={<EditOutlined />}
              onClick={() => setDashboardEditTabId(tab.id)}
            >
              {t("reports.editDashboard")}
            </Button>
            <Button
              icon={<CopyOutlined />}
              loading={aiPromptPreparing}
              disabled={!aiPromptText}
              onClick={() => void copyAiPrompt()}
            >
              {t("reports.aiPrompt")}
            </Button>
          </div>
          <AccessLogReportWidgetGrid
            widgets={run?.widgets ?? []}
            placeholderSpecs={tab.widgets}
            loading={loading || dashboardLoading}
            widgetMetrics={customMetrics}
            widgetQueryMeta={buildWidgetQueryMeta(spec.widgets)}
            rules={ruleLabelContext}
            onFetchTable={makeTableFetcher(spec)}
            onTableWidgetData={(widgetId, data) =>
              patchCustomWidgetData(tab.id, widgetId, data)
            }
            onRemoveWidget={(widgetId) => removeWidget(tab.id, widgetId)}
          />
        </div>
      ),
    };
  });

  return (
    <div className="flex w-full flex-col gap-4">
      <header className="flex flex-row flex-wrap items-center justify-between gap-3">
        <Link
          href="/manage/access-log"
          className="text-[12px] text-zinc-500 transition hover:text-teal-300"
        >
          {t("accessLog.backToLog")}
        </Link>
      </header>

      <Tabs
        activeKey={activeTab}
        onChange={setActiveTab}
        type="editable-card"
        hideAdd
        destroyOnHidden
        onEdit={(key, action) => {
          if (action === "remove" && typeof key === "string") {
            confirmRemoveTab(key);
          }
        }}
        tabBarExtraContent={
          <Button
            type="dashed"
            size="small"
            icon={<PlusOutlined />}
            onClick={() => setAddTabOpen(true)}
          >
            {t("reports.addTab")}
          </Button>
        }
        items={[
          {
            key: "summary",
            label: t("reports.summaryTab"),
            closable: false,
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
                    {t("common.refresh")}
                  </Button>
                </div>
                <AccessLogReportWidgetGrid
                  widgets={summaryWidgets}
                  placeholderSpecs={summarySpec.widgets}
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
          ...customTabItems,
        ]}
      />

      <ReportDashboardAddTabModal
        open={addTabOpen}
        onCancel={() => setAddTabOpen(false)}
        onConfirm={confirmAddTab}
      />

      <ReportDashboardDeleteTabModal
        open={deleteTabId != null}
        tabTitle={deleteTabTitle}
        onCancel={() => setDeleteTabId(null)}
        onConfirm={confirmDeleteTab}
      />

      <ReportDashboardEditModal
        open={dashboardEditTab != null}
        tab={dashboardEditTab}
        onCancel={() => setDashboardEditTabId(null)}
        onSubmit={(widgets) => {
          if (dashboardEditTabId) {
            applyTabWidgetsEdit(dashboardEditTabId, widgets);
          }
        }}
      />

      <ProxyInspectRuleViewModal
        open={viewRule?.kind === "inspect"}
        ruleId={viewRule?.kind === "inspect" ? viewRule.id : null}
        onClose={() => setViewRule(null)}
      />
    </div>
  );
}
