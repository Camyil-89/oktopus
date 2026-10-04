# attack-and-tests

Интеграционные проверки прокси и ACL. Нужен запущенный **`go run ./cmd serve`** (API `http://127.0.0.1:8000`, по умолчанию `admin` / `admin`).

Итог каждого прогона: **`RESULT: OK`** или **`RESULT: FAIL`**.

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

```bash
go run ./attack-and-tests/mitm-host-mismatch all
# … остальные PoC — см. таблицу выше; каждый: go run ./attack-and-tests/<name> all
```

## ACL / `http_access` (API evaluate)

Каждая директива — отдельный каталог с `policy.squid` и `main.go`. Публикует политику, для каждого кейса проверяет **`/api/proxy/acl/evaluate`** и **реальный запрос через прокси** (CONNECT или HTTP GET) в режимах **mitm** и **tunnel**. По умолчанию proxy-auth static (`attack-poc`); **`acl-ldap-group`** — backend **ldap** и учётки `user`/`user2` из dev OpenLDAP (группы `allow_all` / `deny_all`). `SkipProxy` — только evaluate (вымышленный `src`).

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

Один тест:

```bash
go run ./attack-and-tests/acl-dstdomain
```

Все ACL-тесты **по очереди** (один `go run`, каждый сьют со своим publish/setup):

```bash
go run ./attack-and-tests/run-all-acl
```

Флаги: `-api`, `-api-user`, `-api-pass` (как у PoC).

Общий код: [`setup/`](./setup/), раннер: [`aclrun/`](./aclrun/).
