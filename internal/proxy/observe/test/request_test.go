package observe_test

import (
	"context"
	stdhttp "net/http"
	"testing"

	"oktopus/internal/proxy/auth"
	"oktopus/internal/proxy/observe"
)

func TestRequestTools_lazy(t *testing.T) {
	req, err := stdhttp.NewRequest(stdhttp.MethodGet, "https://example.com/p?q=1&x=2", nil)
	if err != nil {
		t.Fatal(err)
	}
	tools := observe.RequestToolsFor(context.Background(), req)

	if tools.Path() != "/p" {
		t.Fatalf("path: %q", tools.Path())
	}
	q := tools.Query()
	if q.Get("q") != "1" || q.Get("x") != "2" {
		t.Fatalf("query: %v", q)
	}
	if tools.SNI() != "" {
		t.Fatalf("sni without context: %q", tools.SNI())
	}
}

func TestRequestTools_SNI(t *testing.T) {
	ctx := observe.WithSNI(context.Background(), "app.example.com")
	req := &stdhttp.Request{}
	tools := observe.RequestToolsFor(ctx, req)
	if tools.SNI() != "app.example.com" {
		t.Fatalf("sni: %q", tools.SNI())
	}
}

func TestRequestTools_Identity(t *testing.T) {
	req := &stdhttp.Request{}
	tools := observe.RequestToolsFor(context.Background(), req)
	if tools.Identity() != nil {
		t.Fatal("expected nil without auth context")
	}

	ctx := auth.WithIdentity(context.Background(), auth.Identity{
		Username: "user",
		Groups:   []string{"allow_all"},
	})
	tools = observe.RequestToolsFor(ctx, req)
	id := tools.Identity()
	if id == nil || id.Username != "user" || len(id.Groups) != 1 {
		t.Fatalf("identity: %+v", id)
	}
}
