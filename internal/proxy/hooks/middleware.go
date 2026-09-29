package hooks

import (
	"context"
	stdhttp "net/http"
)

// ConnectMiddleware — middleware на установку CONNECT-туннеля.
type ConnectMiddleware func(ctx context.Context, hostPort string) Decision

// HTTPRequestMiddleware — middleware до запроса к origin.
type HTTPRequestMiddleware func(ctx context.Context, req *stdhttp.Request) Decision

// HTTPResponseMiddleware — middleware после ответа origin, до отдачи клиенту.
type HTTPResponseMiddleware func(ctx context.Context, req *stdhttp.Request, res *stdhttp.Response) Decision

// ChainConnect выполняет middleware по порядку; первый запрет или редирект останавливает цепочку.
func ChainConnect(mws ...ConnectMiddleware) ConnectMiddleware {
	return func(ctx context.Context, hostPort string) Decision {
		for _, mw := range mws {
			if mw == nil {
				continue
			}
			d := mw(ctx, hostPort)
			if d.Handled() {
				return d
			}
		}
		return AllowDecision()
	}
}

// ChainHTTPRequest — цепочка middleware на HTTP-запрос.
func ChainHTTPRequest(mws ...HTTPRequestMiddleware) HTTPRequestMiddleware {
	return func(ctx context.Context, req *stdhttp.Request) Decision {
		for _, mw := range mws {
			if mw == nil {
				continue
			}
			d := mw(ctx, req)
			if d.Handled() {
				return d
			}
		}
		return AllowDecision()
	}
}

// ChainHTTPResponse — цепочка middleware на HTTP-ответ.
func ChainHTTPResponse(mws ...HTTPResponseMiddleware) HTTPResponseMiddleware {
	return func(ctx context.Context, req *stdhttp.Request, res *stdhttp.Response) Decision {
		for _, mw := range mws {
			if mw == nil {
				continue
			}
			d := mw(ctx, req, res)
			if d.Handled() {
				return d
			}
		}
		return AllowDecision()
	}
}
