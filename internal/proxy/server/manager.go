package server

import (
	"context"
	"errors"
	"log"
	"net"
	stdhttp "net/http"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	"oktopus/internal/apperr"
	"oktopus/internal/proxy/accesslog"
	"oktopus/internal/proxy/acl"
	"oktopus/internal/proxy/auth"
	"oktopus/internal/proxy/config"
	"oktopus/internal/proxy/hooks"
	"oktopus/internal/proxy/inspect"
	"oktopus/internal/proxy/metrics"
	"oktopus/internal/startup"
)

// Manager запускает прокси и поддерживает hot reload конфигурации.
type Manager struct {
	hooks     *hooks.Hooks
	log       *log.Logger
	accessRec accesslog.Recorder
	instanceID   uuid.UUID
	instanceName string

	mu        sync.Mutex
	listen    string
	httpSrv   *stdhttp.Server
	holder    atomic.Pointer[Server]
	listening                 atomic.Bool
	listenRestartAfterPanic   atomic.Bool
	lastStartErr              atomic.Value // string
	authCache *auth.AuthCache
	conns         activeConns
	accessBatcher *accesslog.Batcher

	appliedCfg     config.Config
	appliedEngine  *acl.Engine
	appliedInspect *inspect.Runner
}

const (
	proxyListenRetryInterval = 10 * time.Second
	panicRecoveryDelay       = 5 * time.Second
)

// ErrRuntimeDeferred — прокси временно не запускается (например выключен); RunContext повторит позже.
var ErrRuntimeDeferred = errors.New("proxy runtime deferred")

// RuntimeLoader возвращает конфиг и зависимости на каждой итерации цикла запуска.
type RuntimeLoader func(ctx context.Context) (config.Config, *acl.Engine, *inspect.Runner, error)

type activeConns struct {
	mu    sync.Mutex
	conns map[net.Conn]struct{}
}

func (a *activeConns) track(conn net.Conn, state stdhttp.ConnState) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.conns == nil {
		a.conns = make(map[net.Conn]struct{})
	}
	switch state {
	case stdhttp.StateNew:
		a.conns[conn] = struct{}{}
	case stdhttp.StateClosed:
		delete(a.conns, conn)
	}
}

func (a *activeConns) count() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return len(a.conns)
}

func (a *activeConns) closeAll() {
	a.mu.Lock()
	defer a.mu.Unlock()
	for c := range a.conns {
		_ = c.Close()
	}
	a.conns = make(map[net.Conn]struct{})
}

// NewManager создаёт менеджер прокси.
func NewManager(h *hooks.Hooks, logger *log.Logger) *Manager {
	return &Manager{
		hooks: h,
		log:   logger,
	}
}

func (m *Manager) SetInstanceID(id uuid.UUID) {
	m.instanceID = id
}

func (m *Manager) SetInstanceName(name string) {
	m.instanceName = name
}

func (m *Manager) instanceLabel() string {
	if m.instanceName != "" {
		return m.instanceName
	}
	if m.instanceID != uuid.Nil {
		return m.instanceID.String()
	}
	return ""
}

func (m *Manager) logf(format string, args ...interface{}) {
	if m.log == nil {
		return
	}
	if label := m.instanceLabel(); label != "" {
		m.log.Printf(label+": "+format, args...)
		return
	}
	m.log.Printf(format, args...)
}

func (m *Manager) instanceLogger() *log.Logger {
	if m.log == nil {
		return log.Default()
	}
	label := m.instanceLabel()
	if label == "" {
		return m.log
	}
	return log.New(m.log.Writer(), label+": ", m.log.Flags())
}

// SetAccessRecorder задаёт логгер ACL-решений (например accesslog.Batcher).
func (m *Manager) SetAccessRecorder(rec accesslog.Recorder) {
	m.accessRec = rec
	if b, ok := rec.(*accesslog.Batcher); ok {
		m.accessBatcher = b
	}
}

type reloadableHandler struct {
	m *Manager
}

func (h reloadableHandler) ServeHTTP(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	defer func() {
		if rec := recover(); rec != nil {
			h.m.recoverHandlerPanic(rec)
			stdhttp.Error(w, "internal error", stdhttp.StatusInternalServerError)
		}
	}()
	s := h.m.holder.Load()
	if s == nil {
		stdhttp.Error(w, "proxy not ready", stdhttp.StatusServiceUnavailable)
		return
	}
	s.ServeHTTP(w, r)
}

func (m *Manager) recoverHandlerPanic(recovered interface{}) {
	m.logf("proxy: panic in request handler: %v\n%s", recovered, debug.Stack())
	m.markListenRestartAfterPanic()
	m.triggerListenRestart()
}

// triggerListenRestart останавливает текущий http.Server, чтобы RunContext перезапустил слушатель.
func (m *Manager) triggerListenRestart() {
	m.mu.Lock()
	srv := m.httpSrv
	m.mu.Unlock()
	if srv == nil {
		return
	}
	go func() {
		_ = srv.Close()
	}()
}

// Apply пересобирает прокси и применяет конфиг (hot reload).
func (m *Manager) Apply(ctx context.Context, cfg config.Config, aclEngine *acl.Engine, inspectRunner *inspect.Runner) error {
	cfg = cfg.WithDefaults()

	if m.holder.Load() != nil && m.listen == cfg.Listen &&
		config.RuntimeEqual(m.appliedCfg, cfg) &&
		m.appliedEngine == aclEngine && m.appliedInspect == inspectRunner {
		return nil
	}

	authCache := m.prepareAuthCache(cfg.Auth)
	newSrv, err := New(cfg, m.instanceID, aclEngine, inspectRunner, m.hooks, m.accessRec, m.instanceLogger(), authCache)
	if err != nil {
		m.logStartFailure(err)
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.httpSrv == nil || m.listen != cfg.Listen {
		ln, err := net.Listen("tcp", cfg.Listen)
		if err != nil {
			m.logStartFailure(err)
			return err
		}
		if m.httpSrv != nil {
			_ = m.httpSrv.Shutdown(context.Background())
			m.httpSrv = nil
			m.listening.Store(false)
		}
		m.listen = cfg.Listen
		m.holder.Store(newSrv)
		m.conns.closeAll()
		m.logListen(newSrv)
		srv := &stdhttp.Server{
			Handler:           reloadableHandler{m: m},
			ReadHeaderTimeout: cfg.ReadHeaderTimeout,
			IdleTimeout:       cfg.IdleTimeout,
			ConnState:         m.conns.track,
		}
		m.httpSrv = srv
		m.markListeningReady()
		m.clearStartError()
		m.rememberApplied(cfg, aclEngine, inspectRunner)
		go m.runServe(ln, srv)
		return nil
	}

	m.holder.Store(newSrv)
	m.conns.closeAll()
	m.logf("proxy: configuration reloaded")
	if m.listening.Load() {
		m.clearStartError()
	}
	m.rememberApplied(cfg, aclEngine, inspectRunner)
	return nil
}

func (m *Manager) rememberApplied(cfg config.Config, aclEngine *acl.Engine, inspectRunner *inspect.Runner) {
	m.appliedCfg = cfg
	m.appliedEngine = aclEngine
	m.appliedInspect = inspectRunner
}

func (m *Manager) runServe(ln net.Listener, srv *stdhttp.Server) {
	defer func() {
		if rec := recover(); rec != nil {
			m.logf("proxy: panic in listener: %v\n%s", rec, debug.Stack())
			m.markListenRestartAfterPanic()
			m.listening.Store(false)
		}
	}()
	err := srv.Serve(ln)
	m.listening.Store(false)
	if err != nil && err != stdhttp.ErrServerClosed {
		m.logStartFailure(err)
	}
}

func (m *Manager) prepareAuthCache(a config.AuthConfig) *auth.AuthCache {
	ttl := a.CacheTTL
	if ttl <= 0 {
		if m.authCache != nil {
			m.authCache.Clear()
		}
		return nil
	}
	if m.authCache == nil {
		m.authCache = auth.NewAuthCache(ttl)
	} else {
		m.authCache.Reconfigure(ttl)
	}
	return m.authCache
}

func (m *Manager) logListen(s *Server) {
	logger := m.instanceLogger()
	logger.Printf("listening on %s (connect=%s)", s.cfg.Listen, s.cfg.Connect)
	if s.cfg.Connect == config.ConnectMITM {
		logger.Printf("MITM CA: %s", s.cfg.CACertPath)
	}
	LogConnectCAStatus(logger, s.cfg)
	if s.auth != nil {
		logger.Printf("proxy auth: Basic realm=%q backend=%s", s.cfg.Auth.Realm, authBackendLabel(s.cfg.Auth))
		if s.auth.Cache != nil {
			logger.Printf("proxy auth cache: ttl=%s", s.auth.Cache.TTL())
		}
	}
	if s.aclSource == "database" {
		logger.Printf("acl: database")
	}
}

// ProxyActive сообщает, слушает ли прокси настроенный адрес.
func (m *Manager) ProxyActive() bool {
	return m.listening.Load()
}

// ProxyListen возвращает адрес прослушивания активного прокси.
func (m *Manager) ProxyListen() string {
	s := m.holder.Load()
	if s == nil {
		return ""
	}
	return s.cfg.Listen
}

// ProxyLastStartError — текст последней ошибки запуска (пусто, если прокси слушает или ошибки не было).
func (m *Manager) ProxyLastStartError() string {
	v, _ := m.lastStartErr.Load().(string)
	return v
}

func (m *Manager) setStartError(err error) {
	if err == nil {
		m.clearStartError()
		return
	}
	m.lastStartErr.Store(apperr.ProxyStartErrorMessage(err))
}

func (m *Manager) logStartFailure(err error) {
	if err == nil {
		return
	}
	m.setStartError(err)
	m.logf("proxy: не удалось запустить: %s (%v)", apperr.ProxyStartErrorMessage(err), err)
}

func (m *Manager) clearStartError() {
	m.lastStartErr.Store("")
}

func (m *Manager) markListeningReady() {
	m.listening.Store(true)
	startup.MarkProxyReady()
}

// ProxyTraffic возвращает метрики нагрузки для API.
func (m *Manager) ProxyTraffic() metrics.Snapshot {
	var snap metrics.Snapshot
	if m.instanceID != uuid.Nil {
		snap = metrics.DefaultRegistry().Collector(m.instanceID).TrafficSnapshot()
	} else {
		snap = metrics.SnapshotTraffic()
	}
	snap.ActiveConnections = m.conns.count()
	snap.ActiveWebSocketConnections = metrics.ActiveWebSocketConnections()
	if m.accessBatcher != nil {
		snap.AccessLogQueuePending = m.accessBatcher.QueueDepth()
	}
	return snap
}

// RunContext запускает прокси с фиксированным конфигом до отмены ctx.
func (m *Manager) RunContext(ctx context.Context, cfg config.Config, aclEngine *acl.Engine, inspectRunner *inspect.Runner) error {
	return m.RunContextWithLoader(ctx, func(context.Context) (config.Config, *acl.Engine, *inspect.Runner, error) {
		return cfg, aclEngine, inspectRunner, nil
	})
}

// RunContextWithLoader на каждой итерации загружает runtime (например из БД).
// Если порт занят, пишет в лог и повторяет попытку каждые 10 секунд.
func (m *Manager) RunContextWithLoader(ctx context.Context, load RuntimeLoader) error {
	for {
		if err := ctx.Err(); err != nil {
			return nil
		}

		cfg, aclEngine, inspectRunner, err := load(ctx)
		if err != nil {
			if errors.Is(err, ErrRuntimeDeferred) {
				if !waitFor(ctx, proxyListenRetryInterval) {
					m.shutdownProxy()
					return nil
				}
				continue
			}
			return err
		}

		err = m.Apply(ctx, cfg, aclEngine, inspectRunner)
		switch {
		case err != nil && !isListenBindError(err):
			return err
		case err != nil:
			// ошибка уже залогирована в Apply
		case m.listening.Load():
			m.waitUntilNotListening(ctx)
			m.shutdownProxy()
			if ctx.Err() != nil {
				return nil
			}
			retry := proxyListenRetryInterval
			if m.consumeListenRestartAfterPanic() {
				retry = panicRecoveryDelay
				m.logf("proxy: восстановление после паники через %s", retry)
			} else {
				m.logf("proxy: повтор запуска...")
			}
			if !waitFor(ctx, retry) {
				m.shutdownProxy()
				return nil
			}
			continue
		}

		if !waitFor(ctx, proxyListenRetryInterval) {
			m.shutdownProxy()
			return nil
		}
	}
}

// Stop останавливает прокси (слушатель и активные соединения).
func (m *Manager) Stop() {
	m.shutdownProxy()
	m.clearStartError()
}

func (m *Manager) shutdownProxy() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.httpSrv != nil {
		_ = m.httpSrv.Shutdown(context.Background())
		m.httpSrv = nil
	}
	m.listening.Store(false)
	m.appliedCfg = config.Config{}
	m.appliedEngine = nil
	m.appliedInspect = nil
}

func (m *Manager) waitUntilNotListening(ctx context.Context) {
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	for {
		if ctx.Err() != nil {
			return
		}
		if !m.listening.Load() {
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
		}
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

func isListenBindError(err error) bool {
	return apperr.IsListenBindError(err)
}

func (m *Manager) markListenRestartAfterPanic() {
	m.listenRestartAfterPanic.Store(true)
}

func (m *Manager) consumeListenRestartAfterPanic() bool {
	return m.listenRestartAfterPanic.Swap(false)
}
