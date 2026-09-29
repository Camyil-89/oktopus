package acl

import (
	"net"
	"strings"

	"oktopus/internal/proxy/auth"
)

// RequestFields — данные для сопоставления правил.
type RequestFields struct {
	SNI         string
	Path        string
	SrcIP       net.IP
	DstIP       net.IP
	DstResolved []net.IP // DNS для acl dst (CONNECT/HTTP hostname)
	DstPort     int
	sniLower    string
}

func requestSNILower(f RequestFields) string {
	if f.sniLower != "" {
		return f.sniLower
	}
	sni := strings.TrimSpace(f.SNI)
	if sni == "" {
		return ""
	}
	return normalizeHostForMatch(sni)
}

// Decide возвращает true, если запрос разрешён по первой подходящей строке http_access.
// Без правил или без совпадения — запрет (default deny).
func (e *Engine) Decide(id auth.Identity, f RequestFields) bool {
	allow, _ := e.DecideWithRuleRef(id, f)
	return allow
}

func patternMatches(typ RuleType, p compiledPattern, f RequestFields) bool {
	sni := strings.TrimSpace(f.SNI)
	path := f.Path
	if path == "" {
		path = "/"
	}
	switch typ {
	case RuleAll:
		return p.re != nil && p.re.MatchString(sni+path)
	case RuleSNI:
		if sni == "" {
			return false
		}
		sniLower := requestSNILower(f)
		if p.sniFast != sniFastNone {
			return domainPatternMatches(p.sniFast, p.sniHost, sniLower)
		}
		return p.re != nil && p.re.MatchString(sni)
	case RulePath:
		return p.re != nil && p.re.MatchString(path)
	case RuleSRC:
		return ipPatternMatches(p, f.SrcIP)
	case RuleDST:
		return dstPatternMatches(p, f)
	case RulePORT:
		return portPatternMatches(p, DstPortForMatch(f))
	default:
		return false
	}
}
