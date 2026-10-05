package acl_test

import (
	"context"
	"net"
	"testing"

	"oktopus/internal/proxy/acl"
	"oktopus/internal/proxy/auth"
)

func TestDstResolveFlipUsesCacheNotSecondAnswer(t *testing.T) {
	host := "flip.example"
	calls := 0
	acl.SetDstLookupForTest(func(ctx context.Context, h string) ([]net.IP, error) {
		if h != host {
			return nil, &net.DNSError{Err: "no such host", Name: h}
		}
		calls++
		if calls == 1 {
			return []net.IP{net.ParseIP("198.18.0.1")}, nil
		}
		return []net.IP{net.ParseIP("127.0.0.1")}, nil
	})
	defer acl.SetDstLookupForTest(nil)

	eng := mustEngineFromSquid(t, `
acl public dstdomain flip.example
acl loop dst 127.0.0.0/8
acl all all
http_access deny loop
http_access allow public
http_access deny all
`)
	id := auth.Identity{Username: "u"}

	f1 := acl.EnrichRequestFieldsDst(context.Background(), acl.RequestFields{SNI: host, Path: "/"})
	if !eng.Decide(id, f1) {
		t.Fatal("first: expected allow for 198.18.0.1")
	}
	f2 := acl.EnrichRequestFieldsDst(context.Background(), acl.RequestFields{SNI: host, Path: "/"})
	if !eng.Decide(id, f2) {
		t.Fatal("second: cache must keep first IPs (allow), not flip to loopback without re-resolve policy")
	}
	if calls != 1 {
		t.Fatalf("expected 1 DNS lookup (cache), got %d", calls)
	}
}
