package acl_test

import (
	"context"
	"testing"

	"oktopus/internal/proxy/acl"
	"oktopus/internal/proxy/auth"
	"oktopus/internal/proxy/observe"
)

func TestSquidConnectPortOnlyEvaluatesPORT(t *testing.T) {
	eng := mustEngineFromSquid(t, `
acl badport port 666
acl all all
http_access deny badport
http_access allow all
`)
	f := acl.RequestFields{SNI: "ok.test", DstPort: 666}
	if eng.Decide(auth.Identity{}, f) {
		t.Fatal("expected deny on full HTTP decision for port 666")
	}
	h := eng.Hooks(nil)
	ctx := observe.WithConnectPortOnlyACL(context.Background())
	dec := h.OnConnect(ctx, "ok.test:666")
	if dec.Allow {
		t.Fatal("port-only CONNECT pass must deny port 666")
	}
}

func TestSquidDeferPortAtConnectSkipsPORT(t *testing.T) {
	eng := mustEngineFromSquid(t, `
acl badport port 666
acl all all
http_access deny badport
http_access allow all
`)
	h := eng.Hooks(nil)
	ctx := observe.WithDeferPortACLAtConnect(context.Background())
	dec := h.OnConnect(ctx, "ok.test:666")
	if !dec.Allow {
		t.Fatal("CONNECT with defer PORT must allow until HTTP")
	}
}

func TestSquidFastPathInactiveOnPortOnlyConnect(t *testing.T) {
	eng := mustEngineFromSquid(t, `
acl blocked dstdomain .evil.test
acl all all
http_access deny blocked
http_access allow all
`)
	h := eng.Hooks(nil)
	ctx := observe.WithConnectPortOnlyACL(context.Background())
	dec := h.OnConnect(ctx, "evil.test:443")
	if !dec.Allow {
		t.Fatal("port-only CONNECT must not apply dstdomain deny (handled on first CONNECT / HTTP)")
	}
}
