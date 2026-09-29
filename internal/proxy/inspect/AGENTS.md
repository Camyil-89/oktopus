# Proxy inspect — заметки для разработчиков

Пакет `oktopus/internal/proxy/inspect` — Lua на **исходящих HTTP** после ACL allow.

**Справочник для пользователей и ИИ** (семантика, Lua API, примеры): `web/assets/inspect/inspectAgentPrompt.ts` — в UI кнопка «Инструкция для ИИ» на странице правил (`ProxyInspectRulesCard`). Краткая справка в коллапсе — `INSPECT_LUA_API_REFERENCE` в `web/assets/inspect/luaExamples.ts`.

## Чеклист при изменении пакета

1. **Тесты** в `internal/proxy/inspect/test/` (`package inspect_test`).
2. При смене Lua-контракта для пользователя — `web/assets/inspect/luaExamples.ts` и **`web/assets/inspect/inspectAgentPrompt.ts`**.
3. Поля журнала — `internal/proxy/accesslog`, `internal/db/proxyaccesslog`.
4. Модель правил — `internal/db/proxyinspect`, API `internal/api/proxy/view/inspect_handler.go`.

Правило Cursor: `.cursor/rules/proxy-inspect-agents.mdc`.
