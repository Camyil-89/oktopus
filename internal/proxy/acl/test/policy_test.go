package acl_test

import (
	"context"
	stdhttp "net/http"
	"testing"

	"oktopus/internal/proxy/acl"
	"oktopus/internal/proxy/auth"
	"oktopus/internal/proxy/observe"
)

func TestEvaluatePolicyConnectMatchesDecideFields(t *testing.T) {
	e := mustEngineFromSquid(t, `
acl ok dstdomain example.com
http_access allow ok
`)
	id := auth.Identity{Username: "u"}
	ctx := observe.WithRemoteAddr(context.Background(), "10.0.0.1:1234")
	pr := acl.PolicyRequestFromConnect(ctx, "example.com:443")
	res := e.EvaluatePolicy(ctx, id, pr, nil)
	if !res.Allow {
		t.Fatal("expected allow")
	}
	if res.Fields.SNI != "example.com" || res.Fields.DstPort != 443 {
		t.Fatalf("fields: %+v", res.Fields)
	}
}

func TestPolicyRequestFromHTTPUsesURLHost(t *testing.T) {
	ctx := observe.WithSNI(context.Background(), "tunnel.sni")
	req, _ := stdhttp.NewRequest(stdhttp.MethodGet, "https://real.origin/path", nil)
	pr := acl.PolicyRequestFromHTTP(ctx, req)
	if pr.Host != "real.origin" || pr.Path != "/path" {
		t.Fatalf("pr: %+v", pr)
	}
}
