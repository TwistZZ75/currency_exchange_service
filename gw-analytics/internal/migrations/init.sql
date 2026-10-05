CREATE DATABASE IF NOT EXISTS analytics;

CREATE TABLE IF NOT EXISTS analytics.events
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

ALTER TABLE analytics.events
    ADD INDEX IF NOT EXISTS idx_event_time event_time TYPE minmax GRANULARITY 1;