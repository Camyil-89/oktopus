# MITM: разрешённый Host, запрещённый TLS SNI

Сценарий: **CONNECT** и заголовок **Host** — разрешённый `localhost`, в **ClientHello SNI** — запрещённый `internal.blocked` (см. `acl evil_sni`). Для SNI не используйте IP: leaf MITM с CA клиента сверяется по DNS SAN, иначе TLS обрывается до проверки ACL.

Уязвимость — если ACL или исходящее соединение опираются на SNI, а HTTP-проверка — только на Host.

```bash
go run ./attack-and-tests/mitm-allowed-host-evil-sni all
```

| Итог | Смысл |
|------|--------|
| `RESULT: OK` | Обход не сработал |
| `RESULT: FAIL` | Доступ к internal origin при разрешённом Host |
