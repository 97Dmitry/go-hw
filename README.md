# Go HW 2

Gateway предоставляет HTTP API, Ledger хранит помесячные бюджеты и расходы в PostgreSQL и кеширует отчёты в Redis с TTL 30 секунд. Gateway использует Ledger как библиотеку в своём процессе; отдельного сетевого сервиса Ledger нет.

Требуются Go 1.27.1+, Docker Compose (либо локальные PostgreSQL и Redis), Goose. `gateway` и `ledger` — отдельные Go-модули, объединённые `go.work`.

- [Ledger: создание БД, установка Goose, миграции up/down, env, бизнес-правила и кеш](ledger/README.md).
- [Gateway: запуск и cURL, включая повторный запрос отчёта](gateway/README.md).

## Запуск в Docker Compose

Команды выполняются из корня репозитория. Нужны Docker с BuildKit, Docker Compose 2.17+ и [Goose](ledger/README.md#postgresql-redis-и-миграции).

1. Запустите PostgreSQL и дождитесь статуса `healthy`.

   ```bash
   docker compose up -d --wait db
   docker compose ps db
   ```

2. Примените миграции с хоста через опубликованный порт PostgreSQL.

   ```bash
   export DATABASE_URL='postgres://postgres:postgres@localhost:55432/cashapp?sslmode=disable'
   goose -dir ./ledger/migrations postgres "$DATABASE_URL" up
   ```

3. Соберите и запустите Gateway. Compose дождётся готовности `db` и `redis`.

   ```bash
   docker compose up -d --build --wait gateway
   docker compose ps
   ```

   Gateway обращается к `db:5432` и `redis:6379` внутри сети Compose. Redis нужен Ledger для кеша отчётов и проверяется при старте приложения. Оба Go-модуля компилируются в бинарь Gateway. Для этого сборка читает `./gateway` и дополнительный контекст `./ledger`, который соответствует `replace => ../ledger` в `gateway/go.mod`.

4. Проверьте HTTP из контейнера и с хоста. Обе команды должны вернуть `pong`.

   ```bash
   docker compose exec -T gateway wget -qO- http://127.0.0.1:8080/ping
   curl -fsS http://localhost:8080/ping
   ```

Для PostgreSQL 18 используется отдельный том `cashapp-pg18-data`. Если вы запускали прежний стенд с PostgreSQL 17, его данные остаются в старом томе `cashapp-data`. Для переноса данных используйте экспорт из PostgreSQL 17 и импорт в PostgreSQL 18.

### Проверка HTTP API

Пример для bash/zsh на Linux/WSL. Для отчёта используйте интервал текущего месяца UTC.

```bash
base=http://localhost:8080
category="docker-check-$(date +%s)"
date_utc=$(date -u +%Y-%m-%dT%H:%M:%SZ)
from=$(date -u +%Y-%m-01)
to=$(date -u -d "$from +1 month" +%Y-%m-%d)

# 201. Создать бюджет на 100
curl -fsS "$base/api/budgets" -H 'Content-Type: application/json' \
  -d "{\"category\":\"$category\",\"limit\":100}"

# 201. Добавить расход на 30
curl -fsS "$base/api/transactions" -H 'Content-Type: application/json' \
  -d "{\"category\":\"$category\",\"amount\":30,\"description\":\"single\",\"date\":\"$date_utc\"}"

# 200. Импортировать расход на 20, ожидается accepted=1
curl -fsS "$base/api/transactions/bulk" -H 'Content-Type: application/json' \
  -d "[{\"category\":\"$category\",\"amount\":20,\"description\":\"bulk\",\"date\":\"$date_utc\"}]"

# 200. Убедиться, что в списках есть новый бюджет и два расхода
curl -fsS "$base/api/budgets"
curl -fsS "$base/api/transactions"

# 200. Для новой категории ожидается total=50
curl -fsS "$base/api/reports/summary?from=$from&to=$to"
```

Если этот интервал запрашивали в последние 30 секунд, дождитесь истечения TTL кеша отчёта и повторите GET.

### Сборка и запуск через Docker CLI

Соберите оба образа из корня репозитория. Образ Ledger предназначен для отдельной проверки сборки.

```bash
docker build --build-context ledger=./ledger -t cashapp-gateway ./gateway
docker build -t cashapp-ledger ./ledger
```

С запущенными `db` и `redis` и применёнными миграциями освободите порт 8080 командой `docker compose stop gateway`. Затем запустите Gateway в сети Compose.

```bash
docker run --rm --network go-hw-2_default -p 8080:8080 \
  -e DATABASE_URL='postgres://postgres:postgres@db:5432/cashapp?sslmode=disable' \
  -e REDIS_ADDR=redis:6379 cashapp-gateway
```

В другом терминале выполните `curl -fsS http://localhost:8080/ping`. Для остановки контейнера нажмите Ctrl+C в терминале с `docker run`. Команда `docker compose down` останавливает стенд и сохраняет том PostgreSQL.

## HTTP API

| Метод | Путь | Назначение |
| --- | --- | --- |
| GET | `/ping` | Проверка доступности HTTP, ответ `pong` |
| POST | `/api/budgets` | Создать/обновить бюджет категории на текущий месяц UTC |
| GET | `/api/budgets` | Список бюджетов из PostgreSQL |
| POST | `/api/transactions` | Добавить расход в пределах месячного бюджета |
| POST | `/api/transactions/bulk?workers=4` | Импортировать массив расходов с отчётом по отклонённым строкам |
| GET | `/api/transactions` | Список расходов из PostgreSQL |
| GET | `/api/reports/summary?from=YYYY-MM-DD&to=YYYY-MM-DD` | Сумма расходов по категориям за `[from,to)`, кеш Redis |

## Пакетный импорт расходов

`POST /api/transactions/bulk` принимает JSON-массив объектов с полями `amount`, `category`, `description` и `date` в том же формате, что и одиночный `POST /api/transactions`. Поле `date` передаётся в RFC 3339. Необязательный query-параметр `workers` задаёт число воркеров от **1 до 32**, по умолчанию **4**. Некорректный параметр, синтаксис JSON или формат полей, включая строку `date` в другом формате, дают `400` до начала импорта.

После обработки пакета Gateway возвращает `200 OK`, даже если часть или все расходы отклонены. `accepted` и `rejected` содержат количество сохранённых и отклонённых расходов. Индексы в `errors` начинаются с нуля и отсортированы по позиции в исходном массиве. Для пустого массива ответ содержит нулевые счётчики и `"errors":[]`.

Ledger проводит каждый расход отдельной транзакцией с проверкой месячного бюджета. При ограниченном остатке порядок сохранения расходов с несколькими воркерами зависит от выполнения пула; `workers=1` обрабатывает расходы по порядку входного массива. При отмене или таймауте Ledger возвращает частичную сводку и ошибку контекста, а Gateway отвечает `504` с `{"error":"request timeout"}`. Успешно сохранённые расходы остаются в базе, поэтому повтор всего пакета может создать дубликаты.

```bash
category="bulk-food-$(date +%s)"
date_utc=$(date -u +%Y-%m-%dT%H:%M:%SZ)

curl -fsS http://localhost:8081/api/budgets \
  -H 'Content-Type: application/json' \
  -d "{\"category\":\"$category\",\"limit\":100}"

curl -i 'http://localhost:8081/api/transactions/bulk?workers=4' \
  -H 'Content-Type: application/json' \
  --data-binary @- <<EOF
[
  {"amount":60,"category":"$category","description":"lunch","date":"$date_utc"},
  {"amount":120,"category":"$category","description":"over limit","date":"$date_utc"}
]
EOF

# 200 OK: {"accepted":1,"rejected":1,"errors":[{"index":1,"error":"budget exceeded"}]}
```

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
