package observe

import "context"

type sniKey struct{}

// WithSNI сохраняет TLS SNI в контексте (MITM выставляет перед вызовом middleware).
func WithSNI(ctx context.Context, sni string) context.Context {
	if sni == "" {
		return ctx
	}
	return context.WithValue(ctx, sniKey{}, sni)
}

// SNIFromContext возвращает SNI, если он был записан в контекст.
func SNIFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	v, _ := ctx.Value(sniKey{}).(string)
	return v
}
