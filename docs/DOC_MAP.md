# Карта документации (docs ↔ код)

Языки: **`docs/ru/`** и **`docs/en/`** — отдельный HTML на каждый раздел.  
Источник текста: **`docs/content/<locale>/<id>.html`**. Оболочка (меню, шапка) собирается **`node docs/build.mjs`**.

Машиночитаемая карта: **`docs/manifest.json`**.

## Страницы (как в `web/app`)

| `id` | Файл docs (ru/en) | URL кабинета |
|------|-------------------|--------------|
| `intro` | `index.html` | — |
| `quickstart` | `quickstart.html` | — |
| `implementation-examples` | `implementation-examples.html` | — (схемы размещения прокси) |
| `login` | `login.html` | `/login` |
| `manage-home` | `manage/index.html` | `/manage` |
| `manage-rules` | `manage/rules.html` | `/manage/rules` |
| `rules-acl-syntax` | `manage/rules-acl-syntax.html` | — (справочник ACL; промпт для ИИ: `content/prompts/<locale>/rules-acl-syntax.txt`) |
| `manage-access-log` | `manage/access-log.html` | `/manage/access-log` |
| `manage-access-log-reports` | `manage/access-log/reports.html` | `/manage/access-log/reports` |
| `manage-proxy` | `manage/proxy.html` | `/manage/proxy` |
| `manage-users` | `manage/users.html` | `/manage/users` |

## Правка

1. Изменить **`docs/content/ru/<id>.html`** и **`docs/content/en/<id>.html`**.
2. Выполнить **`node docs/build.mjs`** (пересоберёт HTML в `ru/` и `en/`).
3. Новая страница кабинета — добавить `id` в `docs/build.mjs` (`PAGES`), `manifest.json`, content-файлы, пересобрать.

Интерактивные макеты (hover + `ui-tour.js`): **`login`**, все **`manage-*`**, **`rules-acl-syntax`** — в `content/<locale>/<id>.html`, плейсхолдер **`__MANAGE_ASIDE__`** подставляется в `build.mjs`.
