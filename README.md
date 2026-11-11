# Go HW 2

Gateway предоставляет HTTP API, Ledger хранит помесячные бюджеты и расходы в PostgreSQL и кеширует отчёты в Redis с TTL 30 секунд. Gateway использует Ledger как библиотеку в своём процессе; отдельного сетевого сервиса Ledger нет.

Требуются Go 1.27.1+, Docker Compose (либо локальные PostgreSQL и Redis), Goose. `gateway` и `ledger` — отдельные Go-модули, объединённые `go.work`.

- [Ledger: создание БД, установка Goose, миграции up/down, env, бизнес-правила и кеш](ledger/README.md).
- [Gateway: запуск и cURL, включая повторный запрос отчёта](gateway/README.md).

Порядок: PostgreSQL/Redis → `goose up` → Ledger → Gateway → HTTP-проверка. Подробные команды находятся в README сервисов.

| Метод | Путь | Назначение |
| --- | --- | --- |
| POST | `/api/budgets` | Создать/обновить бюджет категории на текущий месяц UTC |
| GET | `/api/budgets` | Список бюджетов из PostgreSQL |
| POST | `/api/transactions` | Добавить расход в пределах месячного бюджета |
| GET | `/api/transactions` | Список расходов из PostgreSQL |
| GET | `/api/reports/summary?from=YYYY-MM-DD&to=YYYY-MM-DD` | Сумма расходов по категориям за `[from,to)`, кеш Redis |

## Проверка

Из корня репозитория:

```bash
go test ./ledger/... ./gateway/...
go test -race ./ledger/... ./gateway/...
go vet ./ledger/... ./gateway/...
# После миграций, с DATABASE_URL и REDIS_* для изолированного тестового стенда:
LEDGER_INTEGRATION=1 go test -race ./ledger/... ./gateway/...
```

Интеграционный тест пишет через код Ledger и оставляет уникальные тестовые записи. Обычные тесты инфраструктуру не требуют. `go test ./...` из корня не подходит: корень не является Go-модулем.

## Слои Ledger

- `ledger/internal/domain` — сущности, валидация и контракты репозиториев, транзакционной операции и кеша.
- `ledger/internal/repository/{postgres,memory}` — хранение и суммирование; PostgreSQL выполняет SQL и управляет транзакциями, без бизнес-правил и HTTP.
- `ledger/internal/service` — интерфейс приложения и единая бизнес-логика для обоих хранилищ: валидация, проверка лимита в целых копейках, отчёты.
- `ledger/factory.go` — сборка зависимостей: конфигурация из env, подключения PostgreSQL/Redis, репозитории, сервис. Миграции выполняются отдельно.
- `gateway/internal/handler` — HTTP и DTO; зависит от публичного `ledger.Service`, без доступа к деталям хранения.

Публичный пакет `ledger` экспортирует интерфейс и типы через aliases: пакеты `internal` недоступны Gateway по правилам Go. В `gateway/cmd/gateway/main.go` вызывается `svc, closeFn, err := ledger.Open(ctx)`. После успешного подключения выполняется `defer closeFn()`, сервис передаётся в `handler.NewRouter(svc)`. Ошибка подключения возвращается в `main`, логируется и завершает процесс. Контекст фабрики передаётся в стартовый Ping.

Проверка бюджета, подсчёт расходов и INSERT выполняются сервисом внутри доменной транзакционной операции. PostgreSQL удерживает `FOR UPDATE` на месячном бюджете до commit; in-memory реализация сериализует изменения общей блокировкой. Это сохраняет лимит при параллельных запросах и нескольких процессах Gateway.

## Таймауты запросов

Gateway задаёт общий дедлайн **2 секунды** для всех ручек: `HTTP_REQUEST_TIMEOUT=2s make start-gateway`. При отмене/таймауте — `504` и `{"error":"request timeout"}`. Контекст передаётся от HTTP до SQL. Отчёт при промахе кеша рассчитывается параллельно по категориям с heartbeat; подробности и примеры — в [Gateway README](gateway/README.md#таймауты-запросов) и [Ledger README](ledger/README.md#конкурентный-отчёт-и-отмена).
