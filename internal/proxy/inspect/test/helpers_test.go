package inspect_test

import (
	"fmt"

	"github.com/google/uuid"

	"oktopus/internal/proxy/inspect"
)

const benchRuleCount = 500

const benchNoMatchScript = `function inspect(ctx)
  return false
end`

func benchHostMatchScript(host string) string {
	return fmt.Sprintf(`function inspect(ctx)
  return (ctx.host or "") == "%s"
end`, host)
}

func benchRuleInputs(n int, matchHost string) []inspect.RuleInput {
	out := make([]inspect.RuleInput, n)
	for i := 0; i < n; i++ {
		script := benchNoMatchScript
		if matchHost != "" && i == n-1 {
			script = benchHostMatchScript(matchHost)
		}
		out[i] = inspect.RuleInput{
			ID:     uuid.New(),
			Name:   fmt.Sprintf("rule-%d", i),
			Action: 0,
			Script: script,
		}
	}
	return out
}

func mustBenchRunner(n int, matchHost string) *inspect.Runner {
	prog, err := inspect.BuildProgram(benchRuleInputs(n, matchHost))
	if err != nil {
		panic(err)
	}
	return inspect.NewRunner(prog)
}
