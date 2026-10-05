import {
  formatAclPolicySection,
  formatInspectRulesSection,
  loadProxyRulesForReportPrompt,
} from "@/assets/accessLog/reportPromptRules";
import type { AccessLogReportSpec } from "@/types/accessLogReport";
import {
  formatInspectLogFieldsForPrompt,
  inspectLogGroupByField,
  isInspectLogFieldKey,
  mergedInspectLogFieldKeys,
  parseCtxLogKeysFromLuaScripts,
} from "@/utils/accessLogInspectLog";

export const ACCESS_LOG_REPORT_SPEC_VERSION = 1;

export const ACCESS_LOG_REPORT_COLUMN_FIELDS = [
  "instance_id",
  "destination_address",
  "source_address",
  "user",
  "action",
  "denied_by",
  "decision_rule_ref",
  "search_engine",
] as const;

/** @deprecated используйте ACCESS_LOG_REPORT_COLUMN_FIELDS + inspect_log.* */
export const ACCESS_LOG_REPORT_GROUP_BY_FIELDS =
  ACCESS_LOG_REPORT_COLUMN_FIELDS;

export function accessLogReportInspectLogGroupByFields(
  luaScripts: string[] = [],
): string[] {
  return parseCtxLogKeysFromLuaScripts(luaScripts).map((k) =>
    inspectLogGroupByField(k),
  );
}

export function accessLogReportAllGroupByFields(): string[] {
  return [
    ...ACCESS_LOG_REPORT_COLUMN_FIELDS,
    ...accessLogReportInspectLogGroupByFields(),
  ];
}

export const ACCESS_LOG_REPORT_EXAMPLE: AccessLogReportSpec = {
  version: 1,
  time: {
    from: "2026-03-01T00:00:00Z",
    to: "2026-03-02T00:00:00Z",
  },
  filters: {
    inspect_log: {
      upload_method: "POST",
    },
  },
  widgets: [
    {
      id: "hourly",
      type: "timeseries",
      title: "Запросы по часам",
      query: {
        group_by_time: "1h",
        metric: "count",
        split_by: "action",
      },
    },
    {
      id: "top_hosts",
      type: "bar",
      title: "Топ назначений",
      query: {
        group_by: "destination_address",
        metric: "count",
        limit: 12,
        order: "desc",
      },
    },
    {
      id: "upload_types",
      type: "bar",
      title: "Content-Type загрузок (ctx:log)",
      query: {
        group_by: "inspect_log.upload_content_type",
        metric: "count",
        limit: 10,
        order: "desc",
      },
    },
    {
      id: "total",
      type: "stat",
      title: "Всего записей",
      query: { metric: "count" },
    },
    {
      id: "files_by_user",
      type: "table",
      title: "Пользователь × файлы × Content-Type × метод",
      query: {
        group_by_cols: [
          "user",
          "inspect_log.upload_filenames",
          "inspect_log.upload_content_type",
          "inspect_log.upload_method",
        ],
        search_columns: [
          "user",
          "inspect_log.upload_filenames",
          "inspect_log.upload_content_type",
        ],
        metric: "count",
        order: "desc",
      },
    },
    {
      id: "by_user_action",
      type: "table",
      title: "Пользователь × действие",
      query: {
        group_by_cols: ["user", "action"],
        search_columns: ["user"],
        metric: "count",
        order: "desc",
      },
    },
  ],
};

export const ACCESS_LOG_REPORT_WIDGETS_EXAMPLE =
  ACCESS_LOG_REPORT_EXAMPLE.widgets;

export const ACCESS_LOG_REPORT_WIDGETS_REFERENCE = `Виджеты вкладки отчётов (JSON-массив, до 12 элементов)

Каждый элемент:
  id — уникальная строка в отчёте (snake_case)
  type — timeseries | bar | table | stat
  title — подпись на UI
  query — параметры агрегации

query.metric:
  count (по умолчанию) | avg_decide_duration_us

query.group_by_time (для timeseries):
  10m | 1h | 1d | hour_of_day | day_of_week | month_of_year

query.split_by (для timeseries, необязательно):
  action | denied_by

query.group_by (для bar, одно поле):
  ${ACCESS_LOG_REPORT_COLUMN_FIELDS.join(" | ")}
  | inspect_log.<ключ ctx:log>

query.group_by_cols (для table, 1–4 поля из списка group_by)

query.search_columns (для table, необязательно) — подмножество group_by_cols

query.field_nonempty — массив полей group_by: значение не пустая строка

query.limit — bar: топ-N (по умолчанию 20, макс. 100); table: page size (20, макс. 100)
query.order — asc | desc

Период (time) и filters вкладки задаются в UI панели — в JSON-ответе для «Редактировать дашборд» их не включай.`;

export const ACCESS_LOG_REPORT_API_REFERENCE = `Access Log Report Spec (version ${ACCESS_LOG_REPORT_SPEC_VERSION})

Корень JSON:
  version — только ${ACCESS_LOG_REPORT_SPEC_VERSION}
  time.from, time.to — RFC3339 UTC (обязательны), период не более 31 дня
  filters — необязательно (как в списке журнала):
    instance_id — UUID прокси-инстанса (пусто — все инстансы)
    user, source, destination, url — подстрока ILIKE
    search_only — true: только записи с непустым search_engine (ClickHouse)
    field_nonempty — массив полей group_by: значение не пустая строка
    action — 0 deny, 1 allow
    inspect_rule_id — UUID; decision_rule_ref — Squid http_access / system_*
  widgets — массив до 12 элементов

Виджет:
  id — уникальная строка в отчёте
  type — timeseries | bar | table | stat
  title — подпись на UI
  query — параметры агрегации

query.metric:
  count (по умолчанию) | avg_decide_duration_us

query.group_by_time (для timeseries):
  10m | 1h | 1d

query.split_by (для timeseries, необязательно):
  action | denied_by

query.group_by (для bar, одно поле):
  ${ACCESS_LOG_REPORT_COLUMN_FIELDS.join(" | ")}
  | inspect_log.<ключ ctx:log>

query.group_by_cols (для table, 1–4 поля из списка group_by)

query.search_columns (для table, необязательно) — подмножество group_by_cols;
  в UI доступен поиск по подстроке (ILIKE) только по этим столбцам; запросы пагинации — POST .../report/table

filters.inspect_log — объект { "<ключ ctx:log>": "подстрока" } (фильтр по extra inspect-правил)

query.field_nonempty — как filters.field_nonempty, только для виджета

query.limit — bar: топ-N (по умолчанию 20, макс. 100); table: размер страницы по умолчанию (20, макс. 100)
query.order — asc | desc (по значению метрики)

Таблица в ответе: data.columns, data.rows, data.values, data.total, data.page, data.page_size

Поля строк журнала (для формулировки задачи):
  created_at, source_address, destination_address, user, action (0|1),
  inspect_rule_id, denied_by, decision_rule_ref,
  decide_duration_us, full_url, extra.search.engine, extra.search.query

Ответ API POST /api/proxy/access-log/report — тот же список widgets с полем data:
  timeseries: data.series[{ key, label, points[{ t, value }] }]
  bar: data.items[{ label, value }]
  stat: data.value
  table: data.columns, data.rows, data.values, data.total, data.page, data.page_size`;

function mergeInspectLogKeysForPrompt(
  luaScripts: string[],
  extraInspectKeys?: string[],
): string[] {
  const keys = new Set(mergedInspectLogFieldKeys(luaScripts));
  if (extraInspectKeys?.length) {
    for (const k of extraInspectKeys) {
      if (isInspectLogFieldKey(k)) {
        keys.add(k);
      }
    }
  }
  return [...keys].sort();
}

function buildAccessLogReportAiPromptStatic(inspectKeys: string[]): string {
  const inspectLogSection = formatInspectLogFieldsForPrompt(inspectKeys);

  return `Ты помогаешь собрать JSON-массив widgets для вкладки отчётов Oktopus (журнал доступа).
Результат вставляется в «Редактировать дашборд» на странице /manage/reports.

${ACCESS_LOG_REPORT_WIDGETS_REFERENCE}

${inspectLogSection}

Справка по полному report spec (time, filters, API) — для понимания query:
${ACCESS_LOG_REPORT_API_REFERENCE}

Требования к ответу:
- Верни только JSON-массив widgets (от 1 до 12 элементов), без markdown и без комментариев.
- Не оборачивай в объект: без version, time, filters и без поля "widgets" — только массив [...].
- Не выходи за whitelist полей group_by / inspect_log.
- id виджетов — короткие латиницей (snake_case), уникальные в массиве.
- Учитывай id и смысл правил ACL/инспекции из секций ниже при group_by decision_rule_ref / inspect_log.

Пример ответа:
${JSON.stringify(ACCESS_LOG_REPORT_WIDGETS_EXAMPLE, null, 2)}`;
}

/** Собирает промпт с актуальными ACL (полный pattern) и inspect (полный Lua). */
export async function buildAccessLogReportAiPrompt(
  extraInspectKeys?: string[],
): Promise<string> {
  const { policies, inspect } = await loadProxyRulesForReportPrompt();
  const inspectKeys = mergeInspectLogKeysForPrompt(
    inspect.map((r) => r.script),
    extraInspectKeys,
  );
  const staticPart = buildAccessLogReportAiPromptStatic(inspectKeys);
  const aclSection = policies
    .map(({ instanceName, policy }) =>
      [
        `### Инстанс: ${instanceName}`,
        formatAclPolicySection(policy),
      ].join("\n"),
    )
    .join("\n\n");
  const inspectSection = formatInspectRulesSection(inspect);
  return `${staticPart}

---

${aclSection}

---

${inspectSection}`;
}

export function defaultSummaryReportSpec(
  from: string,
  to: string,
): AccessLogReportSpec {
  return {
    version: 1,
    time: { from, to },
    filters: {},
    widgets: [
      {
        "id": "total_requests",
        "type": "stat",
        "title": "Всего записей",
        "query": {
          "metric": "count"
        }
      },
      {
        "id": "avg_decide_duration_stat",
        "type": "stat",
        "title": "Среднее время принятия решения (мкс)",
        "query": {
          "metric": "avg_decide_duration_us"
        }
      },
      {
        "id": "hourly_traffic",
        "type": "timeseries",
        "title": "Запросы (разрешения/блокировки)",
        "query": {
          "group_by_time": "1h",
          "metric": "count",
          "split_by": "action"
        }
      },
      {
        "id": "avg_duration_timeseries",
        "type": "timeseries",
        "title": "Среднее время принятия решения (мкс)",
        "query": {
          "group_by_time": "1h",
          "metric": "avg_decide_duration_us",
          "split_by": "action"
        }
      },
      {
        "id": "top_destinations",
        "type": "bar",
        "title": "Топ назначений",
        "query": {
          "group_by": "destination_address",
          "metric": "count",
          "limit": 20,
          "order": "desc"
        }
      },
      {
        "id": "top_sources",
        "type": "bar",
        "title": "Топ источников",
        "query": {
          "group_by": "source_address",
          "metric": "count",
          "limit": 20,
          "order": "desc"
        }
      },
      {
        "id": "top_users",
        "type": "bar",
        "title": "Топ пользователей",
        "query": {
          "group_by": "user",
          "metric": "count",
          "limit": 20,
          "order": "desc"
        }
      },
      {
        "id": "action_distribution",
        "type": "bar",
        "title": "Распределение действий",
        "query": {
          "group_by": "action",
          "metric": "count",
          "limit": 2,
          "order": "desc"
        }
      },
      {
        "id": "denied_by_distribution",
        "type": "bar",
        "title": "Распределение по denied_by",
        "query": {
          "group_by": "denied_by",
          "metric": "count",
          "limit": 20,
          "order": "desc"
        }
      },
      {
        "id": "decision_rule_ref_distribution",
        "type": "bar",
        "title": "Распределение по правилам решения",
        "query": {
          "group_by": "decision_rule_ref",
          "metric": "count",
          "limit": 20,
          "order": "desc"
        }
      },
      {
        "id": "search_engine_distribution",
        "type": "bar",
        "title": "Поисковые системы",
        "query": {
          "group_by": "search_engine",
          "field_nonempty": ["search_engine"],
          "metric": "count",
          "limit": 20,
          "order": "desc"
        }
      },
      {
        "id": "user_action_table",
        "type": "table",
        "title": "Пользователь × действие",
        "query": {
          "group_by_cols": [
            "user",
            "action"
          ],
          "search_columns": [
            "user"
          ],
          "metric": "count",
          "order": "desc"
        }
      }
    ],
  };
}
