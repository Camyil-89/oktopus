package observe

import (
	"context"
	"strings"
)

type mitmClientHelloSNIKey struct{}
type connectDestHostPortKey struct{}

// WithMITMClientHelloSNI — SNI из ClientHello клиента (до нормализации policy host).
func WithMITMClientHelloSNI(ctx context.Context, sni string) context.Context {
	sni = normalizeTraceHost(strings.TrimSpace(sni))
	if sni == "" {
		return ctx
	}
	return context.WithValue(ctx, mitmClientHelloSNIKey{}, sni)
}

// MITMClientHelloSNIFromContext возвращает ClientHello SNI в MITM-сессии.
func MITMClientHelloSNIFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	v, _ := ctx.Value(mitmClientHelloSNIKey{}).(string)
	return v
}

// WithCONNECTDestHostPort — цель CONNECT (host:port), как в r.Host CONNECT.
func WithCONNECTDestHostPort(ctx context.Context, hostPort string) context.Context {
	hostPort = strings.TrimSpace(hostPort)
	if hostPort == "" {
		return ctx
	}
	return context.WithValue(ctx, connectDestHostPortKey{}, hostPort)
}

// CONNECTDestHostPortFromContext возвращает host:port CONNECT-туннеля.
func CONNECTDestHostPortFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	v, _ := ctx.Value(connectDestHostPortKey{}).(string)
	return v
}
