package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"os"
	"os/signal"
	"runtime/debug"
	"sync"
	"syscall"
	"time"

	apiconfig "oktopus/internal/api/config"
	"oktopus/internal/api"
	"oktopus/internal/db"
	proxyinstancesservice "oktopus/internal/db/proxyinstances/service"
	proxyserver "oktopus/internal/proxy/server"
	"oktopus/internal/proxy"
	"oktopus/internal/proxy/accesslog"
	"oktopus/internal/proxy/forbidden"
	"oktopus/internal/proxy/gateway"
	"oktopus/internal/startup"
)

const panicRecoveryDelay = 5 * time.Second

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
	fleet := proxy.NewFleet(hooks, logger)
	accessBatcher := accesslog.NewBatcher(func(flushCtx context.Context, batch []accesslog.Entry) error {
		return rt.ProxyAccessLog.FlushAccessLog(flushCtx, batch)
	}, accesslog.BatcherConfig{
		OnFlushError: func(err error) { logger.Printf("access log flush: %v", err) },
	})
	defer accessBatcher.Close()
	fleet.SetAccessRecorder(accessBatcher)
	go rt.ProxyAccessLog.RunRetentionLoop(ctx, 0)
	rt.ProxyInstances.BindApplier(fleet)
	rt.ProxyInstances.BindInspect(rt.ProxyInspect)
	rt.ProxyACL.BindReloader(&proxyHotReloader{instances: rt.ProxyInstances})
	rt.ProxyInspect.BindReloader(&proxyHotReloader{instances: rt.ProxyInstances})

	loadCtx, loadCancel := context.WithTimeout(context.Background(), 30*time.Second)
	_, errRT := rt.ProxyInstances.LoadFleetRuntime(loadCtx)
	loadCancel()
	if errRT != nil {
		log.Printf("serve: proxy runtime: %v", errRT)
		return 1
	}

	errCh := make(chan error, 2)
	var wg sync.WaitGroup
	apiReady := make(chan struct{})
	var apiReadyOnce sync.Once
	signalAPIReady := func() {
		apiReadyOnce.Do(func() { close(apiReady) })
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		runServiceLoop(ctx, logger, "api", errCh, nil, func() error {
			srv := api.NewServer(apiCfg, rt, fleet)
			return srv.ListenAndServe(ctx, signalAPIReady)
		})
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		select {
		case <-apiReady:
		case <-ctx.Done():
			return
		}
		if ctx.Err() != nil {
			return
		}
		runServiceLoop(ctx, logger, "proxy", errCh, func() { fleet.Stop() }, func() error {
			return fleet.RunContextWithLoader(ctx, loadProxyFleet(rt.ProxyInstances))
		})
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

func runServiceLoop(ctx context.Context, logger *log.Logger, name string, errCh chan error, onPanic func(), fn func() error) {
	for {
		if ctx.Err() != nil {
			return
		}
		var err error
		panicked := false
		func() {
			defer func() {
				if rec := recover(); rec != nil {
					logger.Printf("serve: %s: panic: %v\n%s", name, rec, debug.Stack())
					if onPanic != nil {
						onPanic()
					}
					panicked = true
				}
			}()
			err = fn()
		}()
		if ctx.Err() != nil {
			return
		}
		if panicked {
			logger.Printf("serve: %s: перезапуск после паники через %s", name, panicRecoveryDelay)
			if !waitFor(ctx, panicRecoveryDelay) {
				return
			}
			continue
		}
		if err != nil {
			errCh <- err
			return
		}
		return
	}
}

func loadProxyFleet(instances *proxyinstancesservice.Service) proxyserver.FleetRuntimeLoader {
	return func(ctx context.Context) ([]proxyinstancesservice.InstanceRuntime, error) {
		runtimes, err := instances.LoadFleetRuntime(ctx)
		if err != nil {
			return nil, err
		}
		if len(runtimes) == 0 {
			startup.AllowRemoteListPollStartup()
			return nil, proxyserver.ErrRuntimeDeferred
		}
		return runtimes, nil
	}
}

func waitFor(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

type proxyHotReloader struct {
	instances *proxyinstancesservice.Service
}

func (r *proxyHotReloader) ReloadProxy(ctx context.Context) error {
	return r.instances.ReloadFleet(ctx)
}
