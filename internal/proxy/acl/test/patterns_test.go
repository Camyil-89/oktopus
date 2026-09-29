package acl_test

import (
	"testing"

	"oktopus/internal/proxy/acl"
	"oktopus/internal/proxy/acl/squid"
	"oktopus/internal/proxy/auth"
)

func TestRuleRegexsMultiline(t *testing.T) {
	e := mustEngineFromSquid(t, `
acl all all
http_access deny blocked
http_access allow all
`, squid.ListInput{
		Name:     "blocked",
		ListType: "dstdomain",
		Body:     "blocked.evil\nother.bad\n",
	})
	id := auth.Identity{Username: "u"}
	if !e.Decide(id, acl.RequestFields{SNI: "ok.com", Path: "/"}) {
		t.Fatal("ok allow")
	}
	if e.Decide(id, acl.RequestFields{SNI: "blocked.evil", Path: "/"}) {
		t.Fatal("deny blocked.evil")
	}
	if e.Decide(id, acl.RequestFields{SNI: "other.bad", Path: "/"}) {
		t.Fatal("deny other.bad")
	}
	if e.HTTPAccessCount() != 2 {
		t.Fatalf("want 2 http_access lines, got %d", e.HTTPAccessCount())
	}
}

func TestRuleRegexsTextBlockComments(t *testing.T) {
	e := mustEngineFromSquid(t, `
http_access deny blocked
`, squid.ListInput{
		Name:     "blocked",
		ListType: "dstdomain",
		Body:     "# comment\nalpha.test\n\nbeta.test\n",
	})
	id := auth.Identity{Username: "u"}
	if e.Decide(id, acl.RequestFields{SNI: "alpha.test", Path: "/"}) {
		t.Fatal("deny alpha")
	}
	if e.Decide(id, acl.RequestFields{SNI: "beta.test", Path: "/"}) {
		t.Fatal("deny beta")
	}
}

func TestRuleRegexsOrderVsNextRule(t *testing.T) {
	e := mustEngineFromSquid(t, `
acl deny_one dstdomain conflict.com
http_access allow allow_list
http_access deny deny_one
`, squid.ListInput{
		Name:     "allow_list",
		ListType: "dstdomain",
		Body:     "conflict.com\n",
	})
	id := auth.Identity{Username: "u"}
	if !e.Decide(id, acl.RequestFields{SNI: "conflict.com", Path: "/"}) {
		t.Fatal("first http_access (allow list) must win")
	}
}

func TestRuleRegexsBulkSamePerfShape(t *testing.T) {
	body := benchHostLines(100)
	e := mustEngineFromSquid(t, `
http_access deny deny_hosts
`, squid.ListInput{Name: "deny_hosts", ListType: "dstdomain", Body: body})
	if e.HTTPAccessCount() != 1 {
		t.Fatal("one http_access line")
	}
	st := e.SNIPatternStats()
	if st.Indexed != 100 {
		t.Fatalf("expected sni index stats indexed=%d", st.Indexed)
	}
	id := auth.Identity{Username: "u"}
	if e.Decide(id, acl.RequestFields{SNI: "host50.example.com", Path: "/"}) {
		t.Fatal("deny hit")
	}
}
