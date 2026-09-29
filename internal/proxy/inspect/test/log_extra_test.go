package inspect_test

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"oktopus/internal/proxy/inspect"
)

func TestEval_ctxLog_inExtra(t *testing.T) {
	ruleID := uuid.MustParse("018f0000-0000-7000-8000-000000000099")
	prog, err := inspect.BuildProgram([]inspect.RuleInput{{
		ID: ruleID, Name: "log", Action: 0,
		Script: `function inspect(ctx)
  ctx:log("reason", "test")
  ctx:log({ files = {"a.txt"} })
  return true
end`,
	}})
	if err != nil {
		t.Fatal(err)
	}
	res, err := inspect.NewRunner(prog).Eval(inspect.RequestContextFromHTTP(context.Background(), httptest.NewRequest("POST", "http://x/", nil)))
	if err != nil {
		t.Fatal(err)
	}
	block, ok := res.RuleLogs[ruleID.String()]
	if !ok {
		t.Fatalf("logs: %+v", res.RuleLogs)
	}
	if block["reason"] != "test" {
		t.Fatalf("reason: %v", block["reason"])
	}
}
