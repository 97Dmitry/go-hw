# Gateway

Существующие HTTP-эндпойнты и JSON-контракты сохранены. Gateway встраивает Ledger, который теперь хранит данные в PostgreSQL; отчёты кешируются в Redis на 30 секунд. Добавлен `GET /api/reports/summary`.

## Запуск

Из корня проекта создайте PostgreSQL и Redis и примените миграции по [инструкции Ledger](../ledger/README.md). Краткий порядок:

```bash
docker compose up -d --wait
export DATABASE_URL='postgres://postgres:postgres@localhost:55432/cashapp?sslmode=disable'
export REDIS_ADDR='localhost:56379'
export REDIS_DB=0
export REDIS_PASSWORD=''
# Установка Goose описана в ledger/README.md
goose -dir ./ledger/migrations postgres "$DATABASE_URL" up
make start-ledger
# В другом терминале с теми же env:
make start-gateway
```

Gateway слушает `localhost:8081` (сервер привязан к `:8081`). Ошибки подключения PostgreSQL/Redis останавливают запуск. `DATABASE_URL` имеет приоритет над `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASS`, `DB_NAME`. Значения по умолчанию и команды отката `goose … down` описаны в Ledger README.

## HTTP-проверка

Команды для bash/zsh на Linux/WSL. Для повторного независимого прогона выберите новую категорию.

```bash
category="food-$(date +%s)"
date_utc=$(date -u +%Y-%m-%dT%H:%M:%SZ)
from=$(date -u +%Y-%m-01)
to=$(date -u -d "$from +1 month" +%Y-%m-%d)

# 201: создание бюджета текущего месяца UTC
curl -i http://localhost:8081/api/budgets \
  -H 'Content-Type: application/json' \
  -d "{\"category\":\"$category\",\"limit\":100}"

# 201: расход в пределах лимита
curl -i http://localhost:8081/api/transactions \
  -H 'Content-Type: application/json' \
  -d "{\"amount\":60,\"category\":\"$category\",\"description\":\"lunch\",\"date\":\"$date_utc\"}"

# 409: 60 + 50 > 100; запись не добавляется
curl -i http://localhost:8081/api/transactions \
  -H 'Content-Type: application/json' \
  -d "{\"amount\":50,\"category\":\"$category\",\"description\":\"over limit\",\"date\":\"$date_utc\"}"

curl -i http://localhost:8081/api/budgets
curl -i http://localhost:8081/api/transactions

# Первый запрос: cache miss, следующий в пределах TTL: cache hit.
# Смотреть строки `report cache miss` / `report cache hit` в логах Gateway.
curl -i "http://localhost:8081/api/reports/summary?from=$from&to=$to"
curl -i "http://localhost:8081/api/reports/summary?from=$from&to=$to"
```

Для отчёта обе даты обязательны, формат `YYYY-MM-DD`, `from < to`, верхняя граница исключена. Ответ — `[{"category":"food-…","total":60}]`. Если этот интервал уже запрашивали, кеш может содержать предыдущие значения до истечения 30 секунд.

Перезапустите Ledger и Gateway и повторите GET списков: данные должны сохраниться. При необходимости проверьте таблицы read-only запросами:

```bash
psql "$DATABASE_URL" -c 'SELECT id, category, month, limit_amount FROM budgets ORDER BY id;'
psql "$DATABASE_URL" -c 'SELECT id, amount, category, description, date FROM expenses ORDER BY id;'
```

Ошибки JSON/валидации дают 400, отсутствие бюджета и превышение лимита — 409, ошибки БД — 500 с `{"error":"internal server error"}`. Пустые списки — `[]`. `date` в запросе остаётся RFC 3339, в PostgreSQL хранится день UTC без времени суток.
