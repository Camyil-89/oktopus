# Карта документации (docs ↔ код)

Языки: **`docs/ru/index.html`** и **`docs/en/index.html`** — одна структура, разный текст. Общие стили и скрипт: `docs/assets/`.

Машиночитаемая карта: **`docs/manifest.json`**.

## Разделы (одинаковые `data-doc` в ru и en)

| `data-doc` | URL кабинета | Где править текст |
|------------|--------------|-------------------|
| `intro` | — | `<article id="doc-intro">` в ru и en |
| `quickstart` | — | `<article id="doc-quickstart">` |
| `login` | `/login` | `<article id="doc-login">` |
| `manage-home` | `/manage` | `<article id="doc-manage-home">` |
| `manage-rules` | `/manage/rules` | `<article id="doc-manage-rules">` |
| `manage-access-log` | `/manage/access-log` | `<article id="doc-manage-access-log">` |
| `manage-access-log-reports` | `/manage/access-log/reports` | `<article id="doc-manage-access-log-reports">` |
| `manage-proxy` | `/manage/proxy` | `<article id="doc-manage-proxy">` |
| `manage-users` | `/manage/users` | `<article id="doc-manage-users">` |

Порядок в боковом меню = `web/assets/components/manageNav.ts`.

## Правило для правок

Любое изменение смысла в коде → обновить **оба** файла `docs/ru/index.html` и `docs/en/index.html` в соответствующем `<article id="doc-…">`. Навигацию (`data-doc`, порядок пунктов) менять синхронно в обоих языках.
