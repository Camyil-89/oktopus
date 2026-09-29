package metrics

import (
	"sync/atomic"
	"time"
)

const trafficSnapshotRefresh = time.Second

var cachedTrafficSnapshot atomic.Value // Snapshot

func init() {
	cachedTrafficSnapshot.Store(defaultCollector.snapshot())
	go func() {
		t := time.NewTicker(trafficSnapshotRefresh)
		defer t.Stop()
		for range t.C {
			cachedTrafficSnapshot.Store(defaultCollector.snapshot())
		}
	}()
}

// SnapshotTraffic — последний фоновый снимок (~1 с); не считает агрегаты на каждый HTTP poll.
func SnapshotTraffic() Snapshot {
	if v := cachedTrafficSnapshot.Load(); v != nil {
		return v.(Snapshot)
	}
	return defaultCollector.snapshot()
}

// FreshTrafficSnapshot — снимок без кэша (актуальные байты и счётчики на момент вызова).
func FreshTrafficSnapshot() Snapshot {
	return defaultCollector.snapshot()
}
