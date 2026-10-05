package instancectx

import (
	"context"

	"github.com/google/uuid"
)

type ctxKey struct{}

func WithID(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, ctxKey{}, id)
}

func ID(ctx context.Context) (uuid.UUID, bool) {
	v, ok := ctx.Value(ctxKey{}).(uuid.UUID)
	return v, ok && v != uuid.Nil
}
