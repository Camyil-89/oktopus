package metrics_test

import (
	"testing"

	"oktopus/internal/proxy/metrics"
)

func TestAggregateSnapshotsMergesBucketsAndSumsCounters(t *testing.T) {
	t.Parallel()
	b1 := []metrics.TrafficBucket{
		{T: 100, Requests: 2, Allow: 2, Deny: 0},
		{T: 110, Requests: 1, Allow: 0, Deny: 1},
	}
	b2 := []metrics.TrafficBucket{
		{T: 100, Requests: 3, Allow: 1, Deny: 2},
		{T: 120, Requests: 4, Allow: 4, Deny: 0},
	}
	a := metrics.Snapshot{
		Allowed5m:           3,
		Denied5m:            1,
		RequestsPerSecNow5s: 0.4,
		Buckets10s:          b1,
	}
	b := metrics.Snapshot{
		Allowed5m:           5,
		Denied5m:            2,
		RequestsPerSecNow5s: 0.6,
		Buckets10s:          b2,
	}
	out := metrics.AggregateSnapshots([]metrics.Snapshot{a, b})
	if out.Allowed5m != 8 || out.Denied5m != 3 {
		t.Fatalf("allow/deny: %+v", out)
	}
	if out.RequestsPerSecNow5s != 1.0 {
		t.Fatalf("rps now: %f want 1.0", out.RequestsPerSecNow5s)
	}
	wantAvg := float64(8+3) / 300
	if out.RequestsPerSecAvg5m != wantAvg {
		t.Fatalf("rps avg: %f want %f", out.RequestsPerSecAvg5m, wantAvg)
	}
	if len(out.Buckets10s) != 3 {
		t.Fatalf("buckets len: %d want 3", len(out.Buckets10s))
	}
	byT := map[int64]metrics.TrafficBucket{}
	for _, bk := range out.Buckets10s {
		byT[bk.T] = bk
	}
	if byT[100].Requests != 5 || byT[100].Allow != 3 || byT[100].Deny != 2 {
		t.Fatalf("bucket 100: %+v", byT[100])
	}
	if byT[110].Requests != 1 {
		t.Fatalf("bucket 110: %+v", byT[110])
	}
	if byT[120].Requests != 4 {
		t.Fatalf("bucket 120: %+v", byT[120])
	}
}
