package api

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"

	apiconfig "oktopus/internal/api/config"
	authmw "oktopus/internal/api/auth/middleware"
	authview "oktopus/internal/api/auth/view"
	"oktopus/internal/api/platform/cors"
	recovermw "oktopus/internal/api/platform/recover"
	proxyview "oktopus/internal/api/proxy/view"
	usersview "oktopus/internal/api/users/view"
	"oktopus/internal/db"
)

// Server HTTP API и зависимости.
type Server struct {
	cfg     apiconfig.Config
	db      *db.Runtime
	handler http.Handler
}

// NewServer собирает маршруты API.
func NewServer(cfg apiconfig.Config, runtime *db.Runtime, proxyStatus proxyview.StatusProvider) *Server {
	authLoginLockout := &authmw.LoginLockout{}
	authGuard := authmw.NewGuard([]byte(cfg.JWTSecret), runtime.Users, authLoginLockout)
	authH := authview.NewHandler(runtime.Users, authGuard, authLoginLockout, cfg.CookieSecure)
	usersH := usersview.NewHandler(runtime.Users, authGuard)
	proxyH := proxyview.NewHandler(runtime.ProxySettings, authGuard)
	proxyACLH := proxyview.NewACLHandler(runtime.ProxyACL, authGuard)
	proxyInspectH := proxyview.NewInspectHandler(runtime.ProxyInspect, authGuard)
	statusH := proxyview.NewStatusHandler(runtime.ProxySettings, runtime.ProxyACL, proxyStatus, authGuard)
	accessLogH := proxyview.NewAccessLogHandler(runtime.ProxyAccessLog, authGuard)

	mux := http.NewServeMux()
	authview.Register(mux, authH)
	usersview.Register(mux, usersH)
	proxyview.Register(mux, proxyH, proxyACLH, proxyInspectH, statusH, accessLogH)

	var h http.Handler = mux
	h = cors.Middleware(cfg.CORSOrigins)(h)
	h = recovermw.Middleware(log.Default())(h)

	return &Server{cfg: cfg, db: runtime, handler: h}
}

// ListenAndServe запускает HTTP-сервер до отмены контекста.
// ready вызывается один раз после успешного bind (до приёма соединений).
func (s *Server) ListenAndServe(ctx context.Context, ready func()) error {
	ln, err := net.Listen("tcp", s.cfg.Listen)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	if ready != nil {
		ready()
	}

	srv := &http.Server{Handler: s.handler}

	go func() {
		<-ctx.Done()
		_ = srv.Shutdown(context.Background())
	}()

	log.Printf("api: listening on http://%s", ln.Addr().String())
	if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("serve: %w", err)
	}
	return nil
}
