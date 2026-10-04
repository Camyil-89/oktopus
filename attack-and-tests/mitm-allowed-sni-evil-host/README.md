# MITM: разрешённый TLS SNI, запрещённый HTTP Host

Сценарий: **CONNECT** на разрешённый `localhost:9443`, **ClientHello SNI** = `localhost`, внутри туннеля **GET** с **Host** на запрещённый internal (`127.0.0.1:9555`). Уязвимость — если ACL смотрит только на SNI CONNECT/MITM, а не на фактический Host запроса.

```bash
go run ./attack-and-tests/mitm-allowed-sni-evil-host all
```

| Итог | Смысл |
|------|--------|
| `RESULT: OK` | Обход не сработал |
| `RESULT: FAIL` | Доступ к internal при разрешённом SNI |
