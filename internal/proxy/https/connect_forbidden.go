package https

import (
	"context"
	stdhttp "net/http"

	"oktopus/internal/pki"
	"oktopus/internal/proxy/hooks"
)

// finishConnectDecision обрабатывает редирект или отказ ACL на CONNECT.
// При deny и наличии CA открывает MITM-сессию (200 Established), чтобы клиент
// получил HTML Forbidden на первом HTTP внутри TLS, а не ERR_TUNNEL_CONNECTION_FAILED.
// Возвращает true, если ответ клиенту уже полностью отправлен или запущена MITM-сессия.
func finishConnectDecision(ctx context.Context, w stdhttp.ResponseWriter, r *stdhttp.Request, hostPort string, d hooks.Decision, ca *pki.Authority, h *hooks.Hooks) (done bool, err error) {
	if d.Redirect != nil {
		d.WriteResponse(w, r)
		return true, nil
	}
	if d.Allow {
		return false, nil
	}
	return true, serveForbiddenMITM(ctx, w, r, hostPort, ca, h)
}

func serveForbiddenMITM(ctx context.Context, w stdhttp.ResponseWriter, r *stdhttp.Request, hostPort string, ca *pki.Authority, h *hooks.Hooks) error {
	if ca == nil || h == nil {
		hooks.DenyDecision().WriteResponse(w, r)
		return nil
	}
	mitm := &MITM{CA: ca, Hooks: h}
	return mitm.serveConnectEstablished(ctx, w, r, hostPort, true)
}
