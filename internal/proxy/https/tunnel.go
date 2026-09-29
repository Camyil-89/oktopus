package https

import (
	"context"
	"fmt"
	"net"
	stdhttp "net/http"
	"time"

	"oktopus/internal/pki"
	"oktopus/internal/proxy/hooks"
	"oktopus/internal/proxy/observe"
	"oktopus/internal/proxy/ratelimit"
)

// Tunnel — CONNECT + прозрачный TCP (без расшифровки) для разрешённого трафика.
// При блокировке PORT — та же MITM-сессия, что в mitm (Forbidden на первом HTTP).
type Tunnel struct {
	Hooks     *hooks.Hooks
	CA        *pki.Authority
	RateLimit hooks.RateLimitPolicy
}

// Serve выполняет CONNECT и двунаправленное копирование байт или MITM Forbidden при PORT deny.
func (t *Tunnel) Serve(ctx context.Context, w stdhttp.ResponseWriter, r *stdhttp.Request) error {
	hostPort, proceed, err := serveConnectAfterACL(ctx, w, r, t.Hooks, t.CA)
	if err != nil || !proceed {
		return err
	}

	portCtx := observe.WithConnectPortOnlyACL(ctx)
	if dPort := t.Hooks.RunConnect(portCtx, hostPort); !dPort.Allow {
		return serveForbiddenMITM(ctx, w, r, hostPort, t.CA, t.Hooks)
	}

	return t.serveRelay(ctx, w, r, hostPort)
}

func (t *Tunnel) serveRelay(ctx context.Context, w stdhttp.ResponseWriter, r *stdhttp.Request, hostPort string) error {
	hj, ok := w.(stdhttp.Hijacker)
	if !ok {
		return fmt.Errorf("hijack not supported")
	}

	clientConn, bufrw, err := hj.Hijack()
	if err != nil {
		return err
	}

	if _, err = bufrw.WriteString("HTTP/1.1 200 Connection Established\r\n\r\n"); err != nil {
		clientConn.Close()
		return err
	}
	if err := bufrw.Flush(); err != nil {
		clientConn.Close()
		return err
	}

	dialer := &net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 30 * time.Second,
	}
	upstream, err := dialer.DialContext(ctx, "tcp", hostPort)
	if err != nil {
		clientConn.Close()
		return err
	}

	var flow *ratelimit.Flow
	if t.RateLimit != nil {
		flow = t.RateLimit.DelayFlowConnect(ctx, hostPort)
	}

	go Relay(upstream, clientConn, flow, RelayFromClient, true)
	go Relay(clientConn, upstream, flow, RelayToClient, true)
	return nil
}
