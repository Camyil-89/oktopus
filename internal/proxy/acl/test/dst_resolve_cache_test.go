package acl_test

import (
	"context"
	"net"
	"sync/atomic"
	"testing"

	"oktopus/internal/proxy/acl"
)

func TestEnrichRequestFieldsDstCachesResolvedIPs(t *testing.T) {
	t.Cleanup(func() { acl.SetDstLookupForTest(nil) })

	var lookups atomic.Int32
	acl.SetDstLookupForTest(func(_ context.Context, host string) ([]net.IP, error) {
		lookups.Add(1)
		if host != "cached.example" {
			return nil, nil
		}
		return []net.IP{net.ParseIP("203.0.113.55")}, nil
	})

	base := acl.RequestFields{SNI: "cached.example", Path: "/"}
	f1 := acl.EnrichRequestFieldsDst(context.Background(), base)
	f2 := acl.EnrichRequestFieldsDst(context.Background(), base)

	if len(f1.DstResolved) != 1 || !f1.DstResolved[0].Equal(net.ParseIP("203.0.113.55")) {
		t.Fatalf("first resolve: %+v", f1.DstResolved)
	}
	if len(f2.DstResolved) != 1 || !f2.DstResolved[0].Equal(f1.DstResolved[0]) {
		t.Fatalf("second resolve: %+v", f2.DstResolved)
	}
	if got := lookups.Load(); got != 1 {
		t.Fatalf("expected 1 DNS lookup (cache hit on second), got %d", got)
	}
}
