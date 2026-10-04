package accesslog

import (
	"context"
	"net"
	stdhttp "net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"oktopus/internal/proxy/auth"
	"oktopus/internal/proxy/observe"
)

// Action — итог: 0 DENY, 1 ALLOW.
type Action uint8

const (
	ActionDeny  Action = 0
	ActionAllow Action = 1
)

const (
	// RuleRefAuthFail — неудачная proxy-auth до ACL (decision_rule_ref, action DENY).
	RuleRefAuthFail = "system_auth_fail"
	// RuleRefDefaultDeny — default deny ACL без совпадения (decision_rule_ref).
	RuleRefDefaultDeny = "system_default_deny"
	// RuleRefInspectError — сбой Lua-инспекции (decision_rule_ref, denied_by inspect).
	RuleRefInspectError = "system_inspect_error"
	// RuleRefGatewayError — 502 Bad Gateway после allow (decision_rule_ref, denied_by gateway).
	RuleRefGatewayError = "system_gateway_error"
)

const (
	DeniedByACL     = "acl"
	DeniedByInspect = "inspect"
	DeniedByGateway = "gateway"
)

// PolicyNoteInspectRuleID — синтетический inspect_rule_id для KV policy_anomaly (не правило inspect).
var PolicyNoteInspectRuleID = uuid.MustParse("00000000-0000-4000-8000-000000000001")

// Entry — одна запись решения ACL по запросу или CONNECT.
type Entry struct {
	ID                 uuid.UUID // если задан — id строки в proxy_access_log (иначе генерируется при flush)
	At                 time.Time
	RuleRef            string // UUID инспекции, текст http_access/ssl_verify или system_* → decision_rule_ref
	SourceAddress      string
	DestinationAddress string
	User               *string // nil без proxy-auth
	SpendTime          time.Duration
	FullURLRequest     string
	Action             Action
	ACLRuleRef         string
	InspectRuleRef     string
	DeniedBy           string // DeniedByACL | DeniedByInspect | ""
	InspectError       string
	GatewayErrorType   string // тип upstream-ошибки (extra gateway_error.type)
	InspectRuleLogs    map[string]map[string]any // id правила → payload ctx:log
	PolicyAnomaly observe.PolicyAnomalyPayload // extra policy_anomaly (kind → {detect, …})
}

// Recorder принимает записи; реализация не должна блокировать hot path надолго.
type Recorder interface {
	Record(Entry)
}

// NopRecorder отбрасывает записи.
type NopRecorder struct{}

func (NopRecorder) Record(Entry) {}

func actionFromAllow(allow bool) Action {
	if allow {
		return ActionAllow
	}
	return ActionDeny
}

func userFromContext(ctx context.Context) *string {
	id, ok := auth.IdentityFromContext(ctx)
	if !ok || id.Username == "" {
		return nil
	}
	u := id.Username
	return &u
}

func sourceAddress(ctx context.Context, req *stdhttp.Request) string {
	if req != nil && req.RemoteAddr != "" {
		return hostPortOnly(req.RemoteAddr)
	}
	if a := observe.RemoteAddrFromContext(ctx); a != "" {
		return hostPortOnly(a)
	}
	return ""
}

func hostPortOnly(hostPort string) string {
	hostPort = strings.TrimSpace(hostPort)
	if hostPort == "" {
		return ""
	}
	host, port, err := net.SplitHostPort(hostPort)
	if err != nil {
		return hostPort
	}
	if port == "" {
		return host
	}
	return net.JoinHostPort(host, port)
}

// HTTPEntry собирает поля для обычного proxy HTTP-запроса.
func HTTPEntry(ctx context.Context, req *stdhttp.Request, allow bool, spend time.Duration, ruleRef string) Entry {
	dest := ""
	full := ""
	if req != nil && req.URL != nil {
		full = req.URL.String()
		if req.URL.Host != "" {
			dest = req.URL.Host
		}
	}
	if dest == "" && req != nil {
		dest = req.Host
	}
	e := Entry{
		SourceAddress:      sourceAddress(ctx, req),
		DestinationAddress: dest,
		User:               userFromContext(ctx),
		SpendTime:          spend,
		FullURLRequest:     full,
		Action:             actionFromAllow(allow),
		RuleRef:            ruleRef,
	}
	if !allow && ruleRef != "" {
		e.DeniedBy = DeniedByACL
		e.ACLRuleRef = ruleRef
	}
	attachPolicyAnomaliesHTTP(&e, ctx, req)
	return e
}

func attachPolicyAnomaliesHTTP(e *Entry, ctx context.Context, req *stdhttp.Request) {
	if e == nil || req == nil {
		return
	}
	reqCtx := req.Context()
	tools := observe.RequestToolsFor(ctx, req)
	urlHost := ""
	if req.URL != nil {
		urlHost = req.URL.Host
	}
	trace, _ := observe.PolicyEvalTraceFromContext(ctx)
	if !traceHasData(trace) {
		trace, _ = observe.PolicyEvalTraceFromContext(reqCtx)
	}
	in := observe.PolicyAnomalyDetectInput{
		ConnectHostPort: observe.CONNECTDestHostPortFromContext(reqCtx),
		TLSClientSNI:    observe.MITMClientHelloSNIFromContext(reqCtx),
		PolicyHost:      observe.PolicyHostFromRequest(req, tools.SNI()),
		HTTPHostHeader:  req.Host,
		URLHost:         urlHost,
		ConnectPort:     observe.TracePort(observe.CONNECTDestHostPortFromContext(reqCtx)),
		HTTPPort:        observe.HTTPPortFromRequest(req),
		DstResolved:     trace.DstResolved,
	}
	e.PolicyAnomaly = observe.EvaluatePolicyAnomaly(in)
}

func attachPolicyAnomaliesConnect(e *Entry, ctx context.Context, connectHostPort string) {
	if e == nil {
		return
	}
	trace, _ := observe.PolicyEvalTraceFromContext(ctx)
	in := observe.PolicyAnomalyDetectInput{
		ConnectHostPort: connectHostPort,
		PolicyHost:      trace.PolicyHost,
		ConnectPort:     observe.TracePort(connectHostPort),
		DstResolved:     trace.DstResolved,
	}
	if in.PolicyHost == "" {
		in.PolicyHost = normalizeConnectPolicyHost(connectHostPort)
	}
	e.PolicyAnomaly = observe.EvaluatePolicyAnomaly(in)
}

func traceHasData(t observe.PolicyEvalTrace) bool {
	return strings.TrimSpace(t.PolicyHost) != "" || len(t.DstResolved) > 0 || t.DstPort != 0
}

func normalizeConnectPolicyHost(hostPort string) string {
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

// FinalHTTPEntryAfterACL — одна запись после ACL allow (инспекция отсутствует или пропущена).
func FinalHTTPEntryAfterACL(ctx context.Context, req *stdhttp.Request, aclRuleRef string, aclSpend time.Duration) Entry {
	e := HTTPEntry(ctx, req, true, aclSpend, aclRuleRef)
	e.ACLRuleRef = aclRuleRef
	return e
}

// InspectHTTPOutcome — итог Lua-инспекции для журнала.
type InspectHTTPOutcome struct {
	Spend    time.Duration
	Matched  bool
	Deny     bool
	RuleID   string
	Err      error
	RuleLogs map[string]map[string]any
}

// FinalHTTPEntryAfterInspect — одна запись: ACL allow + итог инспекции (без дубля с ACL).
func FinalHTTPEntryAfterInspect(ctx context.Context, req *stdhttp.Request, aclRuleRef string, aclSpend time.Duration, out InspectHTTPOutcome) Entry {
	totalSpend := aclSpend + out.Spend
	e := HTTPEntry(ctx, req, true, totalSpend, aclRuleRef)
	e.ACLRuleRef = aclRuleRef

	if out.Err != nil {
		e.Action = ActionDeny
		e.DeniedBy = DeniedByInspect
		e.RuleRef = RuleRefInspectError
		e.InspectError = out.Err.Error()
		return e
	}
	if out.Matched && out.Deny {
		e.Action = ActionDeny
		e.DeniedBy = DeniedByInspect
		e.RuleRef = out.RuleID
		e.InspectRuleRef = out.RuleID
		e.InspectRuleLogs = out.RuleLogs
		return e
	}
	e.Action = ActionAllow
	e.RuleRef = aclRuleRef
	if out.Matched && out.RuleID != "" {
		e.InspectRuleRef = out.RuleID
	}
	e.InspectRuleLogs = out.RuleLogs
	return e
}

// ConnectEntry — CONNECT / TLS-туннель (полный URL условный: https://host/).
func ConnectEntry(ctx context.Context, hostPort string, allow bool, spend time.Duration, ruleRef string) Entry {
	hostPort = strings.TrimSpace(hostPort)
	full := ""
	if hostPort != "" {
		full = "https://" + hostPort + "/"
	}
	e := Entry{
		SourceAddress:      sourceAddress(ctx, nil),
		DestinationAddress: hostPort,
		User:               userFromContext(ctx),
		SpendTime:          spend,
		FullURLRequest:     full,
		Action:             actionFromAllow(allow),
		RuleRef:            ruleRef,
	}
	attachPolicyAnomaliesConnect(&e, ctx, hostPort)
	return e
}

// GatewayErrorEntry — 502 после ACL allow: текст ошибки в inspect_error, тип в extra.
func GatewayErrorEntry(ctx context.Context, req *stdhttp.Request, logID uuid.UUID, err error, errType string) Entry {
	msg := ""
	if err != nil {
		msg = err.Error()
	}
	e := HTTPEntry(ctx, req, false, 0, RuleRefGatewayError)
	e.ID = logID
	e.DeniedBy = DeniedByGateway
	e.GatewayErrorType = strings.TrimSpace(errType)
	e.InspectError = msg
	return e
}

// AuthFailEntry — неудачная proxy-авторизация (до ACL), в БД action=DENY.
func AuthFailEntry(ctx context.Context, req *stdhttp.Request, spend time.Duration) Entry {
	dest := ""
	full := ""
	if req != nil {
		if req.Method == stdhttp.MethodConnect {
			dest = strings.TrimSpace(req.Host)
			if dest != "" {
				full = "https://" + dest + "/"
			}
		} else if req.URL != nil {
			full = req.URL.String()
			if req.URL.Host != "" {
				dest = req.URL.Host
			}
		}
		if dest == "" {
			dest = strings.TrimSpace(req.Host)
		}
	}
	var user *string
	if u, _, ok := auth.ProxyBasicCredentials(req); ok && u != "" {
		user = &u
	}
	return Entry{
		SourceAddress:      sourceAddress(ctx, req),
		DestinationAddress: dest,
		User:               user,
		SpendTime:          spend,
		FullURLRequest:     full,
		Action:             ActionDeny,
		RuleRef:            RuleRefAuthFail,
	}
}
