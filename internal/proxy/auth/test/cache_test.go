package auth_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"log"
	stdhttp "net/http"
	"net/http/httptest"
	"testing"
	"time"

	"oktopus/internal/proxy/auth"
)

type stubAuth struct {
	calls int
	id    auth.Identity
}

func (s *stubAuth) Authenticate(_ context.Context, _, _ string) (auth.Identity, error) {
	s.calls++
	return s.id, nil
}

func TestGateLDAPCacheAndLog(t *testing.T) {
	inner := &stubAuth{id: auth.Identity{Username: "user", Groups: []string{"allow_all"}}}
	var buf bytes.Buffer
	logger := log.New(&buf, "", 0)
	g := &auth.Gate{
		Realm:   "test",
		Auth:    inner,
		Cache:   auth.NewAuthCache(time.Minute),
		Backend: "ldap",
		Log:     logger,
	}

	req := httptest.NewRequest(stdhttp.MethodGet, "http://example.com/", nil)
	req.Header.Set("Proxy-Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("user:pass")))
	rec := httptest.NewRecorder()
	_, ok := g.Require(rec, req)
	if !ok || inner.calls != 1 {
		t.Fatalf("first auth: ok=%v calls=%d", ok, inner.calls)
	}
	if !bytes.Contains(buf.Bytes(), []byte("saved to cache")) {
		t.Fatalf("log: %s", buf.String())
	}

	buf.Reset()
	rec2 := httptest.NewRecorder()
	_, ok = g.Require(rec2, req)
	if !ok || inner.calls != 1 {
		t.Fatalf("cache hit: ok=%v calls=%d", ok, inner.calls)
	}
	if buf.Len() != 0 {
		t.Fatalf("unexpected log on cache hit: %s", buf.String())
	}
}
