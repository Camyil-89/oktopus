package acl_test

import (
	"testing"

	"oktopus/internal/proxy/acl"
	"oktopus/internal/proxy/auth"
)

func TestSquidFastPathDenyDstAllowAll(t *testing.T) {
	eng := mustEngineFromSquid(t, `
acl blocked dstdomain .evil.test
acl all all
http_access deny blocked
http_access allow all
`)
	id := auth.Identity{}
	if eng.Decide(id, acl.RequestFields{SNI: "evil.test"}) {
		t.Fatal("expected deny")
	}
	if !eng.Decide(id, acl.RequestFields{SNI: "ok.test"}) {
		t.Fatal("expected allow")
	}
}
