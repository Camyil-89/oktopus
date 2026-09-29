package acl_test

import (
	"testing"

	"oktopus/internal/proxy/acl"
	"oktopus/internal/proxy/auth"
)

func TestExplainAllowedMatchesDecide(t *testing.T) {
	policies := []string{
		`
acl bad dstdomain blocked.evil
acl all all
http_access deny bad
http_access allow all
`,
		`
acl deny_other dstdomain other.com
acl allow_wild dstdomain *.allowed.com
acl all all
http_access deny deny_other
http_access allow allow_wild
http_access deny all
`,
		`
acl devs ldap_group devs
acl bad dstdomain blocked.evil
acl all all
http_access deny bad
http_access allow devs
http_access deny all
`,
	}
	fields := []acl.RequestFields{
		{SNI: "blocked.evil", Path: "/"},
		{SNI: "ok.com", Path: "/"},
		{SNI: "app.allowed.com", Path: "/api/x"},
		{SNI: "  host.com  ", Path: ""},
	}
	idents := []auth.Identity{
		{Username: "u"},
		{Username: "u", Groups: []string{"devs"}},
		{Username: "u", Groups: []string{"other"}},
	}
	for _, cfg := range policies {
		e := mustEngineFromSquid(t, cfg)
		for _, id := range idents {
			for _, f := range fields {
				got := e.Decide(id, f)
				ex := e.Explain(id, f)
				if ex.Allowed != got {
					t.Fatalf("Explain.Allowed=%v Decide=%v id=%+v fields=%+v", ex.Allowed, got, id, f)
				}
			}
		}
	}
}
