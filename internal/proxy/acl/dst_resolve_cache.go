package acl

import (
	"context"
	"net"
	"strings"
	"sync"
	"time"
)

const dstResolveCacheTTL = 3 * time.Minute

type dstResolveCache struct {
	mu      sync.Mutex
	entries map[string]dstResolveCacheEntry
}

type dstResolveCacheEntry struct {
	ips     []net.IP
	expires time.Time
}

var dstResolveCacheStore dstResolveCache

func dstResolveCacheKey(host string) string {
	return strings.ToLower(strings.TrimSpace(host))
}

func cloneIPs(ips []net.IP) []net.IP {
	if len(ips) == 0 {
		return nil
	}
	out := make([]net.IP, len(ips))
	copy(out, ips)
	return out
}

func (c *dstResolveCache) ensureEntries() {
	if c.entries == nil {
		c.entries = make(map[string]dstResolveCacheEntry)
	}
}

func clearDstResolveCache() {
	dstResolveCacheStore.mu.Lock()
	dstResolveCacheStore.entries = make(map[string]dstResolveCacheEntry)
	dstResolveCacheStore.mu.Unlock()
}

func lookupDstIPsCached(ctx context.Context, host string) ([]net.IP, error) {
	key := dstResolveCacheKey(host)
	if key == "" {
		return nil, nil
	}
	now := time.Now()
	dstResolveCacheStore.mu.Lock()
	dstResolveCacheStore.ensureEntries()
	if e, ok := dstResolveCacheStore.entries[key]; ok {
		if now.Before(e.expires) {
			ips := cloneIPs(e.ips)
			dstResolveCacheStore.mu.Unlock()
			return ips, nil
		}
		delete(dstResolveCacheStore.entries, key)
	}
	dstResolveCacheStore.mu.Unlock()

	ips, err := dstLookup(ctx, host)
	if err != nil || len(ips) == 0 {
		return ips, err
	}
	stored := cloneIPs(ips)
	dstResolveCacheStore.mu.Lock()
	dstResolveCacheStore.entries[key] = dstResolveCacheEntry{
		ips:     stored,
		expires: now.Add(dstResolveCacheTTL),
	}
	dstResolveCacheStore.mu.Unlock()
	return cloneIPs(stored), nil
}
