# CONNECT без Proxy-Authorization

При включённом static proxy-auth CONNECT без учётки должен вернуть **407**.

```bash
go run ./attack-and-tests/connect-no-proxy-auth all
```
