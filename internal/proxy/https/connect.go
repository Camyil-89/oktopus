package https

import (
	"context"
	stdhttp "net/http"
)

// ConnectHandler обрабатывает HTTP CONNECT (HTTPS).
type ConnectHandler interface {
	Serve(ctx context.Context, w stdhttp.ResponseWriter, r *stdhttp.Request) error
}
