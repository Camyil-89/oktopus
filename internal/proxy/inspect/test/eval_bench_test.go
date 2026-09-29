// Бенчмарки inspect (Lua).
//
// Таблица с читаемым временем:
//
//	go run ./cmd bench ./internal/proxy/inspect/test/
//
// Или go test:
//
//	go test -bench=Benchmark -benchmem ./internal/proxy/inspect/test/
package inspect_test

import (
	"fmt"
	"testing"

	"github.com/google/uuid"

	"oktopus/internal/benchfmt"
	"oktopus/internal/proxy/inspect"
)

var (
	benchRunnerNoMatch   = mustBenchRunner(benchRuleCount, "")
	benchRunnerLastMatch = mustBenchRunner(benchRuleCount, "host-match.example.com")
	benchCtxNoMatch      = inspect.RequestContext{
		Method: "GET",
		Host:   "ok.example.com",
		Path:   "/",
	}
	benchCtxLastMatch = inspect.RequestContext{
		Method: "GET",
		Host:   "host-match.example.com",
		Path:   "/",
	}
)

func BenchmarkBuildProgram_500Rules(b *testing.B) {
	inputs := benchRuleInputs(benchRuleCount, "")
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		prog, err := inspect.BuildProgram(inputs)
		if err != nil {
			b.Fatal(err)
		}
		if prog == nil {
			b.Fatal("nil program")
		}
	}
	benchfmt.ReportHumanDuration(b)
}

func BenchmarkNewRunner_500Rules(b *testing.B) {
	prog, err := inspect.BuildProgram(benchRuleInputs(benchRuleCount, ""))
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		r := inspect.NewRunner(prog)
		if r.RuleCount() != benchRuleCount {
			b.Fatalf("rule count %d", r.RuleCount())
		}
	}
	benchfmt.ReportHumanDuration(b)
}

func BenchmarkEval_500_NoMatch(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		res, err := benchRunnerNoMatch.Eval(benchCtxNoMatch)
		if err != nil {
			b.Fatal(err)
		}
		if res.Matched {
			b.Fatal("expected no match")
		}
	}
	benchfmt.ReportHumanDuration(b)
}

func BenchmarkEval_500_FirstRuleMatch(b *testing.B) {
	prog, err := inspect.BuildProgram([]inspect.RuleInput{{
		ID:     uuid.New(),
		Name:   "first",
		Action: 0,
		Script: benchHostMatchScript("host0.example.com"),
	}})
	if err != nil {
		b.Fatal(err)
	}
	runner := inspect.NewRunner(prog)
	ctx := inspect.RequestContext{Method: "GET", Host: "host0.example.com", Path: "/"}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		res, err := runner.Eval(ctx)
		if err != nil {
			b.Fatal(err)
		}
		if !res.Matched || !res.Deny {
			b.Fatal("expected deny match")
		}
	}
	benchfmt.ReportHumanDuration(b)
}

func BenchmarkEval_500_MiddleRuleMatch(b *testing.B) {
	host := fmt.Sprintf("host%d.example.com", benchRuleCount/2)
	inputs := benchRuleInputs(benchRuleCount, "")
	mid := benchRuleCount / 2
	inputs[mid].Script = benchHostMatchScript(host)
	prog, err := inspect.BuildProgram(inputs)
	if err != nil {
		b.Fatal(err)
	}
	runner := inspect.NewRunner(prog)
	ctx := inspect.RequestContext{Method: "GET", Host: host, Path: "/"}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		res, err := runner.Eval(ctx)
		if err != nil {
			b.Fatal(err)
		}
		if !res.Matched || !res.Deny {
			b.Fatal("expected deny match")
		}
	}
	benchfmt.ReportHumanDuration(b)
}

func BenchmarkEval_500_LastRuleMatch(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		res, err := benchRunnerLastMatch.Eval(benchCtxLastMatch)
		if err != nil {
			b.Fatal(err)
		}
		if !res.Matched || !res.Deny {
			b.Fatal("expected deny match")
		}
	}
	benchfmt.ReportHumanDuration(b)
}
