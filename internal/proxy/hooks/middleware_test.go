package hooks_test

import (
	"context"
	stdhttp "net/http"
	"net/url"
	"testing"

	"oktopus/internal/proxy/hooks"
)

func TestChainHTTPRequest_deny(t *testing.T) {
	chain := hooks.ChainHTTPRequest(
		func(context.Context, *stdhttp.Request) hooks.Decision { return hooks.AllowDecision() },
		func(context.Context, *stdhttp.Request) hooks.Decision { return hooks.DenyDecision() },
		func(context.Context, *stdhttp.Request) hooks.Decision {
			t.Fatal("should not run after deny")
			return hooks.AllowDecision()
		},
	)
	d := chain(context.Background(), &stdhttp.Request{})
	if d.Allow || d.Redirect != nil {
		t.Fatalf("expected deny, got %+v", d)
	}
}

func TestChainHTTPRequest_redirect(t *testing.T) {
	target, _ := url.Parse("https://example.com/blocked")
	chain := hooks.ChainHTTPRequest(func(context.Context, *stdhttp.Request) hooks.Decision {
		return hooks.RedirectDecision(target)
	})
	d := chain(context.Background(), &stdhttp.Request{})
	if d.Redirect == nil || d.Redirect.String() != target.String() {
		t.Fatalf("redirect: %+v", d)
	}
}
