package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	apiconfig "oktopus/internal/api/config"
	"oktopus/internal/api"
	"oktopus/internal/db"
	"oktopus/internal/proxy"
	"oktopus/internal/proxy/accesslog"
	"oktopus/internal/proxy/forbidden"
	"oktopus/internal/proxy/gateway"
	"oktopus/internal/startup"
)

func runServe(args []string) int {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	migrations := fs.String("migrations", "db/migrations", "каталог goose-миграций")
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(args); err != nil {
		return 2
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	apiCfg, err := apiconfig.Load()
	if err != nil {
		log.Printf("serve: api config: %v", err)
		return 1
	}
	dbCfg := db.ConfigFromEnv()

	startup.MarkProcessStarted(time.Now())

	openCtx, openCancel := context.WithTimeout(context.Background(), 2*time.Minute)
	rt, err := db.Open(openCtx, dbCfg, *migrations)
	openCancel()
	if err != nil {
		log.Printf("serve: db: %v", err)
		return 1
	}
	defer rt.Pool.Close()

	logger := log.New(os.Stdout, "", log.LstdFlags)
	if err := forbidden.Reload(); err != nil {
		logger.Printf("serve: forbidden page: %v", err)
	}
	if err := gateway.Reload(); err != nil {
		logger.Printf("serve: gateway page: %v", err)
	}
	hooks := proxy.DefaultLogHooks(logger)
	mgr := proxy.NewManager(hooks, logger)
	accessBatcher := accesslog.NewBatcher(func(flushCtx context.Context, batch []accesslog.Entry) error {
		return rt.ProxyAccessLog.FlushAccessLog(flushCtx, batch)
	}, accesslog.BatcherConfig{
		OnFlushError: func(err error) { logger.Printf("access log flush: %v", err) },
	})
	defer accessBatcher.Close()
	mgr.SetAccessRecorder(accessBatcher)
	go rt.ProxyAccessLog.RunRetentionLoop(ctx, 0)
	rt.ProxySettings.BindApplier(mgr)
	rt.ProxySettings.BindInspect(rt.ProxyInspect)
	rt.ProxyACL.BindReloader(&proxyHotReloader{settings: rt.ProxySettings, mgr: mgr})
	rt.ProxyInspect.BindReloader(&proxyHotReloader{settings: rt.ProxySettings, mgr: mgr})

	loadCtx, loadCancel := context.WithTimeout(context.Background(), 30*time.Second)
	proxySettings, err := rt.ProxySettings.Get(loadCtx)
	proxyRT, errRT := rt.ProxySettings.LoadProxyRuntime(loadCtx)
	loadCancel()
	if err != nil {
		log.Printf("serve: proxy settings: %v", err)
		return 1
	}
	if errRT != nil {
		log.Printf("serve: proxy runtime: %v", errRT)
		return 1
	}

	errCh := make(chan error, 2)
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		srv := api.NewServer(apiCfg, rt, mgr)
		if err := srv.ListenAndServe(ctx); err != nil {
			errCh <- err
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if !proxySettings.ProxyEnabled {
			startup.AllowRemoteListPollStartup()
			mgr.Stop()
			<-ctx.Done()
			return
		}
		if err := mgr.RunContext(ctx, proxyRT.Config, proxyRT.ACLEngine, proxyRT.InspectRunner); err != nil {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		cancel()
		wg.Wait()
		if err != nil && !errors.Is(err, context.Canceled) {
			log.Printf("serve: %v", err)
			return 1
		}
	case <-ctx.Done():
		wg.Wait()
	}

	return 0
}
