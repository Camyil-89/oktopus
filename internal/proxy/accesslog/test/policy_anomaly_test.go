package accesslog_test

import (
	"context"
	stdhttp "net/http"
	"testing"

	"oktopus/internal/proxy/accesslog"
	"oktopus/internal/proxy/observe"
)

func TestHTTPEntryPolicyAnomalyMITM(t *testing.T) {
	ctx := observe.WithCONNECTDestHostPort(context.Background(), "localhost:9443")
	ctx = observe.WithMITMClientHelloSNI(ctx, "internal.blocked")
	req, err := stdhttp.NewRequest(stdhttp.MethodGet, "https://localhost/secret", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Host = "localhost"
	req = req.WithContext(ctx)

	e := accesslog.HTTPEntry(ctx, req, false, 0, "http_access deny evil")
	if e.PolicyNameMismatch == nil {
		t.Fatal("expected policy name mismatch note")
	}
	if e.PolicyNameMismatch.TLSClientSNI != "internal.blocked" {
		t.Fatalf("sni: %q", e.PolicyNameMismatch.TLSClientSNI)
	}
}
