package clickhouse

import (
	"context"
	"fmt"

	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// Config — параметры подключения ClickHouse.
type Config struct {
	Addr     string
	Database string
	Username string
	Password string
}

// Open подключается к ClickHouse (native protocol).
func Open(ctx context.Context, cfg Config) (driver.Conn, error) {
	conn, err := ch.Open(&ch.Options{
		Addr: []string{cfg.Addr},
		Auth: ch.Auth{
			Database: cfg.Database,
			Username: cfg.Username,
			Password: cfg.Password,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("clickhouse open: %w", err)
	}
	if err := conn.Ping(ctx); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("clickhouse ping: %w", err)
	}
	if err := EnsureSchema(ctx, conn); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return conn, nil
}

// EnsureSchema создаёт таблицы журнала, если их ещё нет.
func EnsureSchema(ctx context.Context, conn driver.Conn) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS proxy_access_log
(
    id UUID,
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
ORDER BY (created_at, id)`,
		`CREATE TABLE IF NOT EXISTS proxy_access_log_inspect_kv
(
    log_id UUID,
    created_at DateTime64(3, 'UTC'),
    inspect_rule_id UUID,
    field_key LowCardinality(String),
    field_value String
)
ENGINE = MergeTree
PARTITION BY toYYYYMM(created_at)
ORDER BY (created_at, field_key, inspect_rule_id, log_id)`,
	}
	for _, sql := range stmts {
		if err := conn.Exec(ctx, sql); err != nil {
			return fmt.Errorf("clickhouse schema: %w", err)
		}
	}
	return nil
}
