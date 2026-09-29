package squid_test

import (
	"net"
	"testing"

	"oktopus/internal/proxy/acl"
	"oktopus/internal/proxy/acl/squid"
	"oktopus/internal/proxy/auth"
)

func TestSquidPatternMatch(t *testing.T) {
	cp, err := acl.CompilePatternLine(acl.RuleSNI, ".evil.test")
	if err != nil {
		t.Fatal(err)
	}
	m := acl.NewSquidMatchPatterns(acl.RuleSNI, []acl.CompiledPatternPublic{cp})
	if !m.Matches(auth.Identity{}, acl.RequestFields{SNI: "app.evil.test"}, acl.SquidPhaseHTTP) {
		t.Fatal("expected sni match")
	}
}

func TestCompileSquidBasic(t *testing.T) {
	cfg := `
acl localnet src 10.0.0.0/8
acl blocked dstdomain .evil.test
acl badport port 666
acl all all

http_access deny blocked
http_access deny badport localnet
http_access allow all
`
	engine, err := squid.Compile(cfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	id := auth.Identity{Groups: []string{"x"}}
	if !engine.Decide(id, acl.RequestFields{SNI: "safe.test", DstPort: 443, SrcIP: mustIP("10.1.2.3")}) {
		t.Fatal("expected allow via allow all")
	}
	if engine.Decide(id, acl.RequestFields{SNI: "app.evil.test", DstPort: 443}) {
		t.Fatal("expected deny blocked domain")
	}
	if !engine.Decide(id, acl.RequestFields{SNI: "ok.test", DstPort: 443, SrcIP: mustIP("10.1.2.3")}) {
		t.Fatal("expected allow from allow all")
	}
}

func TestMergeDBList(t *testing.T) {
	cfg := `
acl all all
http_access deny nets
http_access allow all
`
	engine, err := squid.Compile(cfg, []squid.ListInput{{
		Name: "nets", ListType: "src", Body: "192.168.0.0/16",
	}})
	if err != nil {
		t.Fatal(err)
	}
	id := auth.Identity{}
	if engine.Decide(id, acl.RequestFields{SrcIP: mustIP("192.168.1.1"), SNI: "x", DstPort: 80}) {
		t.Fatal("expected deny from list")
	}
}

func TestAnalyzeParallelMergedLists(t *testing.T) {
	cfg := `acl all all
http_access deny list_a
http_access deny list_b
http_access allow all
`
	lists := []squid.ListInput{
		{Name: "list_a", ListType: "dstdomain", Body: "a.example.com\n"},
		{Name: "list_b", ListType: "dstdomain", Body: "b.example.com\n"},
	}
	res := squid.Analyze(cfg, lists)
	if !res.OK {
		t.Fatalf("expected ok, got %v", res.Diagnostics)
	}
}

func mustIP(s string) net.IP {
	return net.ParseIP(s)
}
