# gw-analytics

Аналитический сервис: читает события о крупных денежных переводах из Kafka,
пишет в ClickHouse, отдаёт агрегации по периодам через REST API и Prometheus-метрики.

## Что делает

- **Consumes** события из Kafka-топика `large_transfers` (source: `gw-currency-wallet`).
- **Сохраняет** их в ClickHouse (`analytics.events`, `ReplacingMergeTree`).
- **Отдаёт** агрегации по HTTP:
  - количество событий по `type`/`status`;
  - статистика latency (avg / p95 / p99);
  - частота ошибок (`error_rate`).
- **Публикует** метрики Prometheus на `/metrics`.
- **Идемпотентен**: повторные сообщения не создают дубликаты (unique по `transaction_id`).
- **Устойчив**: at-least-once + DLQ для битых сообщений + retry с backoff.

## Архитектура

```
                 ┌──────────────────────┐
                 │  gw-currency-wallet  │
                 └──────────┬───────────┘
                            │ produces
                            ▼
                 ┌──────────────────────┐
                 │        Kafka         │
                 │  (KRaft, no ZK)      │
                 └─────┬──────────┬─────┘
                       │          │
        ┌──────────────┘          └──────────────┐
        ▼                                        ▼
┌──────────────────┐                    ┌──────────────────┐
│  gw-analytics    │                    │  gw-notifier     │
│  (этот сервис)   │                    │  → MongoDB       │
└────────┬─────────┘                    └──────────────────┘
         │
         ├──► ClickHouse (analytics.events)
         ├──► HTTP API (:8082)
         └──► /metrics (:9092)
```

## HTTP API

| Метод | Путь                              | Описание                                    |
|-------|-----------------------------------|---------------------------------------------|
| GET   | `/health`                         | health-check                                |
| GET   | `/api/v1/analytics/events`        | количество событий по `type`/`status`      |
| GET   | `/api/v1/analytics/latency`       | avg / p95 / p99 latency                     |
| GET   | `/api/v1/analytics/errors`        | error rate по `type`                        |
| GET   | `/swagger/index.html`             | Swagger UI (если сгенерирован `docs/`)      |

### Параметры запроса

| Параметр | Тип   | По умолчанию       | Возможные значения           |
|----------|-------|--------------------|------------------------------|
| `period` | query | `1m`               | `1m`, `5m`, `1h`, `1d`, `1w` |
| `from`   | query | `now - 1h`         | RFC3339                      |
| `to`     | query | `now`              | RFC3339                      |

### Примеры

```bash
# Количество событий за последний час с шагом 5 минут
curl "http://localhost:8082/api/v1/analytics/events?period=5m"

# Latency за сутки
curl "http://localhost:8082/api/v1/analytics/latency?period=1h&from=2026-10-05T00:00:00Z"

# Частота ошибок за неделю
curl "http://localhost:8082/api/v1/analytics/errors?period=1d"
```

### Пример ответа `events`

```json
{
  "from":   "2026-10-05T02:00:00Z",
  "to":     "2026-10-05T03:00:00Z",
  "period": "5m",
  "items": [
    {
      "bucket": "2026-10-05T02:55:00Z",
      "type":   "deposit",
      "status": "success",
      "count":  12
    }
  ]
}
```

### Пример ответа `latency`

```json
{
  "items": [
    {
      "bucket": "2026-10-05T02:55:00Z",
      "type":   "exchange",
      "avg_ms": 34.5,
      "p95_ms": 120.0,
      "p99_ms": 210.0,
      "count":  8
    }
  ]
}
```

### Пример ответа `errors`

```json
{
  "items": [
    {
      "bucket":      "2026-10-05T02:55:00Z",
      "type":        "withdraw",
      "error_count": 1,
      "total_count": 50,
      "error_rate":  0.02
    }
  ]
}
```

## Метрики Prometheus

Отдаются на `METRICS_PORT` (по умолчанию `9092`) по пути `/metrics`.

| Метрика                                              | Тип       | Описание                                       |
|------------------------------------------------------|-----------|------------------------------------------------|
| `gw_analytics_events_processed_total{status}`        | counter   | обработано событий (`saved` / `failed`)        |
| `gw_analytics_dlq_events_total`                      | counter   | ушло в DLQ                                     |
| `gw_analytics_clickhouse_errors_total`               | counter   | batch insert исчерпал retry                    |
| `gw_analytics_retry_attempts`                        | histogram | распределение числа retry                      |
| `gw_analytics_batch_size`                            | histogram | размер батча при вставке                       |
| `gw_analytics_end_to_end_latency_seconds`            | histogram | latency от event_time до записи в ClickHouse   |
| `gw_analytics_clickhouse_write_duration_seconds`     | histogram | длительность insert (включая retry)            |
| `gw_analytics_http_requests_total{method,path,status}` | counter | HTTP-запросы                                   |

## Гарантии доставки

- **At-least-once**: offset коммитится **после** успешной записи батча в ClickHouse.
- **Идемпотентность**: `ReplacingMergeTree(version)` + `FINAL` в запросах —
  повторная вставка того же `transaction_id` не даёт дублей при чтении.
- **DLQ**: битые сообщения (невалидный JSON, ошибка валидации, исчерпание retry)
  уходят в топик `KAFKA_DLQ_TOPIC` с полем `original` и причиной ошибки.
- **Retry**: экспоненциальный backoff (`RETRY_INITIAL → x2 → … → RETRY_MAX`).

## Схема ClickHouse

```sql
CREATE TABLE analytics.events
(
    transaction_id  String,
    user_id         Int64,
    type            String,
    status          String,
    from_currency   String,
    to_currency     String,
    amount          Float64,
    event_time      DateTime64(3),
    received_at     DateTime64(3),
    latency_ms      Int64,
    topic           String,
    partition       Int32,
    offset          Int64,
    version         UInt64
)
ENGINE = ReplacingMergeTree(version)
ORDER BY transaction_id
PARTITION BY toYYYYMM(event_time)
TTL toDateTime(event_time) + INTERVAL 90 DAY;
```

- `ReplacingMergeTree(version)` дедуплицирует по `transaction_id`.
- `PARTITION BY toYYYYMM(event_time)` — партиции по месяцам.
- `TTL ... + INTERVAL 90 DAY` — автоудаление старых данных.

## Конфигурация

Все параметры читаются из `config.env` (для локального запуска) или из env
(для Docker). Любой пропущенный ключ — сервис не стартует.

| Переменная              | Пример                    | Описание                             |
|-------------------------|---------------------------|--------------------------------------|
| `KAFKA_BROKERS`         | `kafka:9092`              | адреса брокеров (через запятую)      |
| `KAFKA_TOPIC`           | `large_transfers`         | топик-источник                       |
| `KAFKA_DLQ_TOPIC`       | `large_transfers_dlq`     | топик DLQ                            |
| `KAFKA_GROUP_ID`        | `gw-analytics`            | consumer group                       |
| `CLICKHOUSE_ADDR`       | `clickhouse:9000`         | native-адрес ClickHouse              |
| `CLICKHOUSE_DB`         | `analytics`               | база                                 |
| `CLICKHOUSE_USER`       | `default`                 | пользователь                         |
| `CLICKHOUSE_PASSWORD`   | `""`                      | пароль                               |
| `CLICKHOUSE_TIMEOUT`    | `5s`                      | таймаут подключения                  |
| `BATCH_SIZE`            | `500`                     | размер батча                         |
| `BATCH_TIMEOUT`         | `1s`                      | таймаут флаша                        |
| `QUEUE_SIZE`            | `5000`                    | буфер задач                          |
| `RETRY_MAX_ATTEMPTS`    | `5`                       | число попыток retry                  |
| `RETRY_INITIAL`         | `200ms`                   | начальная пауза backoff              |
| `RETRY_MAX`             | `10s`                     | максимальная пауза                   |
| `HTTP_PORT`             | `8082`                    | порт REST API                        |
| `HTTP_READ_TIMEOUT`     | `10s`                     | таймаут чтения запроса               |
| `HTTP_WRITE_TIMEOUT`    | `10s`                     | таймаут записи ответа                |
| `METRICS_ENABLED`       | `true`                    | включить `/metrics`                  |
| `METRICS_PORT`          | `9092`                    | порт `/metrics`                      |
| `LOG_LEVEL`             | `info`                    | `debug` / `info` / `warn` / `error`  |
| `SHUTDOWN_TIMEOUT`      | `20s`                     | таймаут graceful shutdown            |

## Запуск

### В Docker

```bash
docker compose up -d clickhouse kafka kafka-init gw-analytics
docker compose logs -f gw-analytics
```

### Swagger UI

```bash
make swag              # сгенерировать docs/
make docker-build      # пересобрать образ
make docker-up         # поднять
# открыть http://localhost:8082/swagger/index.html
```

## Проверка работоспособности

```bash
# 1. Все контейнеры подняты
make ps

# 2. Логи сервиса
make logs
# ожидаемо:
#   clickhouse connected  addr=clickhouse:9000 db=analytics
#   kafka ready            brokers=[kafka:9092] topic=large_transfers
#   http api started       port=8082
#   metrics server started port=9092
#   processor started      batch_size=500 batch_timeout=1s queue_size=5000

# 3. API отвечает
curl http://localhost:8082/health
# {"status":"ok"}

# 4. Метрики
curl http://localhost:9092/metrics | grep gw_analytics
```

### E2E: wallet → Kafka → analytics → ClickHouse

```bash
# 1. Крупный депозит через wallet (порог LARGE_TRANSFER_THRESHOLD = 30000)
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"user1","password":"pass123"}' | jq -r .token)

curl -X POST http://localhost:8080/api/v1/wallet/deposit \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"amount":50000,"currency":"USD"}'

# 2. Проверить ClickHouse
docker compose exec clickhouse clickhouse-client \
  --query "SELECT transaction_id, type, amount FROM analytics.events FINAL ORDER BY event_time DESC LIMIT 5"

# 3. Проверить API
curl "http://localhost:8082/api/v1/analytics/events?period=1m"
```

### Проверка идемпотентности

```bash
# Отправить одно событие дважды
docker compose exec kafka kafka-console-producer \
  --bootstrap-server kafka:9092 --topic large_transfers

# Вставить дважды:
# {"transaction_id":"test-idem-001","user_id":1,"type":"deposit","to_currency":"USD","amount":50000,"timestamp":"2026-10-05T03:00:00Z"}

# В ClickHouse должно быть 1, а не 2
docker compose exec clickhouse clickhouse-client \
  --query "SELECT count() FROM analytics.events FINAL WHERE transaction_id='test-idem-001'"
```

### Проверка DLQ

```bash
# Отправить битое сообщение
docker compose exec kafka kafka-console-producer \
  --bootstrap-server kafka:9092 --topic large_transfers
# Вставить: not a json

# В логах будет "message sent to dlq"
make logs

# В DLQ-топике появится запись
docker compose exec kafka kafka-console-consumer \
  --bootstrap-server kafka:9092 --topic large_transfers_dlq \
  --from-beginning --max-messages 1
```

