# WebSocket через прокси

Проверка echo WebSocket:

| Сценарий | Путь в oktopus |
|----------|----------------|
| **wss** | `CONNECT` → TLS (MITM или tunnel) → upgrade |
| **ws** | explicit HTTP proxy (`GET http://host/…` + Upgrade, `internal/proxy/http`) |

Нужен запущенный `go run ./cmd serve`.

```bash
go run ./attack-and-tests/websocket-proxy all
```

Lab поднимает echo-серверы на `127.0.0.1:9443` (TLS) и `127.0.0.1:9080` (HTTP). PoC публикует `acl.example.squid`, включает proxy-auth и прогоняет **mitm** и **tunnel** (`connect_mode`).

Для tunnel wss TLS к origin без проверки CA (самоподписанный lab); для mitm — CA прокси с API.

## Интерпретация

| Итог | Смысл |
|------|--------|
| `RESULT: OK` | wss и ws echo прошли в **mitm** и **tunnel** |
| `RESULT: FAIL` | хотя бы один режим или сценарий не прошёл |

Флаги: `-api`, `-api-user`, `-api-pass`, `-skip-setup`, `-proxy`, `-ca`, `-wss-connect`, `-ws-url`, `-connect-mode` (только `run`).
