package acl

// squidFastPath — специализированный hot path для частой формы политики:
// http_access deny <dstdomain-list>
// http_access allow all
type squidFastPath struct {
	denyDst      *sniIndex
	denyRuleRef  string
	allowRuleRef string
}

func buildSquidFastPath(access []SquidHTTPAccess) squidFastPath {
	if len(access) != 2 {
		return squidFastPath{}
	}
	deny := access[0]
	allow := access[1]
	if deny.Action != ActionDeny || allow.Action != ActionAllow {
		return squidFastPath{}
	}
	if len(deny.Clauses) != 1 || len(allow.Clauses) != 1 {
		return squidFastPath{}
	}
	if deny.Clauses[0].Negated || allow.Clauses[0].Negated {
		return squidFastPath{}
	}
	dst, ok := deny.Clauses[0].Matcher.(SquidMatchPatterns)
	if !ok || dst.RuleType != RuleSNI || dst.sniIndex == nil {
		return squidFastPath{}
	}
	if _, ok := allow.Clauses[0].Matcher.(SquidMatchAll); !ok {
		return squidFastPath{}
	}
	return squidFastPath{
		denyDst:      dst.sniIndex,
		denyRuleRef:  deny.RuleRef,
		allowRuleRef: allow.RuleRef,
	}
}

func (fp squidFastPath) active() bool {
	return fp.denyDst != nil && fp.allowRuleRef != ""
}

func (fp squidFastPath) decide(f RequestFields, _ SquidPhase, _ *EngineTiming, skipRuleTypes map[RuleType]bool) (allow bool, ruleRef string, handled bool) {
	if !fp.active() {
		return false, "", false
	}
	if skipRuleTypes != nil && skipRuleTypes[RuleSNI] {
		return false, "", false
	}
	sni := requestSNILower(f)
	if sni != "" && fp.denyDst.hasMatch(sni) {
		return false, fp.denyRuleRef, true
	}
	return true, fp.allowRuleRef, true
}
