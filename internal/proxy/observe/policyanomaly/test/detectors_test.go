package policyanomaly_test

import (
	"testing"

	"oktopus/internal/proxy/observe/policyanomaly"
)

func TestRegisteredKinds(t *testing.T) {
	kinds := policyanomaly.RegisteredKinds()
	if len(kinds) < 5 {
		t.Fatalf("expected at least 5 detectors, got %v", kinds)
	}
}

func TestEvaluate_evilSNI(t *testing.T) {
	p := policyanomaly.Evaluate(policyanomaly.Input{
		ConnectHostPort: "localhost:9443",
		TLSClientSNI:    "internal.blocked",
		PolicyHost:      "localhost",
		HTTPHostHeader:  "localhost",
	})
	r, ok := p[policyanomaly.KindHostSNIMismatch]
	if !ok || !r.Detect {
		t.Fatalf("expected host_sni_mismatch detect, got %+v", p)
	}
	if r.TLSClientSNI != "internal.blocked" || r.PolicyHost != "localhost" {
		t.Fatalf("fields: %+v", r)
	}
}

func TestEvaluate_evilHost(t *testing.T) {
	p := policyanomaly.Evaluate(policyanomaly.Input{
		ConnectHostPort: "localhost:9443",
		TLSClientSNI:    "localhost",
		PolicyHost:      "127.0.0.1",
		HTTPHostHeader:  "127.0.0.1:9555",
	})
	if !p[policyanomaly.KindHostSNIMismatch].Detect {
		t.Fatal("expected host_sni_mismatch")
	}
}

func TestEvaluate_aligned(t *testing.T) {
	p := policyanomaly.Evaluate(policyanomaly.Input{
		ConnectHostPort: "localhost:9443",
		TLSClientSNI:    "localhost",
		PolicyHost:      "localhost",
		HTTPHostHeader:  "localhost",
	})
	if p[policyanomaly.KindHostSNIMismatch].Detect {
		t.Fatalf("unexpected mismatch: %+v", p)
	}
}

func TestEvaluate_allKindsHaveDetectField(t *testing.T) {
	p := policyanomaly.Evaluate(policyanomaly.Input{})
	for _, kind := range policyanomaly.RegisteredKinds() {
		if _, ok := p[kind]; !ok {
			t.Fatalf("missing kind %q", kind)
		}
	}
}

func TestEvaluate_urlHostMismatch(t *testing.T) {
	p := policyanomaly.Evaluate(policyanomaly.Input{
		URLHost:        "internal.blocked",
		HTTPHostHeader: "localhost",
		PolicyHost:     "internal.blocked",
	})
	if !p[policyanomaly.KindURLHostMismatch].Detect {
		t.Fatalf("got %+v", p)
	}
	if p[policyanomaly.KindHostSNIMismatch].Detect {
		t.Fatalf("host_sni should be false: %+v", p)
	}
}

func TestEvaluate_connectPortMismatch(t *testing.T) {
	p := policyanomaly.Evaluate(policyanomaly.Input{
		ConnectHostPort: "localhost:9443",
		ConnectPort:     9443,
		HTTPPort:        9778,
		PolicyHost:      "localhost",
	})
	if !p[policyanomaly.KindConnectPortMismatch].Detect {
		t.Fatalf("got %+v", p)
	}
}

func TestEvaluate_connectLiteralIP(t *testing.T) {
	p := policyanomaly.Evaluate(policyanomaly.Input{
		ConnectHostPort: "127.0.0.1:9777",
		PolicyHost:      "127.0.0.1",
	})
	if !p[policyanomaly.KindConnectLiteralIP].Detect {
		t.Fatalf("got %+v", p)
	}
}

func TestEvaluate_dstResolvePrivate(t *testing.T) {
	p := policyanomaly.Evaluate(policyanomaly.Input{
		PolicyHost:  "9667.127.0.0.1.sslip.io",
		DstResolved: []string{"127.0.0.1"},
	})
	if !p[policyanomaly.KindDstResolvePrivate].Detect {
		t.Fatalf("got %+v", p)
	}
}

func TestPayload_AnyDetected(t *testing.T) {
	p := policyanomaly.Payload{
		policyanomaly.KindHostSNIMismatch: {Detect: false},
	}
	if p.AnyDetected() {
		t.Fatal("expected false")
	}
	p[policyanomaly.KindURLHostMismatch] = policyanomaly.CheckResult{Detect: true}
	if !p.AnyDetected() {
		t.Fatal("expected true")
	}
}
