package db

import (
	"os"
)

const (
	envDatabaseURL       = "DATABASE_URL"
	envClickHouseAddr    = "CLICKHOUSE_ADDR"
	envClickHouseDB      = "CLICKHOUSE_DATABASE"
	envClickHouseUser    = "CLICKHOUSE_USER"
	envClickHousePass    = "CLICKHOUSE_PASSWORD"
	envAdminUsername     = "OKTOPUS_ADMIN_USERNAME"
	envAdminPassword     = "OKTOPUS_ADMIN_PASSWORD"
	defaultDatabaseURL   = "postgres://oktopus:oktopus@127.0.0.1:5434/oktopus?sslmode=disable"
	defaultClickHouseAddr = "127.0.0.1:9054"
	defaultClickHouseDB  = "oktopus"
	defaultClickHouseUser = "oktopus"
	defaultClickHousePass = "oktopus"
	defaultAdminUser     = "admin"
	defaultAdminPassword = "admin"
)

// ClickHouseConfig — журнал доступа (proxy access log).
type ClickHouseConfig struct {
	Addr     string
	Database string
	Username string
	Password string
}

// Config подключения и начального администратора.
type Config struct {
	DatabaseURL   string
	ClickHouse    ClickHouseConfig
	AdminUsername string
	AdminPassword string
}

// ConfigFromEnv читает DATABASE_URL и учётку admin из окружения.
func ConfigFromEnv() Config {
	return Config{
		DatabaseURL: envOr(envDatabaseURL, defaultDatabaseURL),
		ClickHouse: ClickHouseConfig{
			Addr:     envOr(envClickHouseAddr, defaultClickHouseAddr),
			Database: envOr(envClickHouseDB, defaultClickHouseDB),
			Username: envOr(envClickHouseUser, defaultClickHouseUser),
			Password: envOr(envClickHousePass, defaultClickHousePass),
		},
		AdminUsername: envOr(envAdminUsername, defaultAdminUser),
		AdminPassword: envOr(envAdminPassword, defaultAdminPassword),
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
