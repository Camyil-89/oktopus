package acl

import (
	"oktopus/internal/proxy/auth"
)

const (
	// SystemRuleDefaultDeny — нет совпадения с правилами или пустой ACL.
	SystemRuleDefaultDeny = "system_default_deny"
)

// DecideWithRuleRef возвращает итог ACL и текст сработавшей строки (http_access allow …, ssl_verify …, system_*).
func (e *Engine) DecideWithRuleRef(id auth.Identity, f RequestFields) (allow bool, ruleRef string) {
	allow, ruleRef, _ = e.decideWithRuleRef(id, f, nil, nil)
	return allow, ruleRef
}

func (e *Engine) decideWithRuleRef(id auth.Identity, f RequestFields, tm *EngineTiming, skipRuleTypes map[RuleType]bool) (allow bool, ruleRef string, matched RuleType) {
	f = normalizeRequestFields(f)
	if e == nil || len(e.squidAccess) == 0 {
		return false, SystemRuleDefaultDeny, ""
	}
	phase := squidPhaseFromSkip(skipRuleTypes)
	allow, ruleRef = e.decideSquid(id, f, phase, tm, skipRuleTypes)
	return allow, ruleRef, ""
}
