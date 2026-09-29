package acl_test

import (
	"testing"

	"oktopus/internal/proxy/acl"
	"oktopus/internal/proxy/auth"
)

func TestWildcardApexOnly(t *testing.T) {
	e := mustEngineFromSquid(t, `
acl wild dstdomain *.wild.com
acl all all
http_access deny wild
http_access allow all
`)
	id := auth.Identity{Username: "u"}
	if !e.Decide(id, acl.RequestFields{SNI: "wild.com", Path: "/"}) {
		t.Fatal("apex must not match *.wild.com")
	}
	if e.Decide(id, acl.RequestFields{SNI: "x.wild.com", Path: "/"}) {
		t.Fatal("subdomain must deny")
	}
}

func TestDomainExactSuffixWildcard(t *testing.T) {
	e := mustEngineFromSquid(t, `
acl exact dstdomain exact.only.com
acl suffix dstdomain .suffix.com
acl wild dstdomain *.wild.com
acl all all
http_access deny exact
http_access deny suffix
http_access deny wild
http_access allow all
`)
	id := auth.Identity{Username: "u"}

	if e.Decide(id, acl.RequestFields{SNI: "exact.only.com", Path: "/"}) {
		t.Fatal("exact deny")
	}
	if e.Decide(id, acl.RequestFields{SNI: "suffix.com", Path: "/"}) {
		t.Fatal("suffix root deny")
	}
	if e.Decide(id, acl.RequestFields{SNI: "a.suffix.com", Path: "/"}) {
		t.Fatal("suffix child deny")
	}
	if !e.Decide(id, acl.RequestFields{SNI: "wild.com", Path: "/"}) {
		t.Fatal("wildcard must not match apex")
	}
	if e.Decide(id, acl.RequestFields{SNI: "sub.wild.com", Path: "/"}) {
		t.Fatal("wildcard child deny")
	}
	if !e.Decide(id, acl.RequestFields{SNI: "ok.net", Path: "/"}) {
		t.Fatal("unknown host allow")
	}
}

func TestDomainIDNCyrillic(t *testing.T) {
	e := mustEngineFromSquid(t, `
acl idn dstdomain пример.рф
acl all all
http_access deny idn
http_access allow all
`)
	id := auth.Identity{Username: "u"}
	if e.Decide(id, acl.RequestFields{SNI: "пример.рф", Path: "/"}) {
		t.Fatal("unicode SNI must deny")
	}
	if e.Decide(id, acl.RequestFields{SNI: "xn--e1afmkfd.xn--p1ai", Path: "/"}) {
		t.Fatal("punycode SNI must deny same IDN rule")
	}
	if !e.Decide(id, acl.RequestFields{SNI: "other.com", Path: "/"}) {
		t.Fatal("other host allow")
	}
}

func TestDecideRuleOrderInFile(t *testing.T) {
	e := mustEngineFromSquid(t, `
acl conflict dstdomain conflict.com
http_access allow conflict
http_access deny conflict
`)
	id := auth.Identity{Username: "u"}
	if !e.Decide(id, acl.RequestFields{SNI: "conflict.com", Path: "/"}) {
		t.Fatal("first http_access (allow) must win")
	}
}
