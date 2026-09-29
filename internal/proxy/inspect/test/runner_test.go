package inspect_test

import (
	"testing"

	"github.com/google/uuid"

	"oktopus/internal/proxy/inspect"
)

func TestEval_denyStopsImmediately(t *testing.T) {
	t.Parallel()
	denyID := uuid.MustParse("018f0000-0000-7000-8000-000000000001")
	allowID := uuid.MustParse("018f0000-0000-7000-8000-000000000002")
	prog, err := inspect.BuildProgram([]inspect.RuleInput{
		{
			ID:     denyID,
			Name:   "deny-all",
			Action: 0,
			Script: `function inspect(ctx) return true end`,
		},
		{
			ID:     allowID,
			Name:   "allow-never-reached",
			Action: 1,
			Script: `function inspect(ctx) return true end`,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	res, err := inspect.NewRunner(prog).Eval(inspect.RequestContext{Method: "GET"})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Matched || !res.Deny || res.RuleID != denyID.String() {
		t.Fatalf("got %+v", res)
	}
}

func TestEval_allowContinuesUntilDeny(t *testing.T) {
	t.Parallel()
	allowID := uuid.MustParse("018f0000-0000-7000-8000-000000000001")
	denyID := uuid.MustParse("018f0000-0000-7000-8000-000000000002")
	prog, err := inspect.BuildProgram([]inspect.RuleInput{
		{
			ID:     allowID,
			Name:   "allow-tag",
			Action: 1,
			Script: `function inspect(ctx) return true end`,
		},
		{
			ID:     denyID,
			Name:   "deny-post",
			Action: 0,
			Script: `function inspect(ctx) return ctx.method == "POST" end`,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	runner := inspect.NewRunner(prog)

	res, err := runner.Eval(inspect.RequestContext{Method: "GET"})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Matched || res.Deny || res.RuleID != allowID.String() {
		t.Fatalf("GET: got %+v", res)
	}

	res, err = runner.Eval(inspect.RequestContext{Method: "POST"})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Matched || !res.Deny || res.RuleID != denyID.String() {
		t.Fatalf("POST: got %+v", res)
	}
}

func TestEval_allowOnlyNoMatch(t *testing.T) {
	t.Parallel()
	prog, err := inspect.BuildProgram([]inspect.RuleInput{
		{
			ID:     uuid.New(),
			Name:   "allow-never",
			Action: 1,
			Script: `function inspect(ctx) return false end`,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	res, err := inspect.NewRunner(prog).Eval(inspect.RequestContext{Method: "GET"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Matched {
		t.Fatalf("got %+v", res)
	}
}
