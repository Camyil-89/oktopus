package acl_test

import (
	"testing"

	"oktopus/internal/proxy/acl"
	"oktopus/internal/proxy/auth"
)

func TestExplainRuleMatched(t *testing.T) {
	e := mustEngineFromSquid(t, `
acl bad dstdomain blocked.evil
acl all all
http_access deny bad
http_access allow all
`)
	id := auth.Identity{Username: "u"}
	ex := e.Explain(id, acl.RequestFields{SNI: "blocked.evil", Path: "/"})
	if ex.Allowed {
		t.Fatal("expected deny")
	}
	if len(ex.Steps) != 1 || ex.Steps[0].Kind != acl.StepDefaultDeny {
		t.Fatalf("steps: %+v", ex.Steps)
	}
	ex2 := e.Explain(id, acl.RequestFields{SNI: "ok.com", Path: "/"})
	if !ex2.Allowed || ex2.Steps[0].Kind != acl.StepRuleMatched {
		t.Fatalf("allow steps: %+v", ex2.Steps)
	}
}
