package observe

import "context"

type deferPortACLKey struct{}

type portOnlyACLKey struct{}

// WithDeferPortACLAtConnect — PORT не проверяется на CONNECT (проверка на первом HTTP в TLS).
func WithDeferPortACLAtConnect(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, deferPortACLKey{}, true)
}

// DeferPortACLAtConnect сообщает, что правила PORT нужно пропустить в OnConnect.
func DeferPortACLAtConnect(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	v, _ := ctx.Value(deferPortACLKey{}).(bool)
	return v
}

// WithConnectPortOnlyACL — на CONNECT учитываются только правила PORT (повторная проверка в tunnel).
func WithConnectPortOnlyACL(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, portOnlyACLKey{}, true)
}

// ConnectPortOnlyACL сообщает, что в OnConnect нужно матчить только PORT.
func ConnectPortOnlyACL(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	v, _ := ctx.Value(portOnlyACLKey{}).(bool)
	return v
}
