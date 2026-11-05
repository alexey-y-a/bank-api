-- Отдельная БД аналитики: события из Kafka
CREATE DATABASE IF NOT EXISTS analytics;

CREATE TABLE IF NOT EXISTS analytics.transactions_log
(
    event_id        String,
    event_type      LowCardinality(String),
    from_account_id Int64,
    to_account_id   Int64,
    amount          Int64,
    created_at      DateTime64(3, 'UTC'),
    inserted_at     DateTime64(3, 'UTC') DEFAULT now64(3)
)

ENGINE = ReplacingMergeTree
ORDER BY event_id;