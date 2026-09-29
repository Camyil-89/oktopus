package acl_test

import (
	"net"
	"testing"

	"oktopus/internal/proxy/acl"
	"oktopus/internal/proxy/auth"
)

func TestDecideSRCExactCIDRRange(t *testing.T) {
	e := mustEngineFromSquid(t, `
acl deny_src src 10.0.0.0/24 10.0.2.5
acl allow_src src 10.0.1.10-10.0.1.20
acl ok dstdomain ok.com
http_access deny deny_src
http_access allow allow_src
http_access allow ok
`)
	id := auth.Identity{Username: "u"}

	if e.Decide(id, acl.RequestFields{SNI: "ok.com", Path: "/", SrcIP: net.ParseIP("10.0.0.50")}) {
		t.Fatal("cidr deny expected")
	}
	if !e.Decide(id, acl.RequestFields{SNI: "ok.com", Path: "/", SrcIP: net.ParseIP("10.0.1.15")}) {
		t.Fatal("range allow expected")
	}
	if e.Decide(id, acl.RequestFields{SNI: "other.com", Path: "/", SrcIP: net.ParseIP("10.0.3.1")}) {
		t.Fatal("default deny for unknown src")
	}
}

func TestDecideDSTFromSNIIPLiteral(t *testing.T) {
	e := mustEngineFromSquid(t, `
acl bad dst 203.0.113.0/24
acl ok dstdomain ok.com
http_access deny bad
http_access allow ok
`)
	id := auth.Identity{Username: "u"}

	if e.Decide(id, acl.RequestFields{SNI: "203.0.113.9", Path: "/"}) {
		t.Fatal("dst cidr via sni expected deny")
	}
	if !e.Decide(id, acl.RequestFields{SNI: "ok.com", Path: "/"}) {
		t.Fatal("unrelated sni allow")
	}
}

func TestDecideSRCBeforeSNIOrder(t *testing.T) {
	e := mustEngineFromSquid(t, `
acl bad_src src 192.168.1.1
acl any url_regex .
http_access deny bad_src
http_access allow any
`)
	id := auth.Identity{Username: "u"}
	if e.Decide(id, acl.RequestFields{SNI: "any.com", Path: "/", SrcIP: net.ParseIP("192.168.1.1")}) {
		t.Fatal("first rule deny")
	}
}

func TestSRCExactAndCIDRBothMatchFirstRuleWins(t *testing.T) {
	e := mustEngineFromSquid(t, `
acl deny_one src 10.0.0.1
acl allow_net src 10.0.0.0/24
acl ok dstdomain ok.com
http_access deny deny_one
http_access allow allow_net
http_access allow ok
`)
	id := auth.Identity{Username: "u"}
	if e.Decide(id, acl.RequestFields{SNI: "ok.com", Path: "/", SrcIP: net.ParseIP("10.0.0.1")}) {
		t.Fatal("expected deny from first matching SRC rule")
	}
}

func TestValidateRulePatternSRC(t *testing.T) {
	if err := acl.ValidateRulePattern(acl.RuleSRC, "not-an-ip"); err == nil {
		t.Fatal("expected error")
	}
	if err := acl.ValidateRulePattern(acl.RuleSRC, "10.0.0.0/24\n10.0.0.1-10.0.0.9"); err != nil {
		t.Fatal(err)
	}
}
