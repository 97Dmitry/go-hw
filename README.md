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
