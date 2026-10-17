# Go HW 2

## Требования

Go 1.27.1+ и Make. `gateway` и `ledger` — отдельные Go-модули. Корневой `go.work` объединяет их при разработке; `gateway/go.mod` также содержит `require` и локальный `replace` для запуска без workspace.

## Запуск HTTP API

```bash
make start-gateway
```

Сервер слушает `http://localhost:8081`. Gateway создаёт ledger-сервис в своём процессе; данные хранятся в памяти и исчезают после перезапуска. `make start-ledger` запускает отдельный учебный консольный пример со своим хранилищем, не общий сервер данных.

## Эндпоинты

| Метод | Путь | Назначение |
| --- | --- | --- |
| `POST` | `/api/budgets` | Создать или обновить бюджет категории на текущий месяц UTC |
| `GET` | `/api/budgets` | Список бюджетов |
| `POST` | `/api/transactions` | Создать транзакцию в пределах бюджета |
| `GET` | `/api/transactions` | Список транзакций |

Для создания бюджета нужны `category` и `limit`. Повторный POST для той же категории и месяца обновляет бюджет и сохраняет ID. Для транзакции нужны `amount`, `category`, `description`, `date`; `date` задаётся в RFC 3339. При отсутствии бюджета или превышении лимита сервер отвечает `409`, при ошибке JSON/валидации — `400`. Ошибки имеют вид `{"error":"..."}`. Пустые списки возвращаются как `[]`. Все ответы используют `Content-Type: application/json; charset=utf-8`.

```bash
curl -i -X POST http://localhost:8081/api/budgets \
  -H 'Content-Type: application/json' \
  -d '{"category":"food","limit":1000}'

curl -i -X POST http://localhost:8081/api/transactions \
  -H 'Content-Type: application/json' \
  -d "{\"amount\":150,\"category\":\"food\",\"description\":\"lunch\",\"date\":\"$(date -u +%Y-%m-%dT%H:%M:%SZ)\"}"

curl -i http://localhost:8081/api/budgets
curl -i http://localhost:8081/api/transactions
```

## Проверка

```bash
go -C gateway test ./...
go -C ledger test ./...
```
