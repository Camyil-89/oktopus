package hooks

import (
	"context"
	stdhttp "net/http"
)

// Hooks — middleware и колбэки прокси (CONNECT, запрос, ответ).
// Каждый колбэк возвращает Decision: по умолчанию AllowDecision() без редиректа.
// Несколько middleware можно собрать через ChainConnect / ChainHTTPRequest / ChainHTTPResponse.
type Hooks struct {
	OnHTTPRequest  HTTPRequestMiddleware
	OnHTTPResponse HTTPResponseMiddleware
	OnConnect      ConnectMiddleware
}

// RunConnect вызывает OnConnect или разрешает подключение.
func (h *Hooks) RunConnect(ctx context.Context, hostPort string) Decision {
	if h == nil || h.OnConnect == nil {
		return AllowDecision()
	}
	return h.OnConnect(ctx, hostPort)
}

// RunHTTPRequest вызывает OnHTTPRequest или разрешает запрос.
func (h *Hooks) RunHTTPRequest(ctx context.Context, req *stdhttp.Request) Decision {
	if h == nil || h.OnHTTPRequest == nil {
		return AllowDecision()
	}
	return h.OnHTTPRequest(ctx, req)
}

// RunHTTPResponse вызывает OnHTTPResponse или разрешает отдачу ответа.
func (h *Hooks) RunHTTPResponse(ctx context.Context, req *stdhttp.Request, res *stdhttp.Response) Decision {
	if h == nil || h.OnHTTPResponse == nil {
		return AllowDecision()
	}
	return h.OnHTTPResponse(ctx, req, res)
}
