package observe

import (
	"context"
	"net"
	stdhttp "net/http"
	"strconv"
	"strings"
)

// PolicyEvalTrace — поля ACL после EvaluatePolicy (для policy_anomaly в журнале).
type PolicyEvalTrace struct {
	PolicyHost  string
	DstPort     int
	DstResolved []string
}

type policyEvalTraceKey struct{}

// WithPolicyEvalTrace сохраняет результат enrich/decide для access log.
func WithPolicyEvalTrace(ctx context.Context, t PolicyEvalTrace) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, policyEvalTraceKey{}, t)
}

// PolicyEvalTraceFromContext читает trace ACL.
func PolicyEvalTraceFromContext(ctx context.Context) (PolicyEvalTrace, bool) {
	if ctx == nil {
		return PolicyEvalTrace{}, false
	}
	t, ok := ctx.Value(policyEvalTraceKey{}).(PolicyEvalTrace)
	return t, ok
}

// TracePort извлекает порт из host:port (CONNECT, Host).
func TracePort(hostPort string) int {
	hostPort = strings.TrimSpace(hostPort)
	if hostPort == "" {
		return 0
	}
	_, portStr, err := net.SplitHostPort(hostPort)
	if err != nil {
		return 0
	}
	p, err := strconv.Atoi(portStr)
	if err != nil || p <= 0 || p > 65535 {
		return 0
	}
	return p
}

// HTTPPortFromRequest — порт назначения для HTTP(S) (как acl httpDstPort).
func HTTPPortFromRequest(req *stdhttp.Request) int {
	if req == nil {
		return 0
	}
	if req.URL != nil && req.URL.Host != "" {
		if p := TracePort(req.URL.Host); p != 0 {
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
		if p := TracePort(req.Host); p != 0 {
			return p
		}
	}
	return 0
}
