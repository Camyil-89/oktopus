package acl_test

import (
	"testing"

	"oktopus/internal/proxy/acl"
	"oktopus/internal/proxy/auth"
)

func TestDecidePORTDenyExact(t *testing.T) {
	e := mustEngineFromSquid(t, `
acl bad_port port 22 3389
acl all all
http_access deny bad_port
http_access allow all
`)
	id := auth.Identity{Username: "u"}
	if e.Decide(id, acl.RequestFields{SNI: "ssh.example.com", Path: "/", DstPort: 22}) {
		t.Fatal("expected deny port 22")
	}
	if !e.Decide(id, acl.RequestFields{SNI: "ssh.example.com", Path: "/", DstPort: 2222}) {
		t.Fatal("expected allow other port")
	}
}

func TestDecidePORTRange(t *testing.T) {
	e := mustEngineFromSquid(t, `
acl bad_range port 8000-8010
acl all all
http_access deny bad_range
http_access allow all
`)
	id := auth.Identity{Username: "u"}
	if e.Decide(id, acl.RequestFields{SNI: "x", Path: "/", DstPort: 8005}) {
		t.Fatal("expected deny in range")
	}
	if !e.Decide(id, acl.RequestFields{SNI: "x", Path: "/", DstPort: 7999}) {
		t.Fatal("expected allow below range")
	}
}

func TestDecidePORTBeforeSNIOrder(t *testing.T) {
	e := mustEngineFromSquid(t, `
acl bad_port port 443
acl any url_regex .
http_access deny bad_port
http_access allow any
`)
	id := auth.Identity{Username: "u"}
	if e.Decide(id, acl.RequestFields{SNI: "ok.com", Path: "/", DstPort: 443}) {
		t.Fatal("port rule should win")
	}
}

func TestValidateRulePatternPORT(t *testing.T) {
	if err := acl.ValidateRulePattern(acl.RulePORT, "not-a-port"); err == nil {
		t.Fatal("expected error")
	}
	if err := acl.ValidateRulePattern(acl.RulePORT, "443\n9000-9010"); err != nil {
		t.Fatal(err)
	}
}

func TestParseDstPort(t *testing.T) {
	if p := acl.ParseDstPort("example.com:8443"); p != 8443 {
		t.Fatalf("got %d", p)
	}
	if p := acl.ParseDstPort("example.com"); p != 0 {
		t.Fatalf("got %d", p)
	}
}
