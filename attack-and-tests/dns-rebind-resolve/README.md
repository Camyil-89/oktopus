# DNS: публичное имя → loopback

`CONNECT` на `9667.127.0.0.1.sslip.io:9667` (внешний DNS отдаёт 127.0.0.1). Ожидаем `deny dst` после enrich.

Нужен доступ к публичному DNS (CI/интернет). Юнит-тест flip: `internal/proxy/acl/test/dns_rebind_flip_test.go`.

```bash
go run ./attack-and-tests/dns-rebind-resolve all
```
