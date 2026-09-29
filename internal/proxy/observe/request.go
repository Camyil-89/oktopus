package observe

import (
	"context"
	stdhttp "net/http"
	"net/url"

	"oktopus/internal/proxy/auth"
	proxyhttp "oktopus/internal/proxy/http"
)

// RequestTools — ленивый доступ к полям запроса в middleware; парсинг только при вызове методов.
type RequestTools struct {
	ctx context.Context
	req *stdhttp.Request
}

// RequestToolsFor создаёт инструменты без парсинга URL и query.
func RequestToolsFor(ctx context.Context, req *stdhttp.Request) *RequestTools {
	return &RequestTools{ctx: ctx, req: req}
}

// Path возвращает путь запроса (без query).
func (t *RequestTools) Path() string {
	if t.req == nil || t.req.URL == nil {
		return ""
	}
	return t.req.URL.Path
}

// Query парсит и возвращает все query-параметры.
func (t *RequestTools) Query() url.Values {
	if t.req == nil || t.req.URL == nil {
		return make(url.Values)
	}
	return t.req.URL.Query()
}

// SNI возвращает TLS Server Name из контекста (после MITM), иначе пустую строку.
func (t *RequestTools) SNI() string {
	return SNIFromContext(t.ctx)
}

// SearchQuery — типичный поисковый термин из query (q, query, text, …); парсит query при вызове.
func (t *RequestTools) SearchQuery() string {
	if t.req == nil {
		return ""
	}
	return proxyhttp.ParseSearchQuery(t.req.URL)
}

// Identity — proxy-auth пользователь (static или LDAP) из context; если не авторизован — nil.
func (t *RequestTools) Identity() *auth.Identity {
	if t.ctx == nil {
		return nil
	}
	id, ok := auth.IdentityFromContext(t.ctx)
	if !ok || id.Username == "" {
		return nil
	}
	out := id
	return &out
}
