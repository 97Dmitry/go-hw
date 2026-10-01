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

## Структура

```
gateway/                     # HTTP-шлюз (отдельный Go-модуль)
├── cmd/gateway/             # main: запуск HTTP-сервера
└── internal/handler/        # роутинг и HTTP-хендлеры
ledger/                      # сервис учёта транзакций (отдельный Go-модуль)
├── cmd/ledger/              # main: сборка зависимостей
└── internal/
    ├── domain/              # сущности и бизнес-правила, без зависимостей
    ├── service/             # бизнес-логика; объявляет интерфейс хранилища
    └── repository/memory/   # in-memory реализация хранилища
```
