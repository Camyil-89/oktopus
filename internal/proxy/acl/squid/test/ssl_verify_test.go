package squid_test

import (
	"testing"

	"oktopus/internal/proxy/acl"
	"oktopus/internal/proxy/acl/squid"
	"oktopus/internal/proxy/auth"
)

func TestSSLVerifySkipByDomain(t *testing.T) {
	cfg := `
acl all all
acl bank dstdomain .sovcombank.ru
http_access allow all
ssl_verify skip bank
ssl_verify require all
`
	engine, err := squid.Compile(cfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	id := auth.Identity{}
	verify, ref := engine.DecideUpstreamTLSVerify(id, acl.RequestFields{SNI: "sovcombank.ru", DstPort: 443})
	if verify {
		t.Fatal("expected skip verify for bank domain")
	}
	if ref != "ssl_verify skip bank" {
		t.Fatalf("ref %q", ref)
	}
	verify, _ = engine.DecideUpstreamTLSVerify(id, acl.RequestFields{SNI: "example.com", DstPort: 443})
	if !verify {
		t.Fatal("expected require verify for other domains")
	}
}

func TestSSLVerifyByLDAPGroup(t *testing.T) {
	cfg := `
acl devs ldap_group Developers
acl all all
http_access allow all
ssl_verify skip devs
ssl_verify require all
`
	engine, err := squid.Compile(cfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	id := auth.Identity{Groups: []string{"Developers"}}
	verify, _ := engine.DecideUpstreamTLSVerify(id, acl.RequestFields{SNI: "any.test", DstPort: 443})
	if verify {
		t.Fatal("expected skip for devs group")
	}
	id2 := auth.Identity{Groups: []string{"Users"}}
	verify, _ = engine.DecideUpstreamTLSVerify(id2, acl.RequestFields{SNI: "any.test", DstPort: 443})
	if !verify {
		t.Fatal("expected require for other users")
	}
}
