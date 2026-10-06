# gw-exchanger

gRPC-сервис курсов валют. Хранит курсы в PostgreSQL, отдаёт их другим сервисам
(`gw-currency-wallet`) по gRPC. Поддерживаемые валюты: `USD`, `RUB`, `EUR`.

## Что делает

- **gRPC-сервер** на `:50051`, реализует `ExchangeService`.
- **PostgreSQL** как хранилище курсов (абстракция через интерфейс `storages.Storage`).
- **Курсы к рублю**: для каждой валюты хранится `rate_to_rub`. Курс пары считается как
  `rate_from / rate_to`.
- **Health-check** (`grpc.health.v1.Health`) и **reflection** — для `grpcurl` и
  service discovery.
- **Structured logs** (JSON, slog) с интерсептором: метод, длительность, gRPC-код.
- **Graceful shutdown** по SIGINT/SIGTERM.

## Архитектура

```
         ┌──────────────────────┐
         │  gw-currency-wallet  │
         └──────────┬───────────┘
                    │ gRPC
                    ▼
         ┌──────────────────────┐
         │    gw-exchanger      │
         │     (этот сервис)    │
         └──────────┬───────────┘
                    │ SQL
                    ▼
         ┌──────────────────────┐
         │      postgres        │
         │     (exchanger)      │
         └──────────────────────┘
```

## gRPC API

Proto: `proto-exchange/exchange/exchange.proto`.

| Метод                       | Запрос             | Ответ                    | Описание                         |
|-----------------------------|--------------------|--------------------------|----------------------------------|
| `GetExchangeMap`            | `Empty`            | `ExchangeMapResponse`    | курсы всех валют                 |
| `GetExchangeRateForCurrency`| `CurrencyRequest`  | `ExchangeRateResponse`   | курс конкретной пары             |

### Сервисы gRPC, доступные для reflection

```bash
grpcurl -plaintext localhost:50051 list
```

```
exchange.ExchangeService
grpc.health.v1.Health
grpc.reflection.v1.ServerReflection
```

### Примеры вызова

```bash
# Все курсы
grpcurl -plaintext localhost:50051 exchange.ExchangeService/GetExchangeMap

# Курс конкретной пары
grpcurl -plaintext -d '{"from_currency":"USD","to_currency":"RUB"}' \
  localhost:50051 exchange.ExchangeService/GetExchangeRateForCurrency

# Health-check
grpcurl -plaintext -d '{"service":""}' \
  localhost:50051 grpc.health.v1.Health/Check
```

### Пример ответа `GetExchangeMap`

```json
{
  "rates": {
    "USD": 90.0,
    "EUR": 100.0,
    "RUB": 1.0
  }
}
```

### Пример ответа `GetExchangeRateForCurrency`

```json
{
  "fromCurrency": "USD",
  "toCurrency": "RUB",
  "rate": 90.0
}
```

### Примеры ошибок

Невалидные поля:

```bash
grpcurl -plaintext -d '{}' localhost:50051 exchange.ExchangeService/GetExchangeRateForCurrency
```

```
ERROR:
  Code: InvalidArgument
  Message: from_currency and to_currency are required
```

Неподдерживаемая валюта:

```bash
grpcurl -plaintext -d '{"from_currency":"BTC","to_currency":"USD"}' \
  localhost:50051 exchange.ExchangeService/GetExchangeRateForCurrency
```

```
ERROR:
  Code: InvalidArgument
  Message: unsupported currency: BTC
```

## Формула курса

В БД хранится курс каждой валюты **к рублю** (`rate_to_rub`). Курс пары считается как:

```
rate(from → to) = rate_to_rub(from) / rate_to_rub(to)
```

Примеры:

- `USD → RUB` = 90 / 1 = 90
- `RUB → USD` = 1 / 90 ≈ 0.0111
- `USD → EUR` = 90 / 100 = 0.9

## Схема БД

```sql
CREATE TABLE IF NOT EXISTS rates (
    currency     TEXT PRIMARY KEY,
    rate_to_rub  NUMERIC(20, 8) NOT NULL CHECK (rate_to_rub > 0),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO rates (currency, rate_to_rub) VALUES
    ('USD', 90.00000000),
    ('EUR', 100.00000000),
    ('RUB', 1.00000000)
ON CONFLICT (currency) DO UPDATE
SET rate_to_rub = EXCLUDED.rate_to_rub,
    updated_at  = NOW();
```

- `currency` — код валюты (`USD`, `RUB`, `EUR`).
- `rate_to_rub` — курс валюты к рублю, положительное число.
- Начальные курсы вставляются при первом старте БД.

## Конфигурация

Все параметры читаются из `config.env` (локально) или из env (Docker). Любой
пропущенный ключ — сервис не стартует.

| Переменная         | Пример        | Описание                             |
|--------------------|---------------|--------------------------------------|
| `GRPC_PORT`        | `50051`       | порт gRPC-сервера                    |
| `DB_HOST`          | `postgres`    | хост Postgres                        |
| `DB_PORT`          | `5432`        | порт Postgres                        |
| `DB_USER`          | `postgres`    | пользователь                         |
| `DB_PASSWORD`      | `postgres`    | пароль                               |
| `DB_NAME`          | `exchanger`   | имя базы                             |
| `DB_SSLMODE`       | `disable`     | ssl-режим                            |
| `LOG_LEVEL`        | `info`        | `debug` / `info` / `warn` / `error`  |
| `SHUTDOWN_TIMEOUT` | `10s`         | таймаут graceful shutdown            |

## Запуск

### В Docker

```bash
docker compose up -d postgres gw-exchanger
docker compose logs -f gw-exchanger
```

### Swagger

Сервис **не отдаёт HTTP**, поэтому Swagger не применяется. Для документации API
используйте proto-файл `proto-exchange/exchange/exchange.proto` и reflection
(`grpcurl -plaintext localhost:50051 describe exchange.ExchangeService`).

## Проверка работоспособности

```bash
# 1. Все контейнеры подняты
make ps

# 2. Логи сервиса
make logs
# ожидаемо:
#   storage connected db=exchanger host=postgres
#   grpc server started port=50051

# 3. gRPC отвечает (через reflection)
grpcurl -plaintext localhost:50051 list
# exchange.ExchangeService
# grpc.health.v1.Health
# grpc.reflection.v1.ServerReflection

# 4. Все курсы
grpcurl -plaintext localhost:50051 exchange.ExchangeService/GetExchangeMap
# {"rates":{"USD":90,"EUR":100,"RUB":1}}

# 5. Health-check
grpcurl -plaintext -d '{"service":""}' localhost:50051 grpc.health.v1.Health/Check
# {"status":"SERVING"}
```

### Проверка всех пар

```bash
# USD → RUB
grpcurl -plaintext -d '{"from_currency":"USD","to_currency":"RUB"}' \
  localhost:50051 exchange.ExchangeService/GetExchangeRateForCurrency

# RUB → USD (обратный)
grpcurl -plaintext -d '{"from_currency":"RUB","to_currency":"USD"}' \
  localhost:50051 exchange.ExchangeService/GetExchangeRateForCurrency

# USD → EUR
grpcurl -plaintext -d '{"from_currency":"USD","to_currency":"EUR"}' \
  localhost:50051 exchange.ExchangeService/GetExchangeRateForCurrency

# Одинаковая валюта (должно вернуть 1.0)
grpcurl -plaintext -d '{"from_currency":"USD","to_currency":"USD"}' \
  localhost:50051 exchange.ExchangeService/GetExchangeRateForCurrency
```

### Проверка ошибок

```bash
# Пустые поля → InvalidArgument
grpcurl -plaintext -d '{}' \
  localhost:50051 exchange.ExchangeService/GetExchangeRateForCurrency

# Неподдерживаемая валюта → InvalidArgument
grpcurl -plaintext -d '{"from_currency":"BTC","to_currency":"USD"}' \
  localhost:50051 exchange.ExchangeService/GetExchangeRateForCurrency
```

### Проверка, что курсы в БД

```bash
docker compose exec postgres psql -U postgres -d exchanger \
  -c "SELECT currency, rate_to_rub FROM rates ORDER BY currency"
```

### Проверка E2E с wallet

```bash
# wallet ходит в exchanger за курсами
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"user1","password":"pass123"}' | jq -r .token)

curl http://localhost:8080/api/v1/exchange/rates \
  -H "Authorization: Bearer $TOKEN"
# {"rates":{"USD":90,"EUR":100,"RUB":1}}
```

Если wallet отвечает с курсами — exchanger работает корректно и доступен по
внутреннему имени `gw-exchanger:50051`.