package hooks

import (
	"context"
	stdhttp "net/http"

	"oktopus/internal/proxy/ratelimit"
)

// RateLimitPolicy выбирает delay pool (Squid delay_access) для сессии.
type RateLimitPolicy interface {
	DelayFlowConnect(ctx context.Context, hostPort string) *ratelimit.Flow
	DelayFlowHTTP(ctx context.Context, req *stdhttp.Request) *ratelimit.Flow
}
