package acl_test

import (
	"context"
	stdhttp "net/http"
	"testing"

	"oktopus/internal/proxy/observe"
)

func TestHTTPPolicyHostDeniesLoopbackDespiteTunnelSNI(t *testing.T) {
	eng := mustEngineFromSquid(t, `
acl public dstdomain public.test
acl loopback dst 127.0.0.0/8
acl all all
http_access deny loopback
http_access allow public
http_access deny all
`)
	mw := eng.Hooks(nil).OnHTTPRequest
	ctx := observe.WithSNI(context.Background(), "public.test")

	req, err := stdhttp.NewRequest(stdhttp.MethodGet, "https://127.0.0.1:9555/secret", nil)
	if err != nil {
		t.Fatal(err)
	}
	dec := mw(ctx, req)
	if dec.Allow {
		t.Fatal("expected deny when URL host is loopback but context SNI is public.test")
	}

	req2, err := stdhttp.NewRequest(stdhttp.MethodGet, "https://public.test/ok", nil)
	if err != nil {
		t.Fatal(err)
	}
	dec2 := mw(ctx, req2)
	if !dec2.Allow {
		t.Fatal("expected allow for public.test URL with same tunnel SNI")
	}
}
