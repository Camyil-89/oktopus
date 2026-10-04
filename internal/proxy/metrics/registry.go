package metrics

import (
	"context"
	"sort"
	"sync"

	"github.com/google/uuid"

	"oktopus/internal/proxy/instancectx"
)

var defaultRegistry = NewRegistry()

func NewRegistry() *Registry {
	return &Registry{byID: make(map[uuid.UUID]*Collector)}
}

type Registry struct {
	mu   sync.Mutex
	byID map[uuid.UUID]*Collector
}

func DefaultRegistry() *Registry {
	return defaultRegistry
}

func (r *Registry) Collector(id uuid.UUID) *Collector {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.byID[id]
	if !ok {
		c = NewCollector()
		r.byID[id] = c
	}
	return c
}

func (r *Registry) Drop(id uuid.UUID) {
	r.mu.Lock()
	delete(r.byID, id)
	r.mu.Unlock()
}

func collectorFromContext(ctx context.Context) *Collector {
	if id, ok := instancectx.ID(ctx); ok {
		return defaultRegistry.Collector(id)
	}
	return defaultCollector
}

const aggregateWindowSec = 300 // 5m, must match metrics.window

// AverageSnapshots — сводка по инстансам для дашборда (сумма счётчиков, merge графиков).
func AverageSnapshots(snaps []Snapshot) Snapshot {
	return AggregateSnapshots(snaps)
}

func AggregateSnapshots(snaps []Snapshot) Snapshot {
	if len(snaps) == 0 {
		return Snapshot{}
	}
	if len(snaps) == 1 {
		return snaps[0]
	}
	var out Snapshot
	var totalDecisions int64
	var weightedDecide, weightedInspect, weightedPolicy float64
	for _, s := range snaps {
		req5m := s.Allowed5m + s.Denied5m
		totalDecisions += req5m
		weightedDecide += s.DecideDurationUsAvg5m * float64(req5m)
		weightedInspect += s.InspectDurationUsAvg5m * float64(req5m)
		weightedPolicy += s.PolicyDurationUsAvg5m * float64(req5m)

		out.Requests1m += s.Requests1m
		out.Allowed5m += s.Allowed5m
		out.Denied5m += s.Denied5m
		out.RequestsPerSecNow5s += s.RequestsPerSecNow5s
		out.ActiveConnections += s.ActiveConnections
		out.ActiveWebSocketConnections += s.ActiveWebSocketConnections
		out.AccessLogQueuePending += s.AccessLogQueuePending
		out.UniqueUsers5m += s.UniqueUsers5m
		out.UniqueSources5m += s.UniqueSources5m

		out.DecideDurationUsP955m = max64(out.DecideDurationUsP955m, s.DecideDurationUsP955m)
		out.DecideDurationUsP995m = max64(out.DecideDurationUsP995m, s.DecideDurationUsP995m)
		out.DecideDurationUsMax5m = max64(out.DecideDurationUsMax5m, s.DecideDurationUsMax5m)
		out.InspectDurationUsP955m = max64(out.InspectDurationUsP955m, s.InspectDurationUsP955m)
		out.InspectDurationUsP995m = max64(out.InspectDurationUsP995m, s.InspectDurationUsP995m)
		out.PolicyDurationUsP955m = max64(out.PolicyDurationUsP955m, s.PolicyDurationUsP955m)
		out.PolicyDurationUsP995m = max64(out.PolicyDurationUsP995m, s.PolicyDurationUsP995m)
	}
	out.RequestsPerSecAvg5m = float64(out.Allowed5m+out.Denied5m) / float64(aggregateWindowSec)
	if totalDecisions > 0 {
		out.DecideDurationUsAvg5m = weightedDecide / float64(totalDecisions)
		out.InspectDurationUsAvg5m = weightedInspect / float64(totalDecisions)
		out.PolicyDurationUsAvg5m = weightedPolicy / float64(totalDecisions)
	}
	out.Buckets10s = mergeBuckets10s(snaps)
	for _, s := range snaps {
		out.BytesTotalUpAllow += s.BytesTotalUpAllow
		out.BytesTotalUpDeny += s.BytesTotalUpDeny
		out.BytesTotalDownAllow += s.BytesTotalDownAllow
		out.BytesTotalDownDeny += s.BytesTotalDownDeny
	}
	return out
}

func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func mergeBuckets10s(snaps []Snapshot) []TrafficBucket {
	byT := make(map[int64]TrafficBucket)
	var order []int64
	for _, s := range snaps {
		for _, b := range s.Buckets10s {
			if existing, ok := byT[b.T]; ok {
				existing.Requests += b.Requests
				existing.Allow += b.Allow
				existing.Deny += b.Deny
				existing.BytesUp += b.BytesUp
				existing.BytesDown += b.BytesDown
				existing.BytesUpAllow += b.BytesUpAllow
				existing.BytesUpDeny += b.BytesUpDeny
				existing.BytesDownAllow += b.BytesDownAllow
				existing.BytesDownDeny += b.BytesDownDeny
				byT[b.T] = existing
				continue
			}
			byT[b.T] = b
			order = append(order, b.T)
		}
	}
	if len(order) == 0 {
		return nil
	}
	sort.Slice(order, func(i, j int) bool { return order[i] < order[j] })
	out := make([]TrafficBucket, len(order))
	for i, t := range order {
		out[i] = byT[t]
	}
	return out
}
