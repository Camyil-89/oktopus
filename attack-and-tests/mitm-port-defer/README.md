# MITM: defer PORT на CONNECT

`CONNECT localhost:9443`, первый HTTPS-запрос на порт `9778` при `deny port 9778`.

```bash
go run ./attack-and-tests/mitm-port-defer all
```
