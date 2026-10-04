package observe_test

import (
	"testing"

	"oktopus/internal/proxy/observe"
)

func TestDetectPolicyNameMismatch_evilSNI(t *testing.T) {
	note := observe.DetectPolicyNameMismatch("localhost:9443", "internal.blocked", "localhost", "localhost")
	if note == nil {
		t.Fatal("expected mismatch")
	}
	if note.Kind != observe.PolicyAnomalyHostSNIMismatch {
		t.Fatalf("kind: %q", note.Kind)
	}
	if note.TLSClientSNI != "internal.blocked" || note.PolicyHost != "localhost" {
		t.Fatalf("fields: %+v", note)
	}
}

func TestDetectPolicyNameMismatch_evilHost(t *testing.T) {
	note := observe.DetectPolicyNameMismatch("localhost:9443", "localhost", "127.0.0.1", "127.0.0.1:9555")
	if note == nil {
		t.Fatal("expected mismatch")
	}
}

func TestDetectPolicyNameMismatch_aligned(t *testing.T) {
	if observe.DetectPolicyNameMismatch("localhost:9443", "localhost", "localhost", "localhost") != nil {
		t.Fatal("expected nil")
	}
}
