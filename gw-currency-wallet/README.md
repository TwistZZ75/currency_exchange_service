# gw-currency-wallet

REST API кошелька с обменом валют. Поддерживает регистрацию, JWT-авторизацию,
пополнение, вывод и обмен валют по курсам `gw-exchanger` (gRPC). О крупных
операциях отправляет события в Kafka для аналитики и уведомлений.

## Что делает

- **Регистрация и логин**: bcrypt для паролей, JWT (HS256) для доступа.
- **Кошелёк**: баланс, пополнение, вывод в трёх валютах (`USD`, `RUB`, `EUR`).
- **Обмен валют**: через gRPC-клиент к `gw-exchanger`, с кэшем курсов (TTL 30 сек).
- **Kafka-producer**: при сумме ≥ `LARGE_TRANSFER_THRESHOLD` шлёт событие в топик `large_transfers`.
- **Идемпотентность**: заголовок `Idempotency-Key` — повторный запрос не меняет баланс.
- **Middleware**: CORS, Rate limit (per-IP), Body limit, Timeout, Request-ID, JWT, Structured logging.

## Архитектура

```
         ┌──────────────────────┐
         │       клиент         │
         └──────────┬───────────┘
                    │ HTTP + JWT
                    ▼
         ┌──────────────────────┐
         │  gw-currency-wallet  │
         │      (этот сервис)   │
         └──┬────────────┬──────┘
            │            │
            │ gRPC       │ produces
            ▼            ▼
   ┌────────────────┐  ┌─────────┐
   │  gw-exchanger  │  │  Kafka  │──► gw-analytics
   │  (курсы)       │  │         │──► gw-notifier
   └────────────────┘  └─────────┘
            ▲
            │ SQL
            │
      ┌────────────────┐
      │   wallet-db    │
      │  (PostgreSQL)  │
      └────────────────┘
```

## HTTP API

| Метод | Путь                      | Auth | Описание                          |
|-------|---------------------------|:----:|-----------------------------------|
| POST  | `/api/v1/register`        | нет  | регистрация                       |
| POST  | `/api/v1/login`           | нет  | логин → JWT                       |
| GET   | `/api/v1/balance`         | да   | баланс                            |
| POST  | `/api/v1/wallet/deposit`  | да   | пополнение                        |
| POST  | `/api/v1/wallet/withdraw` | да   | вывод                             |
| GET   | `/api/v1/exchange/rates`  | да   | курсы валют                       |
| POST  | `/api/v1/exchange`        | да   | обмен валюты                      |
| GET   | `/swagger/index.html`     | нет  | Swagger UI (если сгенерирован `docs/`) |

### Заголовки

| Заголовок         | Обязателен     | Описание                                         |
|-------------------|----------------|--------------------------------------------------|
| `Authorization`   | для защищённых | `Bearer <JWT>`                                   |
| `Content-Type`    | для POST       | `application/json`                               |
| `Idempotency-Key` | нет            | UUID или строка; при повторе вернёт тот же ответ |
| `X-Request-ID`    | нет            | ID для трассировки (иначе генерируется)          |

### Примеры

```bash
# Регистрация
curl -X POST http://localhost:8080/api/v1/register \
  -H 'Content-Type: application/json' \
  -d '{"username":"user1","password":"pass123","email":"u@example.com"}'

# Логин → токен
curl -X POST http://localhost:8080/api/v1/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"user1","password":"pass123"}'

# Баланс
curl http://localhost:8080/api/v1/balance \
  -H "Authorization: Bearer $TOKEN"

# Пополнение 1000 USD
curl -X POST http://localhost:8080/api/v1/wallet/deposit \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"amount":1000,"currency":"USD"}'

# Вывод 50 USD
curl -X POST http://localhost:8080/api/v1/wallet/withdraw \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"amount":50,"currency":"USD"}'

# Курсы
curl http://localhost:8080/api/v1/exchange/rates \
  -H "Authorization: Bearer $TOKEN"

# Обмен 100 USD → EUR
curl -X POST http://localhost:8080/api/v1/exchange \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"from_currency":"USD","to_currency":"EUR","amount":100}'
```

### Пример ответа `register`

**201 Created**

```json
{ "message": "User registered successfully" }
```

**400 Bad Request**

```json
{ "error": "Username or email already exists" }
```

### Пример ответа `login`

**200 OK**

```json
{ "token": "eyJhbGciOiJIUzI1NiIs..." }
```

**401 Unauthorized**

```json
{ "error": "Invalid username or password" }
```

### Пример ответа `balance`

```json
{
  "balance": {
    "USD": 100.00,
    "RUB": 5000.00,
    "EUR": 0.00
  }
}
```

### Пример ответа `deposit`

```json
{
  "message": "Account topped up successfully",
  "new_balance": {
    "USD": 1100.00,
    "RUB": 5000.00,
    "EUR": 0.00
  }
}
```

### Пример ответа `withdraw`

```json
{
  "message": "Withdrawal successful",
  "new_balance": { "USD": 1050.00, "RUB": 5000.00, "EUR": 0.00 }
}
```

### Пример ответа `rates`

```json
{
  "rates": {
    "USD": 90.00,
    "EUR": 100.00,
    "RUB": 1.00
  }
}
```

### Пример ответа `exchange`

```json
{
  "message": "Exchange successful",
  "exchanged_amount": 90.00,
  "rate": 0.9,
  "new_balance": { "USD": 950.00, "RUB": 5000.00, "EUR": 90.00 }
}
```

## Kafka

При сумме ≥ `LARGE_TRANSFER_THRESHOLD` (по умолчанию 30000) операция публикуется
в топик `KAFKA_TOPIC` (`large_transfers`). События читают:

- **gw-analytics** — сохраняет в ClickHouse, считает агрегации.
- **gw-notifier** — сохраняет в MongoDB.

Оба — отдельные consumer groups, получают копии независимо.

**Формат сообщения**:

```json
{
  "transaction_id": "550e8400-e29b-41d4-a716-446655440000",
  "user_id": 1,
  "type": "deposit",
  "from_currency": "",
  "to_currency": "USD",
  "amount": 50000,
  "timestamp": "2026-10-05T02:47:57Z"
}
```

## Гарантии и особенности

- **Идемпотентность**: заголовок `Idempotency-Key`. Если он UUID — используется как есть;
  если строка — превращается в детерминированный UUID v5. Повторный запрос возвращает
  результат первой операции.
- **At-least-once в Kafka**: событие шлётся асинхронно. Если Kafka недоступна — операция
  всё равно успешна, ошибка уходит только в лог.
- **Транзакции БД**: баланс и запись в `operations` меняются атомарно. При конфликте
  `transaction_id` — откат, возврат текущего состояния (`ON CONFLICT DO NOTHING`).
- **Кэш курсов**: TTL 30 сек. Сбрасывается после каждой операции обмена, чтобы следующий
  запрос шёл за свежими данными.

## Схема БД

```sql
CREATE TABLE users (
    id         BIGSERIAL PRIMARY KEY,
    username   TEXT UNIQUE NOT NULL,
    email      TEXT UNIQUE NOT NULL,
    password   TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE wallets (
    user_id    BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    usd        NUMERIC(20, 2) NOT NULL DEFAULT 0 CHECK (usd >= 0),
    rub        NUMERIC(20, 2) NOT NULL DEFAULT 0 CHECK (rub >= 0),
    eur        NUMERIC(20, 2) NOT NULL DEFAULT 0 CHECK (eur >= 0),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE operations (
    id               BIGSERIAL PRIMARY KEY,
    transaction_id   UUID UNIQUE NOT NULL DEFAULT gen_random_uuid(),
    user_id          BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type             TEXT NOT NULL CHECK (type IN ('deposit', 'withdraw', 'exchange')),
    from_currency    TEXT,
    to_currency      TEXT,
    amount           NUMERIC(20, 2) NOT NULL,
    exchanged_amount NUMERIC(20, 2),
    rate             NUMERIC(20, 8),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

- `UNIQUE(transaction_id)` — основа идемпотентности.
- `CHECK (usd >= 0)` и т.д. — БД сама не даст уйти в минус.
- `CHECK (type IN (...))` — только разрешённые типы операций.

## Конфигурация

Все параметры читаются из `config.env` (локально) или из env (Docker). Любой
пропущенный ключ — сервис не стартует.

| Переменная                 | Пример                  | Описание                             |
|----------------------------|-------------------------|--------------------------------------|
| `HTTP_PORT`                | `8080`                  | порт REST API                        |
| `HTTP_READ_TIMEOUT`        | `10s`                   | таймаут чтения запроса               |
| `HTTP_WRITE_TIMEOUT`       | `10s`                   | таймаут записи ответа                |
| `DB_HOST`                  | `wallet-db`             | хост Postgres                        |
| `DB_PORT`                  | `5432`                  | порт Postgres                        |
| `DB_USER`                  | `postgres`              | пользователь                         |
| `DB_PASSWORD`              | `postgres`              | пароль                               |
| `DB_NAME`                  | `wallet`                | имя базы                             |
| `DB_SSLMODE`               | `disable`               | ssl-режим                            |
| `JWT_SECRET`               | `change-me`             | секрет подписи JWT                   |
| `JWT_TTL`                  | `24h`                   | время жизни токена                   |
| `EXCHANGER_ADDR`           | `gw-exchanger:50051`    | адрес gRPC-сервиса exchanger         |
| `EXCHANGER_TIMEOUT`        | `3s`                    | таймаут gRPC-запросов                |
| `RATES_CACHE_TTL`          | `30s`                   | TTL кэша курсов                      |
| `KAFKA_BROKERS`            | `kafka:9092`            | брокеры (через запятую)              |
| `KAFKA_TOPIC`              | `large_transfers`       | топик для крупных операций           |
| `LARGE_TRANSFER_THRESHOLD` | `30000`                 | порог крупной операции               |
| `LOG_LEVEL`                | `info`                  | `debug` / `info` / `warn` / `error`  |
| `SHUTDOWN_TIMEOUT`         | `10s`                   | таймаут graceful shutdown            |

## Запуск

### В Docker

```bash
docker compose up -d wallet-db gw-exchanger kafka kafka-init gw-currency-wallet
docker compose logs -f gw-currency-wallet
```

### Swagger UI

```bash
make swag              # сгенерировать docs/
make docker-build      # пересобрать образ
make docker-up         # поднять
# открыть http://localhost:8080/swagger/index.html
```

## Проверка работоспособности

```bash
# 1. Все контейнеры подняты
make ps

# 2. Логи сервиса
make logs
# ожидаемо:
#   storage connected          db=wallet host=wallet-db
#   exchanger client connected addr=gw-exchanger:50051
#   kafka producer ready       brokers=[kafka:9092] topic=large_transfers
#   http server started        port=8080

# 3. API отвечает
curl http://localhost:8080/health
# {"status":"ok"}
```

### E2E: полный сценарий

```bash
# 1. Регистрация
curl -X POST http://localhost:8080/api/v1/register \
  -H 'Content-Type: application/json' \
  -d '{"username":"user1","password":"pass123","email":"u@example.com"}'

# 2. Логин → токен
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"user1","password":"pass123"}' | jq -r .token)

# 3. Пополнение 50000 USD (крупная операция → Kafka)
curl -X POST http://localhost:8080/api/v1/wallet/deposit \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"amount":50000,"currency":"USD"}'

# 4. Курсы
curl http://localhost:8080/api/v1/exchange/rates \
  -H "Authorization: Bearer $TOKEN"

# 5. Обмен 100 USD → EUR
curl -X POST http://localhost:8080/api/v1/exchange \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"from_currency":"USD","to_currency":"EUR","amount":100}'

# 6. Проверить баланс
curl http://localhost:8080/api/v1/balance \
  -H "Authorization: Bearer $TOKEN"

# 7. Проверить, что событие ушло в Kafka
docker compose exec kafka kafka-console-consumer \
  --bootstrap-server kafka:9092 --topic large_transfers \
  --from-beginning --max-messages 5
```

### Проверка идемпотентности

```bash
KEY="550e8400-e29b-41d4-a716-446655440000"

# Первый вызов
curl -X POST http://localhost:8080/api/v1/wallet/deposit \
  -H "Authorization: Bearer $TOKEN" \
  -H "Idempotency-Key: $KEY" \
  -H 'Content-Type: application/json' \
  -d '{"amount":1000,"currency":"USD"}'

# Повтор с тем же ключом — баланс НЕ увеличится, ответ идентичен
curl -X POST http://localhost:8080/api/v1/wallet/deposit \
  -H "Authorization: Bearer $TOKEN" \
  -H "Idempotency-Key: $KEY" \
  -H 'Content-Type: application/json' \
  -d '{"amount":1000,"currency":"USD"}'
```

Проверка в БД:

```bash
docker compose exec wallet-db psql -U postgres -d wallet \
  -c "SELECT transaction_id, amount, type FROM operations ORDER BY id DESC LIMIT 3"
```

Строка с этим `transaction_id` должна быть **одна**.