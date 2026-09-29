package observe

import (
	"context"
	"time"
)

type aclAllowPassedKey struct{}

// ACLDecision — результат ACL для отложенной записи в журнал (после инспекции).
type ACLDecision struct {
	RuleRef string
	Spend   time.Duration
}

type aclDecisionKey struct{}

// WithACLAllowPassed помечает, что ACL уже разрешил запрос (перед инспекцией).
func WithACLAllowPassed(ctx context.Context) context.Context {
	return context.WithValue(ctx, aclAllowPassedKey{}, true)
}

// ACLAllowPassed — true после успешного ACL allow на этом запросе.
func ACLAllowPassed(ctx context.Context) bool {
	v, _ := ctx.Value(aclAllowPassedKey{}).(bool)
	return v
}

// WithACLDecision сохраняет правило и время ACL allow для финальной записи журнала.
func WithACLDecision(ctx context.Context, d ACLDecision) context.Context {
	return context.WithValue(ctx, aclDecisionKey{}, d)
}

// ACLDecisionFrom читает решение ACL после allow.
func ACLDecisionFrom(ctx context.Context) (ACLDecision, bool) {
	d, ok := ctx.Value(aclDecisionKey{}).(ACLDecision)
	return d, ok
}
