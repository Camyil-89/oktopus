package acl

import (
	"context"
	"net"
	stdhttp "net/http"

	"oktopus/internal/proxy/auth"
	"oktopus/internal/proxy/observe"
	"oktopus/internal/proxy/ratelimit"
)

// SquidDelayAccess — строка delay_access.
type SquidDelayAccess struct {
	LineNo  int
	Pool    int
	Action  Action // allow = применить pool, deny = без ограничения
	Clauses []SquidAccessClause
}

// DelayFlow возвращает ограничитель по первой совпавшей delay_access (Squid).
func (e *Engine) DelayFlow(id auth.Identity, f RequestFields, phase SquidPhase) *ratelimit.Flow {
	if e == nil || e.rateLimit == nil || len(e.squidDelayAccess) == 0 {
		return nil
	}
	f = normalizeRequestFields(f)
	for _, line := range e.squidDelayAccess {
		tmp := SquidHTTPAccess{Clauses: line.Clauses}
		if !lineMatches(tmp, id, f, phase, nil) {
			continue
		}
		if line.Action != ActionAllow {
			return nil
		}
		return e.rateLimit.Flow(line.Pool, f.SrcIP)
	}
	return nil
}

// DelayFlowConnect реализует hooks.RateLimitPolicy для CONNECT.
func (e *Engine) DelayFlowConnect(ctx context.Context, hostPort string) *ratelimit.Flow {
	id, _ := auth.IdentityFromContext(ctx)
	src := ParseClientIP(observe.RemoteAddrFromContext(ctx))
	return e.DelayFlowConnectFor(id, hostPort, src)
}

// DelayFlowConnectFor выбирает pool по identity и IP.
func (e *Engine) DelayFlowConnectFor(id auth.Identity, hostPort string, srcIP net.IP) *ratelimit.Flow {
	sni := hostPort
	if h, _, err := net.SplitHostPort(hostPort); err == nil {
		sni = h
	}
	return e.DelayFlow(id, RequestFields{
		SNI:     sni,
		Path:    "/",
		SrcIP:   srcIP,
		DstPort: ParseDstPort(hostPort),
	}, SquidPhaseConnect)
}

// DelayFlowHTTP выбирает pool для HTTP-запроса.
func (e *Engine) DelayFlowHTTP(ctx context.Context, req *stdhttp.Request) *ratelimit.Flow {
	if e == nil || req == nil {
		return nil
	}
	id, _ := auth.IdentityFromContext(ctx)
	tools := observe.RequestToolsFor(ctx, req)
	sni := tools.SNI()
	if sni == "" && req.URL != nil && req.URL.Host != "" {
		sni = hostOnly(req.URL.Host)
	}
	path := tools.Path()
	if path == "" {
		path = "/"
	}
	src := ParseClientIP(req.RemoteAddr)
	if src == nil {
		src = ParseClientIP(observe.RemoteAddrFromContext(ctx))
	}
	return e.DelayFlow(id, RequestFields{
		SNI: sni, Path: path, SrcIP: src, DstPort: httpDstPort(req),
	}, SquidPhaseHTTP)
}
