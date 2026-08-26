# delivery-api

HTTP-сервис заказов доставки: создание, просмотр, изменение и удаление заказов с позициями.

## Стек

- Go 1.25
- chi v5 — HTTP-роутер и URL-параметры
- PostgreSQL 16 + `database/sql` + pgx (stdlib-драйвер)
- goose — SQL-миграции
- Docker Compose — локальная БД
- godotenv — загрузка `.env`
- golangci-lint — линтер (`make lint`)

## Запуск локально

Логин и пароль Postgres заданы в `docker-compose.yml`. Файл `.env` нужен приложению и goose: из него Makefile подставляет `DATABASE_URL`. URL должен совпадать с пользователем, паролем и именем БД в Compose и с портом хоста `5433`.

1. Клонировать репозиторий:

```bash
git clone <url-репозитория>
cd delivery-api
```

2. Установить goose (бинарник попадёт в `$(go env GOPATH)/bin`; каталог должен быть в `PATH`):

```bash
go install github.com/pressly/goose/v3/cmd/goose@latest
export PATH="$(go env GOPATH)/bin:$PATH"
```

3. Скопировать переменные окружения:

```bash
cp .env.example .env
```

4. Поднять PostgreSQL:

```bash
make db-up
# или: docker compose up -d
```

5. Применить миграции (`DATABASE_URL` берётся из `.env` через `Makefile`):

```bash
make migrate-up
```

6. Запустить API:

```bash
make run
# или: go run ./cmd/server/
```

Сервер слушает `:8080`. Проверка живости:

```bash
curl -i http://localhost:8080/health/liveness
```

Первый запрос — создание заказа (см. пример ниже). Проверка готовности (пинг БД):

```bash
curl -i http://localhost:8080/health/readiness
```

Остановка БД: `make db-down`. Откат последней миграции: `make migrate-down`.

## API

Маршруты `/orders*` идут через middleware `Auth` (пока заглушка: запросы не отклоняются).

| Метод | Путь | Описание |
|-------|------|----------|
| GET | `/health/liveness` | Процесс жив |
| GET | `/health/readiness` | Готовность: пинг PostgreSQL |
| GET | `/orders` | Список заказов (без позиций) |
| GET | `/orders/{id}` | Заказ по id вместе с позициями |
| POST | `/orders` | Создать заказ и позиции |
| PUT | `/orders/{id}` | Обновить адрес и цену заказа |
| DELETE | `/orders/{id}` | Удалить заказ (`204`, позиции — каскадом) |

Ошибки: `{ "error": "..." }`. Валидация: `{ "errors": ["...", "..."] }` и HTTP `400`. Не найден заказ — `404`. Невалидный id или JSON — `400`. Если обработка уперлась в свой таймаут запроса — `503`; если клиент закрыл соединение раньше, ответ не пишется.

Пример создания заказа:

```bash
curl -sS -X POST http://localhost:8080/orders \
  -H 'Content-Type: application/json' \
  -d '{
    "address": "г. Ижевск, ул. Пушкинская, 10",
    "price": 50000,
    "items": [
      {"name": "Пицца", "quantity": 2, "price": 30000}
    ]
  }'
```

Ответ `201`:

```json
{
  "id": 1,
  "address": "г. Ижевск, ул. Пушкинская, 10",
  "price": 50000,
  "status": "new",
  "created_at": "2026-08-26T16:00:00Z",
  "items": [
    {
      "id": 1,
      "order_id": 1,
      "name": "Пицца",
      "quantity": 2,
      "price": 30000
    }
  ]
}
```

`price` — целое число в копейках. У заказа обязательны адрес, неотрицательная цена и хотя бы одна позиция; у позиции — имя, количество > 0, цена ≥ 0. Статус и `created_at` задаёт БД (`new`, `now()`).

## Архитектура

```
cmd/server/          точка входа, пул БД, graceful shutdown
internal/
  config/            Load из env: DSN, порт, таймауты
  handler/           HTTP: роутер, JSON, маппинг ошибок, health
  service/           валидация и сценарии заказа
  repository/        SQL, транзакции
  model/             Order, OrderItem
  apperror/          ErrOrderNotFound, ValidationError
  middleware/        логирование, recover, заглушка Auth
migrations/          goose Up/Down
```

Запрос идёт handler → service → repository; доменная ошибка из нижних слоёв превращается в HTTP-статус в handler, без SQL и бизнес-правил в хендлере.

## Технические решения

### Слои handler / service / repository

Сервис проверяет заказ и не ходит в БД при невалидных данных; репозиторий только читает и пишет. Интерфейс объявляет пакет-потребитель (`OrderService` в handler, `OrderRepository` в service): слои не импортируют друг друга, тесты подставляют fake без PostgreSQL. Альтернатива — SQL и валидация в хендлерах. Слои оставляют HTTP, правила и хранилище независимыми.

### Контекст через весь стек

`ctx` идёт handler → service → repository → `QueryContext` / `ExecContext`. Отмена клиента прерывает запрос к БД. Поверх `r.Context()` хендлер ставит свой дедлайн (`REQUEST_TIMEOUT`, по умолчанию 3 с). В `handleError`: клиент ушёл (`context.Canceled`) — ответ не пишем; свой дедлайн (`DeadlineExceeded`) — `503`. Альтернатива — один общий timeout на сервере без проброса ctx в SQL. Тогда отвалившийся клиент продолжал бы держать соединение с Postgres.

### `database/sql` + pgx stdlib, без ORM

Запросы и транзакции написаны явно: создание заказа и позиций в одной транзакции, ошибки репозитория не смешиваются с HTTP. Альтернатива — GORM, sqlc или драйвер `lib/pq`. `lib/pq` в maintenance mode, автор советует pgx. Явный SQL даёт контроль над `RETURNING` и rollback без скрытого lifecycle ORM.

### Цена в копейках (`int64`)

`float` хранит десятичные дроби приближённо: на сложении копейки «уплывают». Альтернатива — `numeric` в БД и `decimal` в Go; она нужна, когда есть доли копейки или деление с округлением. Здесь сумм в долях нет — `int64` копеек достаточно.

### Доменные ошибки (`apperror`)

Репозиторий отдаёт `ErrOrderNotFound`, сервис — `ValidationError`; handler мапит их в `404` / `400`. Альтернатива — коды HTTP внутри service. Так сервис не зависит от `net/http`, а хендлер — единственное место статусов.

### goose + Compose, приложение с хоста

БД в Docker, API — `go run` и `DATABASE_URL`. Альтернатива — всё в Compose или только `schema.sql`. Миграции версионируются и откатываются; локальный цикл ближе к обычной разработке на Go.

### Graceful shutdown и раздельный health

SIGINT/SIGTERM → `Shutdown` с таймаутом 10 с. Liveness не трогает БД, readiness делает `Ping`. Альтернатива — один `/health` и резкий `os.Exit`. Так оркестратор может отличить «процесс жив» от «БД недоступна».

## Тесты

```bash
go test ./...
```

- **config** — обязательный `DATABASE_URL`, значения по умолчанию и переопределение порта/таймаутов.
- **service** — табличные кейсы `validateOrder` (адрес, позиции, количество); `CreateOrder` не вызывает репозиторий при ошибке валидации и вызывает при валидном заказе (fake repo).
- **handler** — HTTP через `httptest` и fake service: невалидный id → `400`, нет заказа → `404`, битый JSON → `400`, успешный POST → `201`.

Репозиторий и живая БД тестами не покрыты.

## Дальнейшие планы

- Настоящая аутентификация вместо заглушки `Auth`
- Интеграционные тесты репозитория (транзакции, каскадное удаление)
- Обновление позиций заказа в той же транзакции, что и шапка
- Пагинация и фильтр списка заказов
- Смена статуса заказа отдельным сценарием
