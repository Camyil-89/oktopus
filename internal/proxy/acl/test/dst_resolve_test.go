package acl_test

import (
	"context"
	"net"
	"testing"

	"oktopus/internal/proxy/acl"
	"oktopus/internal/proxy/auth"
)

func TestDecideDSTDeniesResolvedLoopback(t *testing.T) {
	t.Cleanup(func() { acl.SetDstLookupForTest(nil) })
	acl.SetDstLookupForTest(func(_ context.Context, host string) ([]net.IP, error) {
		if host == "internal.lab" {
			return []net.IP{net.ParseIP("127.0.0.1")}, nil
		}
		return nil, nil
	})

	e := mustEngineFromSquid(t, `
acl public dstdomain internal.lab
acl loopback dst 127.0.0.0/8
acl any url_regex .
http_access deny loopback
http_access allow public
http_access deny any
`)
	id := auth.Identity{Username: "u"}
	f := acl.EnrichRequestFieldsDst(context.Background(), acl.RequestFields{SNI: "internal.lab", Path: "/"})
	if e.Decide(id, f) {
		t.Fatal("expected deny: resolved 127.0.0.1 matches loopback before dstdomain allow")
	}
}

func TestDecideDSTLiteralIPUnchanged(t *testing.T) {
	e := mustEngineFromSquid(t, `
acl bad dst 203.0.113.0/24
acl ok dstdomain ok.com
http_access deny bad
http_access allow ok
`)
	id := auth.Identity{Username: "u"}
	if e.Decide(id, acl.RequestFields{SNI: "203.0.113.9", Path: "/"}) {
		t.Fatal("dst cidr via sni literal expected deny")
	}
}
