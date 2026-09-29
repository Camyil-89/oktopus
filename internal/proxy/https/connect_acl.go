package https

import (
	"context"
	"fmt"
	stdhttp "net/http"

	"oktopus/internal/pki"
	"oktopus/internal/proxy/hooks"
)

// serveConnectAfterACL — общая фаза CONNECT для Tunnel и MITM: Host, RunConnect, deny/redirect.
// Возвращает hostPort и false, если ответ клиенту уже отправлен (deny MITM / redirect).
func serveConnectAfterACL(ctx context.Context, w stdhttp.ResponseWriter, r *stdhttp.Request, h *hooks.Hooks, ca *pki.Authority) (hostPort string, proceed bool, err error) {
	hostPort = NormalizeHostPort(r.Host)
	if hostPort == "" {
		return "", false, fmt.Errorf("CONNECT without Host")
	}
	d := h.RunConnect(ctx, hostPort)
	if done, err := finishConnectDecision(ctx, w, r, hostPort, d, ca, h); done || err != nil {
		return hostPort, false, err
	}
	return hostPort, true, nil
}
