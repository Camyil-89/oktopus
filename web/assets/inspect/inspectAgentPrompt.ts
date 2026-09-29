import { INSPECT_LUA_API_REFERENCE, INSPECT_LUA_EXAMPLES } from "@/assets/inspect/luaExamples";

/** Полная справка по пакету inspect для ИИ (синхронизировать с internal/proxy/inspect). */
export const INSPECT_AGENT_REFERENCE = `Proxy inspect (Lua) — Oktopus

Пакет internal/proxy/inspect выполняет Lua-правила на исходящих HTTP-запросах прокси (после успешного ACL allow). Решение «запретить запрос» задаётся действием правила в БД (ActionDenyOnMatch / ActionAllowOnMatch), а не строкой "deny" из Lua.

Связанный код при смене контракта:
- internal/db/proxyinspect — CRUD, компиляция в Runner, reload
- internal/proxy/accesslog — InspectRuleLogs, FinalHTTPEntryAfterInspect
- internal/proxy/server/server.go — ACL → inspect; inspectEnabled = connect MITM

## Когда инспекция работает

| Условие | Поведение |
|--------|-----------|
| Режим tunnel (не MITM) | HTTPMiddleware не регистрируется. HTTPS внутри туннеля не разбирается. |
| ACL не пропустил | Middleware не вызывается. |
| runner nil или Empty | allow, журнал «после ACL» без inspect. |
| Ошибка Lua (Eval error) | deny запроса. |
| Совпадение + Action 0 (deny on match) | deny, дальнейшие правила не выполняются. |
| Совпадение + Action 1 (allow on match) | allow, следующие правила выполняются; запоминается последнее совпавшее allow-правило. |
| Ни одно правило не совпало | allow (если не было ошибки). |

Порядок правил — sort_order в БД, только включённые правила в Program.

## Ограничения скрипта

- Не пустой после trim.
- Максимум 64 KiB на правило.
- Обязательна глобальная function inspect(ctx).
- Тело для метаданных (upload_filenames): до 4 MiB, тело восстанавливается для прокси.

## Возврат inspect(ctx)

Lua не выбирает deny/allow напрямую — только «совпало правило или нет». Действие из Action правила.

| Возврат | Совпадение? |
|---------|-------------|
| false, nil | Нет |
| true | Да |
| "", "allow", "skip", "false" (без учёта регистра) | Нет |
| Любая другая строка | Да |
| { action = "..." } | allow/skip/"" → нет; match/log/tag/deny/block → да; неизвестное непустое → да |
| Прочие truthy | Да |

Строки "deny"/"block" в возврате не блокируют сами по себе — только совпадение; блок даёт Action == 0.

## ctx:log

- ctx:log("key", value) — пара в scratch текущего правила.
- ctx:log({ k = v, ... }) — слияние полей.
Значения: bool, string, number, вложенные таблицы. Scratch → журнал access log (extra inspect по id правила). Ключи ctx:log для отчётов: inspect_log.<ключ>.

## Типичные сценарии

1. Жёсткий запрет — Action 0, скрипт true при условии.
2. Whitelist — сверху Action 1, снизу Action 0 «всё остальное».
3. Только аудит — ctx:log при match; для отчётов фильтр inspect_rule_id + поля inspect_log.*`;

function formatExamplesForPrompt(): string {
  return INSPECT_LUA_EXAMPLES
    .map(
      (ex) => `### ${ex.title}
${ex.description}

\`\`\`lua
${ex.script.trim()}
\`\`\``,
    )
    .join("\n\n");
}

/** Текст для вставки в чат с ИИ (Cursor, Claude и т.д.). */
export function buildInspectAgentPrompt(): string {
  return `Ты помогаешь писать и править Lua-правила инспекции исходящих HTTP-запросов для прокси Oktopus.

Отвечай готовым скриптом: одна глобальная function inspect(ctx), без лишнего текста, если пользователь не просит объяснение.
Учитывай порядок правил, Action «Запретить» (0) / «Разрешить» (1) и режим MITM.
Для журнала и отчётов используй ctx:log с осмысленными ключами (upload_method, upload_filenames, …).

${INSPECT_AGENT_REFERENCE}

${INSPECT_LUA_API_REFERENCE}

## Встроенные примеры продукта

${formatExamplesForPrompt()}
`;
}
