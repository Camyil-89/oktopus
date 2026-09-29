package observe

import "context"

type clientAddrKey struct{}

// WithRemoteAddr сохраняет адрес клиента (например r.RemoteAddr) для CONNECT, где нет *http.Request в hooks.
func WithRemoteAddr(ctx context.Context, remoteAddr string) context.Context {
	if remoteAddr == "" {
		return ctx
	}
	return context.WithValue(ctx, clientAddrKey{}, remoteAddr)
}

// RemoteAddrFromContext возвращает адрес, если был установлен WithRemoteAddr.
func RemoteAddrFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	v, _ := ctx.Value(clientAddrKey{}).(string)
	return v
}
