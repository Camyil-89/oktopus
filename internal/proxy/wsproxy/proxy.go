package wsproxy

import (
	"bufio"
	"context"
	"net"
	stdhttp "net/http"

	"oktopus/internal/proxy/hooks"
	"oktopus/internal/proxy/ratelimit"
)

// RelayMode — синхронный relay (MITM) или в фоне (plain HTTP handler).
type RelayMode int

const (
	RelayAsync RelayMode = 0
	RelayBlock RelayMode = 1
)

// Config — общий апстрим и хуки ответа для WebSocket upgrade.
type Config struct {
	Hooks     *hooks.Hooks
	RateLimit hooks.RateLimitPolicy
	Dial      func(ctx context.Context, outReq *stdhttp.Request) (net.Conn, error)
}

// ClientSink доставляет ответ клиенту (политика, 101 или ошибка origin).
type ClientSink interface {
	WritePolicy(ctx context.Context, outReq *stdhttp.Request, d hooks.Decision) error
	DeliverUpstream(ctx context.Context, outReq *stdhttp.Request, res *stdhttp.Response) (RelayLegs, error)
}

// Proxy выполняет handshake с origin и при 101 — relay фреймов.
// ACL на запрос вызывающий код применяет до Proxy.
//
// continueSession: для MITM false — закрыть TLS-сессию; для plain HTTP всегда true при err==nil.
func Proxy(ctx context.Context, cfg Config, outReq *stdhttp.Request, sink ClientSink, mode RelayMode) (continueSession bool, err error) {
	upstream, err := cfg.Dial(ctx, outReq)
	if err != nil {
		return true, err
	}

	if err := outReq.Write(upstream); err != nil {
		upstream.Close()
		return true, err
	}

	upBR := bufio.NewReader(upstream)
	outRes, err := stdhttp.ReadResponse(upBR, outReq)
	if err != nil {
		upstream.Close()
		return true, err
	}
	defer outRes.Body.Close()

	if d := cfg.Hooks.RunHTTPResponse(outReq.Context(), outReq, outRes); d.Handled() {
		upstream.Close()
		_ = sink.WritePolicy(outReq.Context(), outReq, d)
		return true, nil
	}

	legs, err := sink.DeliverUpstream(ctx, outReq, outRes)
	if err != nil {
		upstream.Close()
		return true, err
	}
	if !legs.Switching {
		upstream.Close()
		return true, nil
	}

	var flow *ratelimit.Flow
	if cfg.RateLimit != nil {
		flow = cfg.RateLimit.DelayFlowHTTP(ctx, outReq)
	}

	if mode == RelayBlock {
		RelayPair(legs.ClientWrite, legs.ClientRead, upstream, upBR, flow)
		return false, nil
	}

	go RelayPair(legs.ClientWrite, legs.ClientRead, upstream, upBR, flow)
	return true, nil
}
