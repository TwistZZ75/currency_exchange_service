# README для всего проекта

```markdown
# Currency Exchange Service

Микросервисная система обмена валют с кошельком, аналитикой и уведомлениями
о крупных переводах. Демонстрирует event-driven архитектуру на Go с gRPC, Kafka,
PostgreSQL, MongoDB и ClickHouse.

## Что делает система

Пользователь регистрируется, пополняет кошелёк, обменивает валюты и смотрит баланс.
Все крупные операции (≥ 30 000) уходят в Kafka — их читают сервисы аналитики и
уведомлений, сохраняя данные в ClickHouse и MongoDB соответственно.

## Архитектура

```
                              ┌────────────────────┐
                              │      клиент        │
                              └─────────┬──────────┘
                                        │ HTTP + JWT
                                        ▼
                              ┌────────────────────┐
                              │ gw-currency-wallet │
                              │   (REST + JWT)     │
                              └─┬────────────┬─────┘
                                │            │
                       gRPC     │            │ produces
                                ▼            ▼
                     ┌────────────────┐  ┌──────────┐
                     │  gw-exchanger  │  │  Kafka   │
                     │ (курсы валют)  │  │ (KRaft)  │
                     └───────┬────────┘  └────┬─────┘
                             │                │
                             │ SQL            ├────────────────────┐
                             ▼                │                    │
                     ┌────────────────┐       ▼                    ▼
                     │   postgres     │  ┌─────────────┐    ┌────────────────┐
                     │  (exchanger)   │  │ gw-notifier │    │ gw-analytics   │
                     └────────────────┘  │  → MongoDB  │    │  → ClickHouse  │
                                          └─────────────┘    └────────────────┘

                     ┌────────────────┐
                     │   wallet-db    │
                     │  (PostgreSQL)  │◄──── gw-currency-wallet (SQL)
                     └────────────────┘
```

## Сервисы

| Сервис                  | Роль                                                | Протокол        | Порт  |
|-------------------------|-----------------------------------------------------|-----------------|-------|
| **gw-exchanger**        | Курсы валют `USD`/`RUB`/`EUR`                       | gRPC            | 50051 |
| **gw-currency-wallet**  | Кошелёк: регистрация, баланс, пополнение, обмен     | HTTP REST + JWT | 8080  |
| **gw-notifier**         | Сохраняет крупные переводы в MongoDB                | Kafka → MongoDB | 9090* |
| **gw-analytics**        | Аналитика крупных переводов, агрегации по периодам  | Kafka → CH + HTTP | 8082, 9092* |

*Порт `9090` и `9092` — это порты `/metrics` для Prometheus. `gw-notifier` не отдаёт HTTP API, `gw-analytics` отдаёт на `8082`.

## Технологии

- **Язык**: Go 1.25+
- **HTTP**: Gin
- **gRPC**: google.golang.org/grpc + Protocol Buffers
- **Базы**: PostgreSQL (pgx/v5), MongoDB (mongo-driver), ClickHouse (clickhouse-go/v2)
- **Брокер**: Kafka (KRaft-режим, без Zookeeper)
- **Аутентификация**: JWT HS256 + bcrypt
- **Метрики**: Prometheus
- **Логи**: slog JSON
- **Контейнеризация**: Docker + Docker Compose
- **Тесты**: testify + интеграционные тесты через env-флаги

## Быстрый старт

### 1. Клонирование

```bash
git clone https://github.com/TwistZZ75/currency_exchange_service.git
cd currency_exchange_service
```

### 2. Генерация proto

```bash
cd proto-exchange
protoc --go_out=. --go_opt=paths=source_relative \
       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
       exchange/exchange.proto
cd ..
```

### 3. Запуск всего стека

```bash
docker compose up -d
```

Что поднимется:

- `postgres` — БД exchanger'а
- `wallet-db` — БД кошелька
- `kafka`, `kafka-init` — брокер + создание топиков
- `mongo` — БД notifier'а
- `clickhouse` — БД analytics
- `gw-exchanger`, `gw-currency-wallet`, `gw-notifier`, `gw-analytics` — сервисы

### 4. Проверка

```bash
docker compose ps
```

Ожидаемо: все `Up` (или `healthy`), `gw_kafka_init` — `Exited (0)` (это норма — init-контейнер
отработал и завершился).

```bash
curl http://localhost:8080/health
# {"status":"ok"}
```

## E2E-проверка

Полный сценарий: регистрация → логин → пополнение → обмен → аналитика.

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

# 4. Курсы от exchanger'а
curl http://localhost:8080/api/v1/exchange/rates \
  -H "Authorization: Bearer $TOKEN"
# {"rates":{"USD":90,"EUR":100,"RUB":1}}

# 5. Обмен 100 USD → EUR
curl -X POST http://localhost:8080/api/v1/exchange \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"from_currency":"USD","to_currency":"EUR","amount":100}'

# 6. Баланс
curl http://localhost:8080/api/v1/balance \
  -H "Authorization: Bearer $TOKEN"
```

Через 1–2 секунды проверяем, что событие дошло до аналитики и уведомлений:

```bash
# В ClickHouse
docker compose exec clickhouse clickhouse-client \
  --query "SELECT transaction_id, type, amount FROM analytics.events FINAL ORDER BY event_time DESC LIMIT 3"

# В MongoDB
docker compose exec mongo mongosh notification --quiet \
  --eval 'db.large_transfers.find().pretty()'

# Через HTTP API аналитики
curl "http://localhost:8082/api/v1/analytics/events?period=1m"
```

## Структура репозитория

```
currency_exchange_service/
├── proto-exchange/                # gRPC proto + сгенерированный код
│   ├── exchange/
│   │   ├── exchange.proto
│   │   ├── exchange.pb.go
│   │   └── exchange_grpc.pb.go
│   └── go.mod
├── gw-exchanger/                  # gRPC-сервис курсов
│   ├── cmd/
│   ├── internal/
│   ├── config.env
│   ├── Dockerfile
│   ├── Makefile
│   └── README.md
├── gw-currency-wallet/            # HTTP REST API кошелька
│   ├── cmd/
│   ├── internal/
│   ├── docs/                      # swagger
│   ├── config.env
│   ├── Dockerfile
│   ├── Makefile
│   └── README.md
├── gw-notifier/                   # Kafka → MongoDB
│   ├── cmd/
│   ├── internal/
│   ├── config.env
│   ├── Dockerfile
│   ├── Makefile
│   └── README.md
├── gw-analytics/                  # Kafka → ClickHouse + HTTP API
│   ├── cmd/
│   ├── internal/
│   ├── docs/                      # swagger
│   ├── config.env
│   ├── Dockerfile
│   ├── Makefile
│   └── README.md
├── docker-compose.yml
├── go.work                        # локальный workspace (для разработки)
└── README.md                      # этот файл
```

## Взаимодействие сервисов

| Откуда                  | Куда                    | Как        | Что передаётся                       |
|-------------------------|-------------------------|------------|--------------------------------------|
| `gw-currency-wallet`    | `gw-exchanger`          | gRPC       | запросы курсов                       |
| `gw-currency-wallet`    | `wallet-db`             | SQL        | пользователи, кошельки, операции     |
| `gw-currency-wallet`    | `Kafka` (`large_transfers`) | producer | события крупных операций            |
| `gw-notifier`           | `Kafka` (`large_transfers`) | consumer | те же события                       |
| `gw-notifier`           | `MongoDB`               | driver     | документы о крупных переводах        |
| `gw-analytics`          | `Kafka` (`large_transfers`) | consumer | те же события                       |
| `gw-analytics`          | `ClickHouse`            | driver     | события (ReplacingMergeTree)         |
| `gw-exchanger`          | `postgres`              | SQL        | таблица `rates`                      |

## Топики Kafka

| Топик                  | Producer                | Consumers                              |
|------------------------|-------------------------|----------------------------------------|
| `large_transfers`      | `gw-currency-wallet`    | `gw-notifier`, `gw-analytics`          |
| `large_transfers_dlq`  | `gw-notifier`, `gw-analytics` | (для ручного разбора)            |

Оба consumer'а — **разные consumer groups** (`gw-notifier`, `gw-analytics`), поэтому
получают копии сообщений независимо.

## Конфигурация

У каждого сервиса — свой `config.env`, который читается локально или через env в Docker.
Общие переменные инфраструктуры (в `docker-compose.yml`):

| Переменная        | Значение        |
|-------------------|-----------------|
| `KAFKA_BROKERS`   | `kafka:9092`    |
| `MONGO_URI`       | `mongodb://mongo:27017` |
| `CLICKHOUSE_ADDR` | `clickhouse:9000` |
| `DB_HOST` (wallet)      | `wallet-db`     |
| `DB_HOST` (exchanger)   | `postgres`      |
| `EXCHANGER_ADDR`  | `gw-exchanger:50051` |

Подробнее — в README каждого сервиса.

## Полезные команды

```bash
# Статус всех контейнеров
docker compose ps

# Логи конкретного сервиса
docker compose logs -f gw-currency-wallet
docker compose logs -f gw-exchanger
docker compose logs -f gw-notifier
docker compose logs -f gw-analytics

# Остановить всё
docker compose down

# Остановить и удалить volumes (сброс всех БД)
docker compose down -v

# Пересобрать один сервис
docker compose build --no-cache gw-currency-wallet

# Список топиков Kafka
docker compose exec kafka kafka-topics --list --bootstrap-server kafka:9092

# Проверить содержимое DLQ
docker compose exec kafka kafka-console-consumer \
  --bootstrap-server kafka:9092 --topic large_transfers_dlq \
  --from-beginning --max-messages 5
```

## Тесты

Каждый сервис содержит unit-тесты (без внешних зависимостей) и интеграционные
(требуют запущенных БД/брокера).

```bash
# Unit-тесты для сервиса
cd gw-currency-wallet
make test-unit

# Интеграционные
docker compose up -d wallet-db
make test-integration
```

Подробнее — в README каждого сервиса.

## Swagger

HTTP-сервисы отдают Swagger UI:

- **gw-currency-wallet**: http://localhost:8080/swagger/index.html
- **gw-analytics**: http://localhost:8082/swagger/index.html

`gw-exchanger` использует gRPC — документация через proto-файл и reflection:

```bash
grpcurl -plaintext localhost:50051 describe exchange.ExchangeService
```

## Метрики Prometheus

| Сервис            | Порт  | Путь       |
|-------------------|-------|------------|
| `gw-notifier`     | 9090  | `/metrics` |
| `gw-analytics`    | 9092  | `/metrics` |

```bash
curl http://localhost:9090/metrics | grep gw_notification
curl http://localhost:9092/metrics | grep gw_analytics
```

## Лицензия

Учебный проект.
```