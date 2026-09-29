package acl

import (
	"strings"

	"oktopus/internal/proxy/auth"
	"oktopus/internal/proxy/ratelimit"
)

// Приоритет проверки clause в http_access (port → src → dstdomain → url_regex → ldap → all).
const (
	SquidPriorityPort      = 0
	SquidPrioritySrc       = 1
	SquidPriorityDst       = 2
	SquidPriorityDstDomain = 3
	SquidPriorityURLRegex  = 4
	SquidPriorityLDAP      = 5
	SquidPriorityAll       = 6
)

// SquidMatcher — сопоставление одного именованного acl.
type SquidMatcher interface {
	Matches(id auth.Identity, f RequestFields, phase SquidPhase) bool
}

// SquidPhase — этап обработки запроса.
type SquidPhase int

const (
	SquidPhaseConnect SquidPhase = 0
	SquidPhaseHTTP    SquidPhase = 1
)

// CompiledPatternPublic — скомпилированный шаблон для squid acl.
type CompiledPatternPublic struct {
	inner compiledPattern
	typ   RuleType
}

// CompilePatternLine компилирует одну строку шаблона (для squid acl).
func CompilePatternLine(rt RuleType, line string) (CompiledPatternPublic, error) {
	cp, err := compilePattern(rt, strings.TrimSpace(line), nil)
	if err != nil {
		return CompiledPatternPublic{}, err
	}
	return CompiledPatternPublic{inner: cp, typ: rt}, nil
}

// SquidMatchAll — acl type all.
type SquidMatchAll struct{}

func (SquidMatchAll) Matches(_ auth.Identity, _ RequestFields, _ SquidPhase) bool {
	return true
}

// SquidMatchLDAPGroups — acl type ldap_group (OR по группам).
type SquidMatchLDAPGroups struct {
	Groups []string
}

func (m SquidMatchLDAPGroups) Matches(id auth.Identity, _ RequestFields, _ SquidPhase) bool {
	for _, want := range m.Groups {
		for _, g := range id.Groups {
			if strings.EqualFold(strings.TrimSpace(g), strings.TrimSpace(want)) {
				return true
			}
		}
	}
	return false
}

// SquidMatchPatterns — OR по строкам внутри одного acl.
type SquidMatchPatterns struct {
	RuleType RuleType
	Patterns []CompiledPatternPublic

	sniIndex     *sniIndex
	slowPatterns []compiledPattern
	patternTotal int
	sniIndexed   int
	sniRegexp    int
}

// NewSquidMatchPatterns собирает matcher; для RuleSNI строит SNI index.
func NewSquidMatchPatterns(rt RuleType, patterns []CompiledPatternPublic) SquidMatchPatterns {
	m := SquidMatchPatterns{RuleType: rt, Patterns: patterns}
	if rt == RuleSNI {
		finalizeSquidSNIMatchPatterns(&m)
	}
	return m
}

func (m SquidMatchPatterns) patternCount() int {
	if m.patternTotal > 0 {
		return m.patternTotal
	}
	return len(m.Patterns)
}

func (m SquidMatchPatterns) Matches(_ auth.Identity, f RequestFields, phase SquidPhase) bool {
	if m.RuleType == RuleAll || m.RuleType == RulePath {
		if phase == SquidPhaseConnect {
			return false
		}
	}
	if m.RuleType == RuleSNI && m.sniIndex != nil {
		if sni := requestSNILower(f); sni != "" && m.sniIndex.hasMatch(sni) {
			return true
		}
		for _, p := range m.slowPatterns {
			if patternMatches(RuleSNI, p, f) {
				return true
			}
		}
		return false
	}
	for _, p := range m.Patterns {
		if patternMatches(m.RuleType, p.inner, f) {
			return true
		}
	}
	return false
}

// SquidAccessClause — одно условие в строке http_access.
type SquidAccessClause struct {
	Matcher  SquidMatcher
	Negated  bool
	Priority int
}

// SquidHTTPAccess — одна строка http_access.
type SquidHTTPAccess struct {
	LineNo  int
	Action  Action
	Clauses []SquidAccessClause
	RuleRef string
}

// SquidSSLVerify — одна строка ssl_verify (проверка TLS к origin в MITM).
type SquidSSLVerify struct {
	LineNo     int
	SkipVerify bool // true для skip, false для require
	Clauses    []SquidAccessClause
	RuleRef    string
}

// NewSquidEngine создаёт engine только с squid http_access.
func NewSquidEngine(access []SquidHTTPAccess, sslVerify []SquidSSLVerify, delayAccess []SquidDelayAccess, rateLimit *ratelimit.Runtime) *Engine {
	return &Engine{
		squidAccess:      access,
		squidSSLVerify:   sslVerify,
		squidDelayAccess: delayAccess,
		rateLimit:        rateLimit,
		squidFast:        buildSquidFastPath(access),
	}
}

func (e *Engine) hasSquid() bool {
	return e != nil && len(e.squidAccess) > 0
}

func squidPhaseFromSkip(skip map[RuleType]bool) SquidPhase {
	if skip != nil && skip[RuleSNI] && skip[RulePath] && skip[RuleAll] {
		return SquidPhaseConnect
	}
	return SquidPhaseHTTP
}

func (e *Engine) decideSquid(id auth.Identity, f RequestFields, phase SquidPhase, tm *EngineTiming, skipRuleTypes map[RuleType]bool) (allow bool, ruleRef string) {
	if allow, ruleRef, ok := e.squidFast.decide(f, phase, tm, skipRuleTypes); ok {
		return allow, ruleRef
	}
	for _, line := range e.squidAccess {
		if lineMatches(line, id, f, phase, skipRuleTypes) {
			return line.Action == ActionAllow, line.RuleRef
		}
	}
	return false, SystemRuleDefaultDeny
}

func squidMatcherRuleType(m SquidMatcher) RuleType {
	switch m := m.(type) {
	case SquidMatchPatterns:
		return m.RuleType
	case SquidMatchAll:
		return RuleAll
	default:
		return ""
	}
}

func lineMatches(line SquidHTTPAccess, id auth.Identity, f RequestFields, phase SquidPhase, skipRuleTypes map[RuleType]bool) bool {
	evaluated := false
	for _, cl := range line.Clauses {
		rt := squidMatcherRuleType(cl.Matcher)
		if skipRuleTypes != nil && rt != "" && skipRuleTypes[rt] {
			continue
		}
		evaluated = true
		ok := cl.Matcher.Matches(id, f, phase)
		if cl.Negated {
			ok = !ok
		}
		if !ok {
			return false
		}
	}
	return evaluated
}

// DecideUpstreamTLSVerify — первое совпавшее правило ssl_verify; иначе проверять сертификат (require).
func (e *Engine) DecideUpstreamTLSVerify(id auth.Identity, f RequestFields) (verify bool, ruleRef string) {
	verify = true
	if e == nil || len(e.squidSSLVerify) == 0 {
		return true, ""
	}
	phase := SquidPhaseHTTP
	for _, line := range e.squidSSLVerify {
		if lineMatchesSSLVerify(line, id, f, phase) {
			return !line.SkipVerify, line.RuleRef
		}
	}
	return true, ""
}

func lineMatchesSSLVerify(line SquidSSLVerify, id auth.Identity, f RequestFields, phase SquidPhase) bool {
	tmp := SquidHTTPAccess{Clauses: line.Clauses}
	return lineMatches(tmp, id, f, phase, nil)
}
