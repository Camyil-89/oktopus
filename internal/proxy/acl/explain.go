package acl

import (
	"strings"

	"oktopus/internal/proxy/auth"
)

// ExplainStepKind — этап разбора решения ACL.
type ExplainStepKind string

const (
	StepNoEngine    ExplainStepKind = "no_engine"
	StepRuleMatched ExplainStepKind = "rule_matched"
	StepDefaultDeny ExplainStepKind = "default_deny"
)

// ExplainStep — один шаг цепочки до итога.
type ExplainStep struct {
	Kind    ExplainStepKind `json:"kind"`
	Message string          `json:"message,omitempty"`
}

// ExplainResult — результат проверки запроса против опубликованных правил.
type ExplainResult struct {
	Allowed bool          `json:"allowed"`
	Steps   []ExplainStep `json:"steps"`
}

// Explain симулирует Decide и возвращает цепочку решений.
func (e *Engine) Explain(id auth.Identity, f RequestFields) ExplainResult {
	if e == nil || len(e.squidAccess) == 0 {
		return ExplainResult{
			Allowed: false,
			Steps: []ExplainStep{{
				Kind:    StepNoEngine,
				Message: "нет опубликованных правил — default deny",
			}},
		}
	}
	allowed, ref := e.decideSquid(id, normalizeRequestFields(f), SquidPhaseHTTP, nil, nil)
	if allowed {
		return ExplainResult{
			Allowed: true,
			Steps: []ExplainStep{{
				Kind:    StepRuleMatched,
				Message: "squid " + ref,
			}},
		}
	}
	return ExplainResult{
		Allowed: false,
		Steps: []ExplainStep{{
			Kind:    StepDefaultDeny,
			Message: "нет совпадений http_access — default deny (ref " + ref + ")",
		}},
	}
}

func normalizeRequestFields(f RequestFields) RequestFields {
	f.SNI = strings.TrimSpace(f.SNI)
	if f.Path == "" {
		f.Path = "/"
	}
	if f.SNI != "" {
		f.sniLower = normalizeHostForMatch(f.SNI)
	}
	return f
}
