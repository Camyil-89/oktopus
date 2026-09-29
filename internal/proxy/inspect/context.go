package inspect

import (
	"context"
	stdhttp "net/http"
	"strings"

	"oktopus/internal/proxy/auth"
	"oktopus/internal/proxy/observe"
)

// RequestContext — данные исходящего HTTP-запроса для Lua (только чтение).
type RequestContext struct {
	Method        string
	Host          string
	Path          string
	URL           string
	ContentType   string
	ContentLength int64
	User          string
	Groups        []string
	ClientIP        string
	UploadFilenames []string
	headers         stdhttp.Header
}

func RequestContextFromHTTP(ctx context.Context, req *stdhttp.Request) RequestContext {
	if req == nil {
		return RequestContext{}
	}
	tools := observe.RequestToolsFor(ctx, req)
	host := observe.PolicyHostFromRequest(req, tools.SNI())
	path := tools.Path()
	if path == "" {
		path = "/"
	}
	fullURL := ""
	if req.URL != nil {
		fullURL = req.URL.String()
	}
	ct := strings.TrimSpace(req.Header.Get("Content-Type"))
	user := ""
	var groups []string
	if id, ok := auth.IdentityFromContext(ctx); ok {
		user = id.Username
		groups = append(groups, id.Groups...)
	}
	clientIP := ""
	if req.RemoteAddr != "" {
		clientIP = req.RemoteAddr
	} else if a := observe.RemoteAddrFromContext(ctx); a != "" {
		clientIP = a
	}
	return RequestContext{
		Method:        req.Method,
		Host:          host,
		Path:          path,
		URL:           fullURL,
		ContentType:   ct,
		ContentLength: req.ContentLength,
		User:          user,
		Groups:        groups,
		ClientIP:      clientIP,
		headers:       req.Header,
	}
}

func (c RequestContext) Header(name string) string {
	if c.headers == nil {
		return ""
	}
	return strings.TrimSpace(c.headers.Get(name))
}

func (c RequestContext) HasGroup(name string) bool {
	name = strings.TrimSpace(name)
	for _, g := range c.Groups {
		if strings.EqualFold(g, name) {
			return true
		}
	}
	return false
}
