package https

import (
	"context"
	stdhttp "net/http"

	"oktopus/internal/proxy/hooks"
)

// applyMITMHTTPPolicy нормализует origin (URL.Host, SNI) и прогоняет тот же ACL-пайплайн,
// что обычный HTTP и WebSocket upgrade — без отдельных проверок.
func applyMITMHTTPPolicy(ctx context.Context, h *hooks.Hooks, req *stdhttp.Request, defaultHost string) (*stdhttp.Request, hooks.Decision) {
	outReq := prepareMITMOutboundRequest(ctx, req, defaultHost)
	d := h.RunHTTPRequest(outReq.Context(), outReq)
	return outReq, d
}
