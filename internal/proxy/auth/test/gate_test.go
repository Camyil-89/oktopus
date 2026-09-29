package auth_test

import (
	"context"
	"encoding/base64"
	stdhttp "net/http"
	"net/http/httptest"
	"testing"
	"time"

	"oktopus/internal/proxy/auth"
)

func TestGateRequire(t *testing.T) {
	st, err := auth.ParseStaticUsers("u:p\n")
	if err != nil {
		t.Fatal(err)
	}
	g := &auth.Gate{
		Realm: "test",
		Auth:  st,
	}

	t.Run("no credentials", func(t *testing.T) {
		var logged int
		g.LogAuthFail = func(_ context.Context, _ *stdhttp.Request, _ time.Duration) {
			logged++
		}
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(stdhttp.MethodGet, "http://example.com/", nil)
		_, ok := g.Require(rec, req)
		if ok || rec.Code != stdhttp.StatusProxyAuthRequired {
			t.Fatalf("code=%d ok=%v", rec.Code, ok)
		}
		if logged != 1 {
			t.Fatalf("LogAuthFail calls: %d", logged)
		}
		if rec.Header().Get("Proxy-Authenticate") == "" {
			t.Fatal("missing Proxy-Authenticate")
		}
	})

	t.Run("valid basic", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(stdhttp.MethodConnect, "https://example.com:443", nil)
		req.Header.Set("Proxy-Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("u:p")))
		ctx, ok := g.Require(rec, req)
		if !ok {
			t.Fatalf("code=%d", rec.Code)
		}
		id, found := auth.IdentityFromContext(ctx)
		if !found || id.Username != "u" {
			t.Fatalf("identity=%+v found=%v", id, found)
		}
	})
}

func TestGateDisabled(t *testing.T) {
	var g *auth.Gate
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(stdhttp.MethodGet, "http://example.com/", nil)
	ctx, ok := g.Require(rec, req)
	if !ok {
		t.Fatal("expected allow when gate nil")
	}
	if _, found := auth.IdentityFromContext(ctx); found {
		t.Fatal("unexpected identity")
	}
}
