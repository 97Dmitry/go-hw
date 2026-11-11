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

## Таймауты запросов

Все HTTP-ручки обёрнуты middleware с таймаутом **2 секунды**. Настройка при запуске: `HTTP_REQUEST_TIMEOUT=2s` (также допустимы `500ms`, `5s`); неположительное или некорректное значение останавливает запуск.

```bash
HTTP_REQUEST_TIMEOUT=2s make start-gateway
curl -i 'http://localhost:8081/api/reports/summary?from=2026-11-01&to=2026-12-01'
# 200: [{"category":"food","total":60}]
```

Период отчёта — UTC `[from,to)`. Некорректные даты дают `400`. При превышении таймаута или отмене контекста API возвращает **504 Gateway Timeout**, `Content-Type: application/json; charset=utf-8`, тело `{"error":"request timeout"}`. При разрыве соединения клиент уже может не получить ответ.

Контекст запроса передаётся во все методы Ledger и SQL-запросы, включая ожидание подключения из пула. Middleware буферизует ответ, поэтому частичный успешный ответ не отправляется перед 504. Отмена кооперативная: код Ledger проверяет контекст и завершает рабочие горутины; Go не может принудительно остановить произвольную горутину.

Для ручной проверки перезапустите Gateway с `HTTP_REQUEST_TIMEOUT=1ms` и запросите отчёт за интервал, который ещё не находится в кеше. На быстром стенде даже 1 мс иногда достаточно для успешного ответа; автоматические тесты используют управляемое ожидание контекста, без искусственных задержек в рабочем коде. В логах расчёта: `report calculation in progress` каждые 400 мс, при отмене — `report calculation canceled`, при завершении — `report heartbeat stopped`. Кешированный ответ не запускает расчёт и heartbeat. После проверки верните `HTTP_REQUEST_TIMEOUT=2s` или удалите переменную.
