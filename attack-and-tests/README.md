# attack-and-tests

Интеграционные проверки прокси и ACL. Нужен запущенный **`go run ./cmd serve`** (API `http://127.0.0.1:8000`, по умолчанию `admin` / `admin`).

Итог каждого прогона: **`RESULT: OK`** или **`RESULT: FAIL`**.

## PoC обхода (lab + прокси)

| Папка | Описание |
|-------|----------|
| [mitm-host-mismatch](./mitm-host-mismatch/) | MITM: CONNECT/SNI vs HTTP Host |
| [tunnel-dst-resolve](./tunnel-dst-resolve/) | Tunnel: dstdomain + `deny dst` при DNS на loopback |

```bash
go run ./attack-and-tests/mitm-host-mismatch all
go run ./attack-and-tests/tunnel-dst-resolve all
```

## ACL / `http_access` (API evaluate)

Каждая директива — отдельный каталог с `policy.squid` и `main.go`. Публикует политику, для каждого кейса проверяет **`/api/proxy/acl/evaluate`** и **реальный запрос через прокси** (CONNECT или HTTP GET) в режимах **mitm** и **tunnel**. По умолчанию proxy-auth static (`attack-poc`); **`acl-ldap-group`** — backend **ldap** и учётки `user`/`user2` из dev OpenLDAP (группы `allow_all` / `deny_all`). `SkipProxy` — только evaluate (вымышленный `src`, `url_regex` — plain HTTP через прокси пока не совпадает с evaluate).

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
