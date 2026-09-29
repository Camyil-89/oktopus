// Пакет proxy — фасад для cmd. Реализация разбита по каталогам:
//
//   config/   — настройки и режим CONNECT
//   hooks/    — колбэки наблюдения
//   http/     — cleartext http:// (абсолютный URL)
//   https/    — CONNECT: tunnel | mitm
//   server/   — ListenAndServe и маршрутизация
//   observe/  — готовые хуки (лог в stdout)
package proxy

import (
	"context"
	"log"

	"oktopus/internal/proxy/config"
	"oktopus/internal/proxy/hooks"
	"oktopus/internal/proxy/observe"
	"oktopus/internal/proxy/server"
)

type Config = config.Config
type AuthConfig = config.AuthConfig
type LDAPConfig = config.LDAPConfig
type ConnectMode = config.ConnectMode
type Hooks = hooks.Hooks

const (
	ConnectTunnel = config.ConnectTunnel
	ConnectMITM   = config.ConnectMITM
)

func DefaultLogHooks(logger *log.Logger) *Hooks {
	return observe.LogHooks(logger)
}

func Run(cfg Config, h *Hooks, logger *log.Logger) error {
	return server.Run(cfg, nil, h, nil, logger)
}

func RunContext(ctx context.Context, cfg Config, h *Hooks, logger *log.Logger) error {
	return server.RunContext(ctx, cfg, nil, h, nil, logger)
}

type Manager = server.Manager

func NewManager(h *Hooks, logger *log.Logger) *Manager {
	return server.NewManager(h, logger)
}
