# gw-notifier

Kafka-consumer крупных денежных переводов. Читает события из топика `large_transfers`
(источник — `gw-currency-wallet`) и сохраняет их в MongoDB. Публикует метрики Prometheus.

## Что делает

- **Consumes** события из Kafka-топика `large_transfers`.
- **Сохраняет** их в MongoDB с идемпотентностью по `transaction_id`.
- **Обрабатывает батчами**: копит до `BATCH_SIZE` или до `BATCH_TIMEOUT`, пишет через `BulkWrite`.
- **At-least-once**: offset коммитится **после** успешной записи в Mongo.
- **DLQ**: битые сообщения и исчерпавшие retry батчи уходят в топик `KAFKA_DLQ_TOPIC`.
- **Retry**: экспоненциальный backoff при ошибках Mongo.
- **Метрики Prometheus** на `/metrics`.

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
         └──┬────────────┬──────┘
            │            │
            ▼            ▼
   ┌────────────────┐  ┌──────────────────┐
   │  gw-notifier   │  │   gw-analytics   │
   │  (этот сервис) │  │   → ClickHouse   │
   └───────┬────────┘  └──────────────────┘
           │
           ▼
   ┌────────────────┐
   │    MongoDB     │
   │  (notification)│
   └────────────────┘
```

## Формат события из Kafka

```json
{
  "transaction_id": "550e8400-e29b-41d4-a716-446655440000",
  "user_id": 1,
  "type": "deposit",
  "status": "success",
  "from_currency": "",
  "to_currency": "USD",
  "amount": 50000,
  "timestamp": "2026-10-05T02:47:57Z"
}
```

Валидация: `transaction_id` не пустой, `user_id > 0`, `amount > 0`. Всё остальное
опционально — служебные поля (`received_at`, `topic`, `partition`, `offset`)
заполняются consumer'ом при записи.

## Формат документа в MongoDB

Коллекция `notification.large_transfers`:

```json
{
  "_id": "ObjectId('...')",
  "transaction_id": "550e8400-e29b-41d4-a716-446655440000",
  "user_id": 1,
  "type": "deposit",
  "status": "success",
  "from_currency": "",
  "to_currency": "USD",
  "amount": 50000,
  "timestamp": "2026-10-05T02:47:57Z",
  "received_at": "2026-10-05T02:47:58.321Z",
  "topic": "large_transfers",
  "partition": 0,
  "offset": 42
}
```

На `transaction_id` создан **unique index** — основа идемпотентности.

## Гарантии и особенности

- **At-least-once**: offset коммитится только после `SaveBatch` в Mongo.
  При падении до коммита Kafka передоставит сообщения.
- **Идемпотентность**: `UpdateOne` с `$setOnInsert` + `upsert=true` + unique index
  на `transaction_id`. Повторная обработка не создаёт дубликат.
- **Batching**: события копятся и пишутся через `BulkWrite` с `ordered=false` —
  ошибка на одном документе не блокирует остальные.
- **Retry**: экспоненциальный backoff (`RETRY_INITIAL → x2 → … → RETRY_MAX`).
  При исчерпании — батч целиком уходит в DLQ.
- **DLQ**: в топик `large_transfers_dlq` пишется JSON с полями `original`, `error`,
  `topic`, `partition`, `offset`, `failed_at`.

## Метрики Prometheus

Отдаются на `METRICS_PORT` (по умолчанию `9090`) по пути `/metrics`.

| Метрика                                            | Тип       | Описание                                       |
|----------------------------------------------------|-----------|------------------------------------------------|
| `gw_notification_events_processed_total{status}`   | counter   | обработано событий (`saved` / `duplicate` / `failed`) |
| `gw_notification_dlq_events_total`                 | counter   | ушло в DLQ                                     |
| `gw_notification_mongo_errors_total`               | counter   | bulk write исчерпал retry                      |
| `gw_notification_retry_attempts`                   | histogram | распределение числа retry                      |
| `gw_notification_batch_size`                       | histogram | размер батча при записи в Mongo                |
| `gw_notification_mongo_write_duration_seconds`     | histogram | длительность bulk write (включая retry)        |
| `gw_notification_processing_duration_seconds`      | histogram | полное время обработки батча                   |

## Конфигурация

Все параметры читаются из `config.env` (локально) или из env (Docker). Любой
пропущенный ключ — сервис не стартует.

| Переменная         | Пример                    | Описание                             |
|--------------------|---------------------------|--------------------------------------|
| `KAFKA_BROKERS`    | `kafka:9092`              | адреса брокеров (через запятую)      |
| `KAFKA_TOPIC`      | `large_transfers`         | топик-источник                       |
| `KAFKA_DLQ_TOPIC`  | `large_transfers_dlq`     | топик DLQ                            |
| `KAFKA_GROUP_ID`   | `gw-notifier`             | consumer group                       |
| `MONGO_URI`        | `mongodb://mongo:27017`   | адрес MongoDB                        |
| `MONGO_DB`         | `notification`            | база                                 |
| `MONGO_COLLECTION` | `large_transfers`         | коллекция                            |
| `MONGO_TIMEOUT`    | `5s`                      | таймаут операций с Mongo             |
| `BATCH_SIZE`       | `100`                     | размер батча                         |
| `BATCH_TIMEOUT`    | `1s`                      | таймаут флаша                        |
| `QUEUE_SIZE`       | `2000`                    | буфер задач между reader и batcher   |
| `RETRY_MAX_ATTEMPTS` | `5`                     | число попыток retry                  |
| `RETRY_INITIAL`    | `100ms`                   | начальная пауза backoff              |
| `RETRY_MAX`        | `5s`                      | максимальная пауза                   |
| `METRICS_ENABLED`  | `true`                    | включить `/metrics`                  |
| `METRICS_PORT`     | `9090`                    | порт `/metrics`                      |
| `LOG_LEVEL`        | `info`                    | `debug` / `info` / `warn` / `error`  |
| `SHUTDOWN_TIMEOUT` | `15s`                     | таймаут graceful shutdown            |

## Запуск

### В Docker

```bash
docker compose up -d mongo kafka kafka-init gw-notifier
docker compose logs -f gw-notifier
```

## Проверка работоспособности

```bash
# 1. Все контейнеры подняты
make ps

# 2. Логи сервиса
make logs
# ожидаемо:
#   mongo connected  db=notification collection=large_transfers
#   kafka ready      brokers=[kafka:9092] topic=large_transfers dlq_topic=large_transfers_dlq group=gw-notifier
#   metrics server started  port=9090
#   processor started       batch_size=100 batch_timeout=1s queue_size=2000

# 3. Метрики
curl http://localhost:9090/metrics | grep gw_notification
```

### E2E: wallet → Kafka → notifier → MongoDB

```bash
# 1. Крупный депозит через wallet (порог LARGE_TRANSFER_THRESHOLD = 30000)
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"user1","password":"pass123"}' | jq -r .token)

curl -X POST http://localhost:8080/api/v1/wallet/deposit \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"amount":50000,"currency":"USD"}'

# 2. Через 1–2 секунды проверить логи notifier
make logs
# ожидаемо:
#   batch saved  count=1 inserted=1 attempts=1

# 3. Проверить MongoDB
docker compose exec mongo mongosh notification --quiet --eval 'db.large_transfers.find().pretty()'
```

Ожидаемо: документ с `transaction_id`, `user_id`, `amount`, `received_at`, `offset`.

### Проверка идемпотентности

```bash
# Отправить одно событие дважды в Kafka
docker compose exec kafka kafka-console-producer \
  --bootstrap-server kafka:9092 --topic large_transfers

# Вставить дважды:
# {"transaction_id":"test-dup-1","user_id":1,"type":"deposit","to_currency":"USD","amount":50000,"timestamp":"2026-10-05T03:00:00Z"}

# В логах:
#   event saved  transaction_id=test-dup-1
#   event already exists, skipped  transaction_id=test-dup-1

# В MongoDB — ровно один документ с этим transaction_id
docker compose exec mongo mongosh notification --quiet --eval \
  'db.large_transfers.countDocuments({transaction_id:"test-dup-1"})'
# 1
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

Формат DLQ-сообщения:

```json
{
  "original": "not a json",
  "error": "unmarshal failed: invalid character 'o' in literal null",
  "topic": "large_transfers",
  "partition": 0,
  "offset": 42,
  "failed_at": "2026-10-05T03:01:00Z"
}
```