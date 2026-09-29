package main

import (
	"log"
	"os"

	"oktopus/internal/proxy"
	"oktopus/internal/proxy/forbidden"
	"oktopus/internal/proxy/gateway"
)

func runProxy(args []string) int {
	p, code := parseProxyFlags("proxy", args)
	if code != 0 {
		return code
	}

	cfg, code := proxyConfigFromCLI(p)
	if code != 0 {
		return code
	}

	logger := log.New(os.Stdout, "", log.LstdFlags)
	if err := forbidden.Reload(); err != nil {
		logger.Printf("proxy: forbidden page: %v", err)
	}
	if err := gateway.Reload(); err != nil {
		logger.Printf("proxy: gateway page: %v", err)
	}

	if err := proxy.Run(cfg, proxy.DefaultLogHooks(logger), logger); err != nil {
		log.Printf("proxy: %v", err)
		return 1
	}
	return 0
}
