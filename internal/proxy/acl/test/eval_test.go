package acl_test

import (
	"testing"

	"oktopus/internal/proxy/acl"
	"oktopus/internal/proxy/auth"
)

func TestDecideGroupAllowDeny(t *testing.T) {
	e := mustEngineFromSquid(t, `
acl allow_all ldap_group allow_all
acl deny_all ldap_group deny_all
acl all all
http_access allow allow_all
http_access deny deny_all
http_access deny all
`)

	if !e.Decide(auth.Identity{Username: "user", Groups: []string{"allow_all"}}, acl.RequestFields{SNI: "x.com", Path: "/"}) {
		t.Fatal("allow_all should pass")
	}
	if e.Decide(auth.Identity{Username: "user2", Groups: []string{"deny_all"}}, acl.RequestFields{SNI: "x.com", Path: "/"}) {
		t.Fatal("deny_all should block")
	}
}

func TestDecideGlobalDenyBeforeGroupAllow(t *testing.T) {
	e := mustEngineFromSquid(t, `
acl allow_all ldap_group allow_all
acl bad dstdomain blocked.evil
acl all all
http_access deny bad
http_access allow allow_all
http_access deny all
`)
	id := auth.Identity{Username: "user", Groups: []string{"allow_all"}}
	if e.Decide(id, acl.RequestFields{SNI: "blocked.evil", Path: "/"}) {
		t.Fatal("global DENY must win over group ALLOW")
	}
	if !e.Decide(id, acl.RequestFields{SNI: "ok.com", Path: "/"}) {
		t.Fatal("allow_all should pass other hosts")
	}
}

func TestDecideSNIDeny(t *testing.T) {
	e := mustEngineFromSquid(t, `
acl bad dstdomain blocked.evil
http_access deny bad
`)
	id := auth.Identity{Username: "any"}
	if e.Decide(id, acl.RequestFields{SNI: "ok.com", Path: "/"}) {
		t.Fatal("ok.com should deny without explicit allow")
	}
	if e.Decide(id, acl.RequestFields{SNI: "blocked.evil", Path: "/"}) {
		t.Fatal("blocked.evil should deny")
	}
}
