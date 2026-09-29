package metrics

import (
	"context"

	"oktopus/internal/proxy/observe"
)

// ObserveHTTPPolicyFromACLContext — ACL-only политика для HTTP allow без inspect (tunnel).
func ObserveHTTPPolicyFromACLContext(ctx context.Context) {
	dec, ok := observe.ACLDecisionFrom(ctx)
	if !ok {
		return
	}
	ObservePolicyTotal(dec.Spend)
}
