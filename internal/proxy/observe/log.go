package observe

import (
	"context"
	"log"
	stdhttp "net/http"

	"oktopus/internal/proxy/hooks"
)

// LogHooks заполняет стандартные колбэки логирования (всегда AllowDecision).
func LogHooks(logger *log.Logger) *hooks.Hooks {
	if logger == nil {
		logger = log.Default()
	}
	return &hooks.Hooks{
		OnHTTPRequest: hooks.ChainHTTPRequest(func(ctx context.Context, req *stdhttp.Request) hooks.Decision {
			// tools := RequestToolsFor(ctx, req)
			// search := tools.SearchQuery()
			// path := tools.Path()
			// sni := tools.SNI()
			// scheme := "https"
			// if req.URL != nil && req.URL.Scheme != "" {
			// 	scheme = req.URL.Scheme
			// }
			// user := "-"
			// if id := tools.Identity(); id != nil {
			// 	user = id.Username
			// }
			// if req.URL != nil {
			// 	logger.Printf("[%s] %s %s user=%s path=%q sni=%q search=%q",
			// 		scheme, req.Method, req.URL.Redacted(), user, path, sni, search)
			// }
			return hooks.AllowDecision()
		}),
		OnHTTPResponse: func(ctx context.Context, req *stdhttp.Request, res *stdhttp.Response) hooks.Decision {
			return hooks.AllowDecision()
		},
		OnConnect: func(ctx context.Context, hostPort string) hooks.Decision {
			// user := "-"
			// if id := RequestToolsFor(ctx, nil).Identity(); id != nil {
			// 	user = id.Username
			// }
			// logger.Printf("[connect] %s user=%s", hostPort, user)
			return hooks.AllowDecision()
		},
	}
}
