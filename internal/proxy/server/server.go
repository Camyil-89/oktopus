package server

import (
	"context"
	"fmt"
	"log"
	stdhttp "net/http"
	"time"

	"github.com/google/uuid"

	"oktopus/internal/pki"
	"oktopus/internal/proxy/instancectx"
	"oktopus/internal/proxy/accesslog"
	"oktopus/internal/proxy/acl"
	"oktopus/internal/proxy/auth"
	"oktopus/internal/proxy/config"
	"oktopus/internal/proxy/hooks"
	"oktopus/internal/proxy/inspect"
	"oktopus/internal/proxy/metrics"
	"oktopus/internal/proxy/observe"
	proxyhttp "oktopus/internal/proxy/http"
	proxyhttps "oktopus/internal/proxy/https"
)

// Server маршрутизирует http:// и CONNECT (https) в соответствующие пакеты.
type Server struct {
	cfg        config.Config
	instanceID uuid.UUID
	aclSource string
	log       *log.Logger
	auth      *auth.Gate
	engine    *acl.Engine
	http      *proxyhttp.Forwarder
	connect   proxyhttps.ConnectHandler
}

// New создаёт сервер. hooks может быть nil. engine — опубликованный ACL из БД.
// accessRec логирует каждое ACL-решение; nil — без записи.
func New(cfg config.Config, instanceID uuid.UUID, engine *acl.Engine, inspectRunner *inspect.Runner, h *hooks.Hooks, accessRec accesslog.Recorder, logger *log.Logger, authCache *auth.AuthCache) (*Server, error) {
	cfg = cfg.WithDefaults()
	if logger == nil {
		logger = log.Default()
	}

	if engine == nil {
		engine = acl.EmptyEngine()
	}
	h = mergeHooks(h, engine, inspectRunner, accessRec, cfg)
	aclSource := "database"

	client := &stdhttp.Client{
		Transport: proxyhttp.NewOutboundTransport(),
		Timeout:   cfg.OutboundTimeout,
		CheckRedirect: func(req *stdhttp.Request, via []*stdhttp.Request) error {
			return stdhttp.ErrUseLastResponse
		},
	}

	var connect proxyhttps.ConnectHandler
	ca, caErr := pki.LoadAuthority(cfg.CACertPath, cfg.CAKeyPath)
	switch cfg.Connect {
	case config.ConnectTunnel:
		if caErr != nil {
			logger.Printf("tunnel: CA not loaded (%v); PORT deny will use CONNECT 403 without in-tab Forbidden page", caErr)
		} else {
			ca.LogSummary(logger)
		}
		connect = &proxyhttps.Tunnel{Hooks: h, CA: ca, RateLimit: engine}
	case config.ConnectMITM:
		if caErr != nil {
			return nil, fmt.Errorf("mitm requires CA (%s, %s): %w", cfg.CACertPath, cfg.CAKeyPath, caErr)
		}
		ca.LogSummary(logger)
		connect = &proxyhttps.MITM{CA: ca, Hooks: h, AccessLog: accessRec, RateLimit: engine}
	default:
		return nil, fmt.Errorf("unknown connect mode: %q", cfg.Connect)
	}

	gate, err := auth.NewGateFromConfig(cfg.Auth, logger, authCache)
	if err != nil {
		return nil, err
	}
	if gate != nil && accessRec != nil {
		gate.LogAuthFail = func(ctx context.Context, r *stdhttp.Request, spend time.Duration) {
			ctx = observe.WithRemoteAddr(ctx, r.RemoteAddr)
			accessRec.Record(accesslog.AuthFailEntry(ctx, r, spend))
		}
	}
	return &Server{
		cfg:        cfg,
		instanceID: instanceID,
		aclSource: aclSource,
		log:       logger,
		auth:      gate,
		engine:    engine,
		http: &proxyhttp.Forwarder{
			Client:    client,
			Hooks:     h,
			RateLimit: engine,
		},
		connect: connect,
	}, nil
}

// ListenAndServe блокируется до ошибки listener.
func (s *Server) ListenAndServe() error {
	s.log.Printf("listening on %s (connect=%s)", s.cfg.Listen, s.cfg.Connect)
	if s.cfg.Connect == config.ConnectMITM {
		s.log.Printf("MITM CA: %s", s.cfg.CACertPath)
	}
	if s.auth != nil {
		s.log.Printf("proxy auth: Basic realm=%q backend=%s", s.cfg.Auth.Realm, authBackendLabel(s.cfg.Auth))
		if s.auth.Cache != nil {
			s.log.Printf("proxy auth cache: ttl=%s", s.auth.Cache.TTL())
		}
	}
	if s.aclSource != "" {
		s.log.Printf("acl: %s", s.aclSource)
	}
	srv := &stdhttp.Server{
		Addr:              s.cfg.Listen,
		Handler:           s,
		ReadHeaderTimeout: s.cfg.ReadHeaderTimeout,
		IdleTimeout:       s.cfg.IdleTimeout,
	}
	return srv.ListenAndServe()
}

func authBackendLabel(a config.AuthConfig) string {
	if a.Backend == "" {
		return "static"
	}
	return a.Backend
}

func (s *Server) ServeHTTP(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	ctx, ok := s.auth.Require(w, r)
	if !ok {
		return
	}
	if s.instanceID != uuid.Nil {
		ctx = instancectx.WithID(ctx, s.instanceID)
	}
	r = r.WithContext(ctx)

	if r.Method == stdhttp.MethodConnect {
		// После Hijack контекст CONNECT может отмениться — MITM живёт на своём соединении.
		connectCtx := context.WithoutCancel(ctx)
		connectCtx = observe.WithRemoteAddr(connectCtx, r.RemoteAddr)
		connectCtx = observe.WithDeferPortACLAtConnect(connectCtx)
		if err := s.connect.Serve(connectCtx, w, r); err != nil {
			s.log.Printf("CONNECT %s: %v", r.Host, err)
		}
		return
	}

	if r.URL == nil || !r.URL.IsAbs() {
		stdhttp.Error(w, "this proxy requires an absolute URL (e.g. GET http://example.com/)", stdhttp.StatusBadRequest)
		return
	}

	if err := s.http.Serve(ctx, w, r); err != nil {
		s.log.Printf("HTTP %s %s: %v", r.Method, r.URL, err)
		stdhttp.Error(w, err.Error(), stdhttp.StatusBadGateway)
	}
}

func mergeHooks(user *hooks.Hooks, engine *acl.Engine, inspectRunner *inspect.Runner, accessRec accesslog.Recorder, cfg config.Config) *hooks.Hooks {
	if engine == nil {
		engine = acl.EmptyEngine()
	}
	aclH := engine.Hooks(accessRec)
	inspectEnabled := cfg.Connect == config.ConnectMITM
	inspectMW := inspect.HTTPMiddleware(inspectRunner, accessRec, inspectEnabled)
	// Сначала ACL (быстро), затем Lua-инспекция только если ACL allow.
	chainReq := chainACLThenInspectHTTP(aclH.OnHTTPRequest, inspectMW, userOnHTTP(user))
	return &hooks.Hooks{
		OnConnect:      hooks.ChainConnect(aclH.OnConnect, userOnConnect(user)),
		OnHTTPRequest:  chainReq,
		OnHTTPResponse: hooks.ChainHTTPResponse(aclH.OnHTTPResponse, userOnHTTPResponse(user)),
	}
}

func userOnHTTP(user *hooks.Hooks) hooks.HTTPRequestMiddleware {
	if user == nil {
		return nil
	}
	return user.OnHTTPRequest
}

func userOnConnect(user *hooks.Hooks) hooks.ConnectMiddleware {
	if user == nil {
		return nil
	}
	return user.OnConnect
}

func userOnHTTPResponse(user *hooks.Hooks) hooks.HTTPResponseMiddleware {
	if user == nil {
		return nil
	}
	return user.OnHTTPResponse
}

// chainACLThenInspectHTTP: ACL → инспекция → пользовательские hooks.
func chainACLThenInspectHTTP(aclMW, inspectMW, userMW hooks.HTTPRequestMiddleware) hooks.HTTPRequestMiddleware {
	return func(ctx context.Context, req *stdhttp.Request) hooks.Decision {
		if aclMW != nil {
			d := aclMW(ctx, req)
			if d.Handled() {
				return d
			}
		}
		if inspectMW != nil {
			d := inspectMW(ctx, req)
			if d.Handled() {
				return d
			}
		} else if req != nil {
			metrics.ObserveHTTPPolicyFromACLContext(req.Context())
		}
		if userMW != nil {
			return userMW(ctx, req)
		}
		return hooks.AllowDecision()
	}
}

// Run — точка входа из cmd (блокируется до ошибки listener).
func Run(cfg config.Config, engine *acl.Engine, h *hooks.Hooks, accessRec accesslog.Recorder, logger *log.Logger) error {
	return RunContext(context.Background(), cfg, engine, h, accessRec, logger)
}

// RunContext запускает прокси до отмены ctx или ошибки listener.
func RunContext(ctx context.Context, cfg config.Config, engine *acl.Engine, h *hooks.Hooks, accessRec accesslog.Recorder, logger *log.Logger) error {
	srv, err := New(cfg, uuid.Nil, engine, nil, h, accessRec, logger, nil)
	if err != nil {
		return err
	}
	return srv.ListenAndServeContext(ctx)
}

// ListenAndServeContext слушает до отмены ctx или ошибки listener.
func (s *Server) ListenAndServeContext(ctx context.Context) error {
	s.log.Printf("listening on %s (connect=%s)", s.cfg.Listen, s.cfg.Connect)
	if s.cfg.Connect == config.ConnectMITM {
		s.log.Printf("MITM CA: %s", s.cfg.CACertPath)
	}
	if s.auth != nil {
		s.log.Printf("proxy auth: Basic realm=%q backend=%s", s.cfg.Auth.Realm, authBackendLabel(s.cfg.Auth))
		if s.auth.Cache != nil {
			s.log.Printf("proxy auth cache: ttl=%s", s.auth.Cache.TTL())
		}
	}
	if s.aclSource != "" {
		s.log.Printf("acl: %s", s.aclSource)
	}
	srv := &stdhttp.Server{
		Addr:              s.cfg.Listen,
		Handler:           s,
		ReadHeaderTimeout: s.cfg.ReadHeaderTimeout,
		IdleTimeout:       s.cfg.IdleTimeout,
	}

	go func() {
		<-ctx.Done()
		_ = srv.Shutdown(context.Background())
	}()

	err := srv.ListenAndServe()
	if err != nil && err != stdhttp.ErrServerClosed {
		return err
	}
	return nil
}
