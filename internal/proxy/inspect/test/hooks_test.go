package inspect_test

import (
	"context"
	stdhttp "net/http"
	"net/http/httptest"
	"testing"

	"oktopus/internal/proxy/hooks"
	"oktopus/internal/proxy/inspect"
	"oktopus/internal/proxy/observe"
)

func TestHTTPMiddleware_skipsWithoutACLAllowPassed(t *testing.T) {
	t.Parallel()
	prog, err := inspect.BuildProgram([]inspect.RuleInput{
		{Name: "x", Action: 0, Script: `function inspect(ctx) return true end`},
	})
	if err != nil {
		t.Fatal(err)
	}
	mw := inspect.HTTPMiddleware(inspect.NewRunner(prog), nil, true)

	req := httptest.NewRequest("POST", "http://example.com/", nil)
	d := mw(context.Background(), req)
	if !d.Allow {
		t.Fatal("expected allow when ACL flag missing (inspect skipped)")
	}
}

func TestHTTPMiddleware_runsAfterACLAllowPassed(t *testing.T) {
	t.Parallel()
	prog, err := inspect.BuildProgram([]inspect.RuleInput{
		{
			Name:   "always",
			Action: 0,
			Script: `function inspect(ctx) return true end`,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	mw := inspect.HTTPMiddleware(inspect.NewRunner(prog), nil, true)
	req := httptest.NewRequest("GET", "http://example.com/", nil)
	req = req.WithContext(observe.WithACLAllowPassed(req.Context()))
	req = req.WithContext(observe.WithACLDecision(req.Context(), observe.ACLDecision{RuleRef: "acl-rule", Spend: 0}))
	d := mw(context.Background(), req)
	if d.Allow {
		t.Fatal("expected deny from inspect rule")
	}
}

func TestChainACLBeforeInspect(t *testing.T) {
	t.Parallel()
	var order []string
	aclMW := func(context.Context, *stdhttp.Request) hooks.Decision {
		order = append(order, "acl")
		return hooks.DenyDecision()
	}
	inspectMW := func(context.Context, *stdhttp.Request) hooks.Decision {
		order = append(order, "inspect")
		return hooks.AllowDecision()
	}
	chain := func(ctx context.Context, req *stdhttp.Request) hooks.Decision {
		if d := aclMW(ctx, req); d.Handled() {
			return d
		}
		return inspectMW(ctx, req)
	}
	req := httptest.NewRequest("GET", "http://example.com/", nil)
	chain(context.Background(), req)
	if len(order) != 1 || order[0] != "acl" {
		t.Fatalf("order: %v", order)
	}
}
