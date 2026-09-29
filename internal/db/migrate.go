package db

import (
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

// RunMigrations применяет SQL-миграции из migrationsDir (goose).
func RunMigrations(databaseURL, migrationsDir string) error {
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("goose dialect: %w", err)
	}

	sqlDB, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return fmt.Errorf("open db for migrations: %w", err)
	}
	defer sqlDB.Close()

	if err := goose.Up(sqlDB, migrationsDir); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}
	return nil
}
