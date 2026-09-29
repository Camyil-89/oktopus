# MITM: расхождение CONNECT/SNI и HTTP Host

## Одна команда

Запустите `go run ./cmd serve`.

PoC сам логинится в API, включает proxy-auth (`attack-poc` / `attack-poc-secret`), публикует `acl.example.squid` и гоняет сценарий в **mitm** и **tunnel**. Разрешённый origin — `localhost:9443` (без правки `hosts`).

```bash
go run ./attack-and-tests/mitm-host-mismatch all
```

Флаги API: `-api`, `-api-user`, `-api-pass`. Ручная настройка: `-skip-setup -proxy … -ca …`.

## Интерпретация

| Итог | Смысл |
|------|--------|
| `RESULT: OK` | Обход не сработал (ожидаемо после фикса) |
| `RESULT: FAIL` | Обход подтверждён хотя бы в одном режиме |

В **tunnel** обход host mismatch обычно не применим — там тоже ожидаем OK.
