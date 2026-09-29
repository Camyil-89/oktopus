package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	apiconfig "oktopus/internal/api/config"
	"oktopus/internal/api"
	"oktopus/internal/db"
)

func runAPI(args []string) int {
	fs := flag.NewFlagSet("api", flag.ExitOnError)
	migrations := fs.String("migrations", "db/migrations", "каталог goose-миграций")
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(args); err != nil {
		return 2
	}

	cfg, err := apiconfig.Load()
	if err != nil {
		log.Printf("api: config: %v", err)
		return 1
	}
	dbCfg := db.ConfigFromEnv()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	openCtx, openCancel := context.WithTimeout(context.Background(), 2*time.Minute)
	rt, err := db.Open(openCtx, dbCfg, *migrations)
	openCancel()
	if err != nil {
		log.Printf("api: db: %v", err)
		return 1
	}
	defer rt.Pool.Close()

	srv := api.NewServer(cfg, rt, nil)
	if err := srv.ListenAndServe(ctx); err != nil {
		log.Printf("api: %v", err)
		return 1
	}
	return 0
}
