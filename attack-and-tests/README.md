# attack-and-tests

Интеграционные проверки прокси и ACL. Нужен запущенный **`go run ./cmd serve`** (API `http://127.0.0.1:8000`, по умолчанию `admin` / `admin`).

Тестовый прокси: инстанс **`attack-and-tests`** на **`127.0.0.1:9000`** (создаётся/находится через `setup.EnsureTestInstance()` — см. [`setup/`](./setup/)).

Итог каждого прогона: **`RESULT: OK`** или **`RESULT: FAIL`**.

Общие флаги API: `-api`, `-api-user`, `-api-pass`. PoC дополнительно: `-skip-setup`, `-proxy` (по умолчанию `127.0.0.1:9000`).

## Команды запуска

Из корня репозитория (`oktopus/`).

### Все ACL-тесты подряд

```bash
go run ./attack-and-tests/run-all-acl
```

### ACL / `http_access` (по одному)

```bash
go run ./attack-and-tests/http-access-chain
go run ./attack-and-tests/http-access-negation
go run ./attack-and-tests/acl-all
go run ./attack-and-tests/acl-src
go run ./attack-and-tests/acl-dst
go run ./attack-and-tests/acl-dstdomain
go run ./attack-and-tests/acl-port
go run ./attack-and-tests/acl-url-regex
go run ./attack-and-tests/acl-ldap-group
go run ./attack-and-tests/proxy-auth-static
go run ./attack-and-tests/proxy-auth-ldap
go run ./attack-and-tests/ssl-verify
go run ./attack-and-tests/delay-access
```

### PoC обхода (`all` = lab + mitm/tunnel + сценарий)

```bash
go run ./attack-and-tests/mitm-host-mismatch all
go run ./attack-and-tests/mitm-allowed-host-evil-sni all
go run ./attack-and-tests/mitm-allowed-sni-evil-host all
go run ./attack-and-tests/tunnel-dst-resolve all
go run ./attack-and-tests/websocket-proxy all
go run ./attack-and-tests/tunnel-connect-ip-bypass all
go run ./attack-and-tests/mitm-port-defer all
go run ./attack-and-tests/tunnel-ipv6-loopback all
go run ./attack-and-tests/plain-http-host-mismatch all
go run ./attack-and-tests/http-url-regex all
go run ./attack-and-tests/mitm-http-smuggling all
go run ./attack-and-tests/ws-upgrade-host-mismatch all
go run ./attack-and-tests/connect-no-proxy-auth all
go run ./attack-and-tests/dns-rebind-resolve all
go run ./attack-and-tests/dstdomain-normalization all
```

У отдельных PoC есть подкоманды `lab` и `run` (см. `go run ./attack-and-tests/<name>` без аргументов).

## PoC обхода (lab + прокси)

| Папка | Описание |
|-------|----------|
| [mitm-host-mismatch](./mitm-host-mismatch/) | MITM: CONNECT/SNI vs HTTP Host |
| [mitm-allowed-host-evil-sni](./mitm-allowed-host-evil-sni/) | MITM: разрешённый Host, запрещённый TLS SNI |
| [mitm-allowed-sni-evil-host](./mitm-allowed-sni-evil-host/) | MITM: разрешённый TLS SNI, запрещённый HTTP Host |
| [tunnel-dst-resolve](./tunnel-dst-resolve/) | Tunnel: dstdomain + `deny dst` при DNS на loopback |
| [websocket-proxy](./websocket-proxy/) | WebSocket echo: wss/ws в mitm и tunnel |
| [tunnel-connect-ip-bypass](./tunnel-connect-ip-bypass/) | CONNECT по IP в обход `dstdomain` (`deny dst`) |
| [mitm-port-defer](./mitm-port-defer/) | MITM: CONNECT :443 vs URL на запрещённый порт |
| [tunnel-ipv6-loopback](./tunnel-ipv6-loopback/) | CONNECT `[::1]` и `acl dst ::1/128` |
| [plain-http-host-mismatch](./plain-http-host-mismatch/) | Plain HTTP: absolute URL vs `Host` |
| [http-url-regex](./http-url-regex/) | `url_regex` на реальном HTTP через прокси |
| [mitm-http-smuggling](./mitm-http-smuggling/) | MITM: CL.TE smuggling probe |
| [ws-upgrade-host-mismatch](./ws-upgrade-host-mismatch/) | WebSocket upgrade: Host internal |
| [connect-no-proxy-auth](./connect-no-proxy-auth/) | CONNECT без `Proxy-Authorization` |
| [dns-rebind-resolve](./dns-rebind-resolve/) | Публичный DNS → loopback (sslip.io) |
| [dstdomain-normalization](./dstdomain-normalization/) | Регистр `LOCALHOST` vs `localhost` |

Общая библиотека PoC: [`poclib/`](./poclib/).

## ACL / `http_access` (API evaluate)

Каждая директива — отдельный каталог с `policy.squid` и `main.go`. Публикует политику на тестовый инстанс, для каждого кейса проверяет **`/api/proxy/instances/{id}/acl/evaluate`** и **реальный запрос через прокси** (CONNECT или HTTP GET) в режимах **mitm** и **tunnel**. По умолчанию proxy-auth static (`attack-poc`); **`acl-ldap-group`** — backend **ldap** и учётки `user`/`user2` из dev OpenLDAP (группы `allow_all` / `deny_all`). `SkipProxy` — только evaluate (вымышленный `src`).

| Папка | Что проверяет |
|-------|----------------|
| [http-access-chain](./http-access-chain/) | Порядок `deny` → `allow` |
| [http-access-negation](./http-access-negation/) | `!acl` в `http_access` |
| [acl-all](./acl-all/) | `acl all` |
| [acl-src](./acl-src/) | `acl src` (CIDR, диапазон) |
| [acl-dst](./acl-dst/) | `acl dst` |
| [acl-dstdomain](./acl-dstdomain/) | `acl dstdomain` (exact, suffix, wildcard) |
| [acl-port](./acl-port/) | `acl port` |
| [acl-url-regex](./acl-url-regex/) | `acl url_regex` |
| [acl-ldap-group](./acl-ldap-group/) | `acl ldap_group` + LDAP auth (группы user / user2) |
| [proxy-auth-static](./proxy-auth-static/) | Proxy-Authorization, backend `static` (login:pass) |
| [proxy-auth-ldap](./proxy-auth-ldap/) | Proxy-Authorization, backend `ldap` (нужен OpenLDAP на `127.0.0.1:1389`) |
| [ssl-verify](./ssl-verify/) | `ssl_verify skip/require` (compile + publish) |
| [delay-access](./delay-access/) | `delay_pools` / `delay_access` (compile + publish) |

Общий код: [`setup/`](./setup/), раннер: [`aclrun/`](./aclrun/).
