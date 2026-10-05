# dstdomain: регистр (LOCALHOST vs localhost)

Политика запрещает `LOCALHOST`, разрешает `localhost`. Обход — CONNECT в другом регистре, если нет канонизации.

```bash
go run ./attack-and-tests/dstdomain-normalization all
```
