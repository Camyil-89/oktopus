package inspect_test

import (
	"testing"

	"github.com/google/uuid"

	"oktopus/internal/proxy/inspect"
)

func TestBuildProgram_loadsInspectFn(t *testing.T) {
	prog, err := inspect.BuildProgram([]inspect.RuleInput{
		{
			ID:     uuid.MustParse("018f0000-0000-7000-8000-000000000001"),
			Name:   "multipart",
			Action: 0,
			Script: `function inspect(ctx)
  return false
end`,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	runner := inspect.NewRunner(prog)
	if runner.Empty() {
		t.Fatal("expected non-empty runner")
	}
	res, err := runner.Eval(inspect.RequestContext{Method: "GET"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Matched {
		t.Fatalf("unexpected match %+v", res)
	}
}

func TestBuildProgram_andEval_multipart(t *testing.T) {
	prog, err := inspect.BuildProgram([]inspect.RuleInput{
		{
			ID:     uuid.MustParse("018f0000-0000-7000-8000-000000000001"),
			Name:   "multipart",
			Action: 0,
			Script: `function inspect(ctx)
  if ctx.method ~= "POST" then return false end
  return (ctx.content_type or ""):find("multipart/", 1, true) ~= nil
end`,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	runner := inspect.NewRunner(prog)
	res, err := runner.Eval(inspect.RequestContext{
		Method:      "POST",
		ContentType: "multipart/form-data; boundary=x",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Matched || !res.Deny {
		t.Fatalf("expected deny match, got %+v", res)
	}
}
