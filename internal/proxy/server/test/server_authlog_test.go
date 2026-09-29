package server_test

import (
	"net/http/httptest"
	"sync"
	"testing"

	"oktopus/internal/proxy/accesslog"
	"oktopus/internal/proxy/acl"
	"oktopus/internal/proxy/config"
	"oktopus/internal/proxy/hooks"
	"oktopus/internal/proxy/server"
)

type recordSpy struct {
	mu      sync.Mutex
	entries []accesslog.Entry
}

func (s *recordSpy) Record(e accesslog.Entry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries = append(s.entries, e)
}

func TestServeHTTPAuthFailRecordsDenyWithSystemRule(t *testing.T) {
	t.Parallel()

	rec := &recordSpy{}
	srv, err := server.New(config.Config{
		Listen:  "127.0.0.1:0",
		Connect: config.ConnectTunnel,
		Auth: config.AuthConfig{
			Enabled:     true,
			StaticUsers: "u:p",
		},
	}, acl.EmptyEngine(), nil, &hooks.Hooks{}, rec, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest("CONNECT", "http://example.com:443", nil)
	req.Host = "example.com:443"
	req.RemoteAddr = "10.0.0.5:1234"
	srv.ServeHTTP(w, req)

	rec.mu.Lock()
	defer rec.mu.Unlock()
	if len(rec.entries) != 1 {
		t.Fatalf("entries: %d", len(rec.entries))
	}
	if rec.entries[0].Action != accesslog.ActionDeny {
		t.Fatalf("action=%d want DENY", rec.entries[0].Action)
	}
	if rec.entries[0].RuleRef != accesslog.RuleRefAuthFail {
		t.Fatalf("ruleRef=%q", rec.entries[0].RuleRef)
	}
}

func TestServeHTTPACLDenyRecordsDeny(t *testing.T) {
	t.Parallel()

	rec := &recordSpy{}
	srv, err := server.New(config.Config{
		Listen:  "127.0.0.1:0",
		Connect: config.ConnectTunnel,
		Auth: config.AuthConfig{
			Enabled:     true,
			StaticUsers: "u:p",
		},
	}, acl.EmptyEngine(), nil, &hooks.Hooks{}, rec, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest("CONNECT", "http://example.com:443", nil)
	req.Host = "example.com:443"
	req.Header.Set("Proxy-Authorization", "Basic dTpw") // u:p
	req.RemoteAddr = "10.0.0.5:1234"
	srv.ServeHTTP(w, req)

	rec.mu.Lock()
	defer rec.mu.Unlock()
	if len(rec.entries) != 1 {
		t.Fatalf("entries: %d", len(rec.entries))
	}
	if rec.entries[0].Action != accesslog.ActionDeny {
		t.Fatalf("action=%d want DENY", rec.entries[0].Action)
	}
}

var _ accesslog.Recorder = (*recordSpy)(nil)
