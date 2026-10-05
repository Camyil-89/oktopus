# CONNECT по IP в обход dstdomain

Запрещён `secret.lab` по имени; lab слушает `127.0.0.1:9777`. Обход — `CONNECT 127.0.0.1:9777` без попадания в `dstdomain`. Ожидаемо блокируется `deny loopback` / `acl dst`.

```bash
go run ./attack-and-tests/tunnel-connect-ip-bypass all
```
