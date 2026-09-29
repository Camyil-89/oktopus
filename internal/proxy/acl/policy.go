package acl

import (
	"context"
	"net"
	stdhttp "net/http"
	"strings"
	"time"

	"oktopus/internal/proxy/auth"
	"oktopus/internal/proxy/observe"
)

// PolicyRequest — единый вход в http_access для CONNECT, plain HTTP, HTTPS MITM и WebSocket
// (WS = HTTP GET upgrade; те же поля, что PolicyRequestFromHTTP).
type PolicyRequest struct {
	Host    string // host для dstdomain / dst (без порта)
	Path    string
	SrcIP   net.IP
	DstPort int
}

// PolicyEvalResult — итог EvaluatePolicy.
type PolicyEvalResult struct {
	Allow   bool
	RuleRef string
	Fields  RequestFields
	Spend   time.Duration
	Prepare time.Duration
	Engine  time.Duration
	Timing  EngineTiming
}

// PolicyRequestFromConnect собирает поля из CONNECT host:port (минимальный запрос).
func PolicyRequestFromConnect(ctx context.Context, hostPort string) PolicyRequest {
	return PolicyRequest{
		Host:    hostOnly(hostPort),
		Path:    "/",
		SrcIP:   ParseClientIP(observe.RemoteAddrFromContext(ctx)),
		DstPort: ParseDstPort(hostPort),
	}
}

// PolicyRequestFromHTTP собирает поля из *http.Request (богаче: URL, Host, path, SNI из контекста MITM).
func PolicyRequestFromHTTP(ctx context.Context, req *stdhttp.Request) PolicyRequest {
	tools := observe.RequestToolsFor(ctx, req)
	host := observe.PolicyHostFromRequest(req, tools.SNI())
	path := tools.Path()
	if path == "" {
		path = "/"
	}
	src := ParseClientIP(req.RemoteAddr)
	if src == nil {
		src = ParseClientIP(observe.RemoteAddrFromContext(ctx))
	}
	return PolicyRequest{
		Host:    strings.TrimSpace(host),
		Path:    path,
		SrcIP:   src,
		DstPort: httpDstPort(req),
	}
}

func (pr PolicyRequest) requestFields() RequestFields {
	return RequestFields{
		SNI:     pr.Host,
		Path:    pr.Path,
		SrcIP:   pr.SrcIP,
		DstPort: pr.DstPort,
	}
}

// EvaluatePolicy — единственная точка decide+dst-resolve для всех транспортов.
// skip — как connectSkipRuleTypes (PORT-only на CONNECT и т.д.).
func (e *Engine) EvaluatePolicy(ctx context.Context, id auth.Identity, pr PolicyRequest, skip map[RuleType]bool) PolicyEvalResult {
	start := time.Now()
	fields := pr.requestFields()
	if needsDstResolve(skip) {
		fields = EnrichRequestFieldsDst(ctx, fields)
	}
	prepareEnd := time.Now()
	var etm EngineTiming
	allow, ruleRef, _ := e.decideWithRuleRef(id, fields, &etm, skip)
	decideEnd := time.Now()
	return PolicyEvalResult{
		Allow:   allow,
		RuleRef: ruleRef,
		Fields:  fields,
		Spend:   time.Since(start),
		Prepare: prepareEnd.Sub(start),
		Engine:  decideEnd.Sub(prepareEnd),
		Timing:  etm,
	}
}

// ConnectSkipFromContext — skip-маска для EvaluatePolicy на CONNECT.
func ConnectSkipFromContext(ctx context.Context) map[RuleType]bool {
	return connectSkipRuleTypes(ctx)
}
