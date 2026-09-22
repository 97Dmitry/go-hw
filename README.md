# Go HW 2

## Требования

- Go 1.27+
- Make

## Запуск

### Запуск gateway:

```bash
make start-gateway
```

Сервис будет доступен по адресу `http://localhost:8081`. Проверка:

```bash
curl http://localhost:8081/ping
```

### Запуск ledger:

```bash
make start-ledger
```
