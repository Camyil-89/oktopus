package acl

import (
	"context"
	"net"
	stdhttp "net/http"
	"strings"
	"time"

	"oktopus/internal/proxy/accesslog"
	"oktopus/internal/proxy/auth"
	"oktopus/internal/proxy/hooks"
	"oktopus/internal/proxy/metrics"
	"oktopus/internal/proxy/observe"
	"oktopus/internal/proxy/outboundtls"
)

// Hooks возвращает middleware для цепочки hooks прокси.
// rec может быть nil — тогда решения не логируются.
func (e *Engine) Hooks(rec accesslog.Recorder) *hooks.Hooks {
	if e == nil {
		e = EmptyEngine()
	}
	return &hooks.Hooks{
		OnConnect:      e.connectMiddleware(rec),
		OnHTTPRequest:  e.httpRequestMiddleware(rec),
		OnHTTPResponse: nil,
	}
}

func (e *Engine) connectMiddleware(rec accesslog.Recorder) hooks.ConnectMiddleware {
	return func(ctx context.Context, hostPort string) hooks.Decision {
		id, _ := authIdentity(ctx)
		res := e.EvaluatePolicy(ctx, id, PolicyRequestFromConnect(ctx, hostPort), ConnectSkipFromContext(ctx))
		parts := metricsDecideParts(res.Prepare, res.Engine, res.Timing)
		metrics.ObserveDecisionWithPolicy(res.Allow, res.Spend, parts, metricsUser(ctx), metricsSource(ctx, nil))
		logCtx := observe.WithPolicyEvalTrace(ctx, policyEvalTraceFromFields(res.Fields))
		if rec != nil {
			rec.Record(accesslog.ConnectEntry(logCtx, hostPort, res.Allow, res.Spend, res.RuleRef))
		}
		if res.Allow {
			return hooks.AllowDecision()
		}
		return hooks.DenyDecision()
	}
}

func (e *Engine) httpRequestMiddleware(rec accesslog.Recorder) hooks.HTTPRequestMiddleware {
	return func(ctx context.Context, req *stdhttp.Request) hooks.Decision {
		id, _ := authIdentity(ctx)
		res := e.EvaluatePolicy(ctx, id, PolicyRequestFromHTTP(ctx, req), nil)
		parts := metricsDecideParts(res.Prepare, res.Engine, res.Timing)
		metrics.ObserveDecision(res.Allow, res.Spend, parts, metricsUser(ctx), metricsSource(ctx, req))
		trace := policyEvalTraceFromFields(res.Fields)
		if req != nil {
			base := req.Context()
			if base == nil {
				base = ctx
			}
			base = observe.WithPolicyEvalTrace(base, trace)
			*req = *req.WithContext(base)
		}
		if rec != nil && !res.Allow {
			rec.Record(accesslog.HTTPEntry(ctx, req, false, res.Spend, res.RuleRef))
		}
		if !res.Allow {
			metrics.ObservePolicyTotal(res.Spend)
		}
		if res.Allow {
			if req != nil {
				base := req.Context()
				if base == nil {
					base = ctx
				}
				base = observe.WithACLAllowPassed(base)
				base = observe.WithACLDecision(base, observe.ACLDecision{
					RuleRef: res.RuleRef,
					Spend:   res.Spend,
				})
				tlsVerify, _ := e.DecideUpstreamTLSVerify(id, res.Fields)
				base = outboundtls.WithVerify(base, tlsVerify)
				*req = *req.WithContext(base)
			}
			return hooks.AllowDecision()
		}
		return hooks.DenyDecision()
	}
}

func authIdentity(ctx context.Context) (auth.Identity, bool) {
	return auth.IdentityFromContext(ctx)
}

func metricsUser(ctx context.Context) string {
	id, ok := authIdentity(ctx)
	if !ok || id.Username == "" {
		return ""
	}
	return id.Username
}

func metricsSource(ctx context.Context, req *stdhttp.Request) string {
	if req != nil && req.RemoteAddr != "" {
		return req.RemoteAddr
	}
	return observe.RemoteAddrFromContext(ctx)
}

func metricsDecideParts(prepare, engine time.Duration, etm EngineTiming) metrics.DecideParts {
	return metrics.DecideParts{
		Prepare:    prepare,
		Engine:     engine,
		Scope:      etm.Scope,
		MatchSNI:   etm.MatchSNI,
		MatchSrcIP: etm.MatchSrcIP,
		MatchDstIP: etm.MatchDstIP,
		MatchSlow:  etm.MatchSlow,
	}
}

func hostOnly(hostPort string) string {
	hostPort = strings.TrimSpace(hostPort)
	if hostPort == "" {
		return ""
	}
	host, _, err := net.SplitHostPort(hostPort)
	if err != nil {
		return hostPort
	}
	return host
}

func httpDstPort(req *stdhttp.Request) int {
	if req == nil {
		return 0
	}
	if req.URL != nil && req.URL.Host != "" {
		if p := ParseDstPort(req.URL.Host); p != 0 {
			return p
		}
		switch strings.ToLower(req.URL.Scheme) {
		case "https":
			return 443
		case "http":
			return 80
		}
	}
	if req.Host != "" {
		if p := ParseDstPort(req.Host); p != 0 {
			return p
		}
	}
	return 0
}

func policyEvalTraceFromFields(f RequestFields) observe.PolicyEvalTrace {
	ips := make([]string, 0, len(f.DstResolved))
	for _, ip := range f.DstResolved {
		if ipValid(ip) {
			ips = append(ips, ip.String())
		}
	}
	return observe.PolicyEvalTrace{
		PolicyHost:  strings.TrimSpace(f.SNI),
		DstPort:     f.DstPort,
		DstResolved: ips,
	}
}

func connectSkipRuleTypes(ctx context.Context) map[RuleType]bool {
	if observe.ConnectPortOnlyACL(ctx) {
		// RuleAll не пропускаем: финальный http_access allow all должен разрешить порт,
		// если ни одно PORT-правило не сработало (tunnel → relay, иначе MITM Forbidden).
		return map[RuleType]bool{
			RuleSNI:  true,
			RulePath: true,
			RuleSRC:  true,
			RuleDST:  true,
		}
	}
	if observe.DeferPortACLAtConnect(ctx) {
		return map[RuleType]bool{RulePORT: true}
	}
	return nil
}
