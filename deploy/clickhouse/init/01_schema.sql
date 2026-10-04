CREATE DATABASE IF NOT EXISTS oktopus;

CREATE TABLE IF NOT EXISTS oktopus.proxy_access_log
(
    id UUID,
    instance_id UUID,
    created_at DateTime64(3, 'UTC'),
    source_address String,
    destination_address String,
    user_name Nullable(String),
    decide_duration_us Int64,
    full_url String,
    action UInt8,
    inspect_rule_id Nullable(UUID),
    denied_by Nullable(String),
    decision_rule_ref String,
    search_engine String,
    search_query String,
    inspect_error String
)
ENGINE = MergeTree
PARTITION BY toYYYYMM(created_at)
ORDER BY (created_at, id);

CREATE TABLE IF NOT EXISTS oktopus.proxy_access_log_inspect_kv
(
    log_id UUID,
    created_at DateTime64(3, 'UTC'),
    inspect_rule_id UUID,
    field_key LowCardinality(String),
    field_value String
)
ENGINE = MergeTree
PARTITION BY toYYYYMM(created_at)
ORDER BY (created_at, field_key, inspect_rule_id, log_id);
