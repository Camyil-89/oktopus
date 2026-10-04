package observe_test

import (
	"testing"

	"oktopus/internal/proxy/observe"
)

func TestObserveReexportsPolicyAnomalyKinds(t *testing.T) {
	kinds := observe.RegisteredPolicyAnomalyKinds()
	if len(kinds) < 5 {
		t.Fatalf("got %v", kinds)
	}
	if observe.PolicyAnomalyHostSNIMismatch != "host_sni_mismatch" {
		t.Fatal(observe.PolicyAnomalyHostSNIMismatch)
	}
}
