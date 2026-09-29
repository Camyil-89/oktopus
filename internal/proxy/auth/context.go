package auth

import "context"

type ctxKey struct{}

// WithIdentity сохраняет пользователя в context после успешной proxy-auth.
func WithIdentity(ctx context.Context, id Identity) context.Context {
	return context.WithValue(ctx, ctxKey{}, id)
}

// IdentityFromContext возвращает identity, если auth уже прошла.
func IdentityFromContext(ctx context.Context) (Identity, bool) {
	if ctx == nil {
		return Identity{}, false
	}
	v := ctx.Value(ctxKey{})
	if v == nil {
		return Identity{}, false
	}
	id, ok := v.(Identity)
	return id, ok
}
