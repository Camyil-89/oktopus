package outboundtls

import (
	"context"
	"crypto/tls"
)

type verifyKey struct{}

// WithVerify помечает, нужно ли проверять сертификат origin при исходящем TLS.
// verify=true — стандартная проверка; verify=false — InsecureSkipVerify.
func WithVerify(ctx context.Context, verify bool) context.Context {
	return context.WithValue(ctx, verifyKey{}, verify)
}

// VerifyFromContext возвращает флаг из контекста. Если не задан — ok=false (по умолчанию проверять).
func VerifyFromContext(ctx context.Context) (verify bool, ok bool) {
	if ctx == nil {
		return true, false
	}
	v, ok := ctx.Value(verifyKey{}).(bool)
	if !ok {
		return true, false
	}
	return v, true
}

// ClientConfig собирает tls.Config для соединения прокси → origin.
func ClientConfig(ctx context.Context, serverName string) *tls.Config {
	cfg := &tls.Config{
		ServerName: serverName,
		MinVersion: tls.VersionTLS12,
	}
	if verify, ok := VerifyFromContext(ctx); ok && !verify {
		cfg.InsecureSkipVerify = true
	}
	return cfg
}
