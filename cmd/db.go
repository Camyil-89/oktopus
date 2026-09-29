package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"oktopus/internal/db"
)

func runDB(args []string) int {
	if len(args) == 0 {
		printDBUsage()
		return 2
	}

	switch args[0] {
	case "up", "migrate":
		return runDBUp(args[1:])
	case "help", "-h", "--help":
		printDBUsage()
		return 0
	default:
		log.Printf("db: неизвестная подкоманда: %s\n", args[0])
		printDBUsage()
		return 2
	}
}

func runDBUp(args []string) int {
	fs := flag.NewFlagSet("db up", flag.ExitOnError)
	migrations := fs.String("migrations", "db/migrations", "каталог goose-миграций")
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(args); err != nil {
		return 2
	}

	cfg := db.ConfigFromEnv()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	rt, err := db.Open(ctx, cfg, *migrations)
	if err != nil {
		log.Printf("db: %v", err)
		return 1
	}
	rt.Pool.Close()
	log.Printf("db: миграции применены, admin «%s» проверен", cfg.AdminUsername)
	return 0
}

func printDBUsage() {
	fmt.Fprintf(os.Stderr, `oktopus db — PostgreSQL (goose + bootstrap admin)

Подкоманды:
  up, migrate   применить миграции и проверить admin (env: DATABASE_URL, OKTOPUS_ADMIN_*)

Переменные окружения:
  DATABASE_URL              по умолчанию postgres://oktopus:oktopus@127.0.0.1:5434/oktopus?sslmode=disable
  OKTOPUS_ADMIN_USERNAME    по умолчанию admin
  OKTOPUS_ADMIN_PASSWORD    по умолчанию admin

`)
}
