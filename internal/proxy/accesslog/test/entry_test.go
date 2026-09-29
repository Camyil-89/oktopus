package accesslog_test

import (
	"context"
	"encoding/base64"
	stdhttp "net/http"
	"testing"

	"oktopus/internal/proxy/accesslog"
)

func TestAuthFailEntryCONNECT(t *testing.T) {
	t.Parallel()
	req, err := stdhttp.NewRequest(stdhttp.MethodConnect, "http://ignored/", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Host = "example.com:443"
	req.RemoteAddr = "10.0.0.1:55555"
	req.Header.Set("Proxy-Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("alice:wrong")))

	e := accesslog.AuthFailEntry(context.Background(), req, 0)
	if e.Action != accesslog.ActionDeny {
		t.Fatalf("action: got %d want DENY", e.Action)
	}
	if e.RuleRef != accesslog.RuleRefAuthFail {
		t.Fatalf("ruleRef: %q", e.RuleRef)
	}
	if e.DestinationAddress != "example.com:443" {
		t.Fatalf("dest: %q", e.DestinationAddress)
	}
	if e.User == nil || *e.User != "alice" {
		t.Fatalf("user: %v", e.User)
	}
}
