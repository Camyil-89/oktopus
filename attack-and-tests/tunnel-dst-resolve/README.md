# Tunnel: dstdomain allow + dst deny при резолве в loopback

## Одна команда

Запустите `go run ./cmd serve`. PoC настраивает прокси через API и проверяет **mitm** и **tunnel**.

```bash
go run ./attack-and-tests/tunnel-dst-resolve all
```

Атака рассчитана на **tunnel**; в **mitm** ожидаем `RESULT: OK` на bypass.

## Интерпретация

| Итог | Смысл |
|------|--------|
| `RESULT: OK` | TCP до lab через bypass не прошёл |
| `RESULT: FAIL` | Маркер `TUNNEL_INTERNAL_HIT` получен (уязвимость) |
