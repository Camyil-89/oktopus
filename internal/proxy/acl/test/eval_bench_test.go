// Бенчмарки ACL (10k правил, Squid-конфиг).
//
// Таблица с читаемым временем и ~RPS:
//
//	go run ./cmd bench ./internal/proxy/acl/test/
//
// Или go test с доп. колонкой ms/op|µs/op:
//
//	go test -bench=Benchmark -benchmem ./internal/proxy/acl/test/
package acl_test

import (
	"fmt"
	"testing"

	"oktopus/internal/benchfmt"
	"oktopus/internal/proxy/acl"
	"oktopus/internal/proxy/auth"
)

var (
	benchEngine10k     = mustBenchEngine10k()
	benchDevLike10k    = mustBenchEngineDevLike()
	benchIdentity      = auth.Identity{Username: "user", Groups: []string{"allow_all"}}
	benchIdentityPlain = auth.Identity{Username: "user"}
)

func mustBenchEngine10k() *acl.Engine {
	e, err := benchSquidOneListDeny(benchRuleCount)
	if err != nil {
		panic(err)
	}
	return e
}

func mustBenchEngineDevLike() *acl.Engine {
	e, err := benchSquidDevLike(benchRuleCount)
	if err != nil {
		panic(err)
	}
	return e
}

func BenchmarkLoadFile_10kOneRuleRegexs(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		e, err := benchSquidOneListDeny(benchRuleCount)
		if err != nil {
			b.Fatal(err)
		}
		if e.HTTPAccessCount() != 1 || e.PatternCount() < benchRuleCount {
			b.Fatalf("unexpected engine shape access=%d patterns=%d", e.HTTPAccessCount(), e.PatternCount())
		}
	}
	benchfmt.ReportHumanDuration(b)
}

func BenchmarkDecide_10k_OneRuleRegexs_NoMatch(b *testing.B) {
	e, err := benchSquidOneListDeny(benchRuleCount)
	if err != nil {
		b.Fatal(err)
	}
	f := acl.RequestFields{SNI: "ok.example.com", Path: "/"}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if e.Decide(benchIdentityPlain, f) {
			b.Fatal("expected deny (default deny on no match)")
		}
	}
	benchfmt.ReportHumanDuration(b)
}

func BenchmarkLoadFile_10kRules(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		e, err := benchSquidSeparateDenyRules(benchRuleCount)
		if err != nil {
			b.Fatal(err)
		}
		if e.HTTPAccessCount() != benchRuleCount {
			b.Fatalf("unexpected engine shape access=%d", e.HTTPAccessCount())
		}
	}
	benchfmt.ReportHumanDuration(b)
}

func BenchmarkDecide_10k_NoMatch(b *testing.B) {
	f := acl.RequestFields{SNI: "ok.example.com", Path: "/"}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if benchEngine10k.Decide(benchIdentityPlain, f) {
			b.Fatal("expected deny (default deny on no match)")
		}
	}
	benchfmt.ReportHumanDuration(b)
}

func BenchmarkDecide_10k_FirstRuleMatch(b *testing.B) {
	f := acl.RequestFields{SNI: "host0.example.com", Path: "/"}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if benchEngine10k.Decide(benchIdentityPlain, f) {
			b.Fatal("expected deny")
		}
	}
	benchfmt.ReportHumanDuration(b)
}

func BenchmarkDecide_10k_MiddleRuleMatch(b *testing.B) {
	f := acl.RequestFields{SNI: fmt.Sprintf("host%d.example.com", benchRuleCount/2), Path: "/"}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if benchEngine10k.Decide(benchIdentityPlain, f) {
			b.Fatal("expected deny")
		}
	}
	benchfmt.ReportHumanDuration(b)
}

func BenchmarkDecide_10k_LastRuleMatch(b *testing.B) {
	f := acl.RequestFields{SNI: fmt.Sprintf("host%d.example.com", benchRuleCount-1), Path: "/"}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if benchEngine10k.Decide(benchIdentityPlain, f) {
			b.Fatal("expected deny")
		}
	}
	benchfmt.ReportHumanDuration(b)
}

func BenchmarkDecide_10k_DevLike_NoMatch(b *testing.B) {
	f := acl.RequestFields{SNI: "ok.com", Path: "/"}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if !benchDevLike10k.Decide(benchIdentity, f) {
			b.Fatal("expected allow")
		}
	}
	benchfmt.ReportHumanDuration(b)
}

func BenchmarkDecide_50k_OneRuleRegexs_NoMatch(b *testing.B) {
	if testing.Short() {
		b.Skip("large ACL bench")
	}
	const n = 50_000
	e, err := benchSquidOneListDeny(n)
	if err != nil {
		b.Fatal(err)
	}
	if e.SlowRuleSlotCount() != 0 {
		b.Fatalf("expected indexed literals only, slow slots=%d", e.SlowRuleSlotCount())
	}
	f := acl.RequestFields{SNI: "ok.example.com", Path: "/"}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if e.Decide(benchIdentityPlain, f) {
			b.Fatal("expected deny")
		}
	}
	benchfmt.ReportHumanDuration(b)
}

func BenchmarkDecide_10k_DevLike_GlobalDenyWins(b *testing.B) {
	f := acl.RequestFields{SNI: "blocked0.evil", Path: "/"}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if benchDevLike10k.Decide(benchIdentity, f) {
			b.Fatal("expected deny")
		}
	}
	benchfmt.ReportHumanDuration(b)
}
