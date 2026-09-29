package acl_test

import (
	"testing"

	"oktopus/internal/proxy/acl"
	"oktopus/internal/proxy/auth"
)

func TestExplainAllowedMatchesDecideSNILiteral(t *testing.T) {
	e := mustEngineFromSquid(t, `
acl deny_other dstdomain other.com
acl allow_wild dstdomain *.allowed.com
acl all all
http_access deny deny_other
http_access allow allow_wild
http_access deny all
`)
	id := auth.Identity{Username: "u"}
	f := acl.RequestFields{SNI: "app.allowed.com", Path: "/"}
	if e.Decide(id, f) != e.Explain(id, f).Allowed {
		t.Fatal("Explain must match Decide")
	}
	if !e.Explain(id, f).Allowed {
		t.Fatal("expected allow")
	}
}
