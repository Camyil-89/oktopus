package metrics_test

import (
	"testing"
	"time"

	"oktopus/internal/proxy/metrics"
)

func TestPercentile95(t *testing.T) {
	t.Parallel()
	got := metrics.Percentile95([]int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10})
	if got != 10 {
		t.Fatalf("p95: got %d want 10", got)
	}
	if metrics.Percentile95(nil) != 0 {
		t.Fatal("empty p95 should be 0")
	}
}

func TestPercentile95UsFromNsSubMicrosecondMass(t *testing.T) {
	t.Parallel()
	const n = 100
	vals := make([]int64, n)
	for i := range vals {
		vals[i] = 500 // 500 ns
	}
	vals[n-1] = 3_800_000 // 3.8 ms outlier
	p95 := metrics.Percentile95UsFromNs(vals)
	if p95 == 0 {
		t.Fatal("p95 must not truncate sub-microsecond tail to 0 µs")
	}
	mean := metrics.MeanNsToUsRound(int64(n-1)*500+3_800_000, n)
	if mean < 30 || mean > 45 {
		t.Fatalf("mean us: got %d", mean)
	}
}

func TestCollectorP95UnderHighLoad(t *testing.T) {
	t.Parallel()
	c := metrics.NewCollector()
	const fast = 500 * time.Nanosecond
	const slow = 5 * time.Millisecond
	for i := 0; i < 80_000; i++ {
		c.RecordDecision(true, fast, metrics.DecideParts{}, "", "")
	}
	for i := 0; i < 200; i++ {
		c.RecordDecision(true, slow, metrics.DecideParts{}, "", "")
	}

	s := c.TrafficSnapshot()
	if s.DecideDurationUsP955m == 0 {
		t.Fatalf("p95 should reflect slow tail, got 0 (avg=%v max=%d)", s.DecideDurationUsAvg5m, s.DecideDurationUsMax5m)
	}
	if s.DecideDurationUsAvg5m <= 0 {
		t.Fatalf("avg us: %v", s.DecideDurationUsAvg5m)
	}
}

func TestCollectorSnapshot(t *testing.T) {
	t.Parallel()
	c := metrics.NewCollector()
	c.RecordDecision(true, 10*time.Microsecond, metrics.DecideParts{}, "alice", "1.2.3.4:1234")
	c.RecordDecision(false, 20*time.Microsecond, metrics.DecideParts{}, "bob", "1.2.3.4:5678")
	c.RecordDecision(true, 30*time.Microsecond, metrics.DecideParts{}, "alice", "1.2.3.4:9999")

	s := c.TrafficSnapshot()
	if s.Allowed5m != 2 || s.Denied5m != 1 {
		t.Fatalf("allow/deny: %+v", s)
	}
	if s.UniqueUsers5m != 2 || s.UniqueSources5m != 1 {
		t.Fatalf("unique: users=%d sources=%d", s.UniqueUsers5m, s.UniqueSources5m)
	}
	if s.DecideDurationUsAvg5m < 19.9 || s.DecideDurationUsAvg5m > 20.1 {
		t.Fatalf("avg us: %v", s.DecideDurationUsAvg5m)
	}
	if s.DecideDurationUsMax5m != 30 {
		t.Fatalf("max us: %d", s.DecideDurationUsMax5m)
	}
	if s.RequestsPerSecNow5s <= 0 {
		t.Fatalf("rps now: %f", s.RequestsPerSecNow5s)
	}
	if s.Requests1m != 3 {
		t.Fatalf("requests_1m: %d want 3", s.Requests1m)
	}
	const seriesBucketCount = 30
	if len(s.Buckets10s) != seriesBucketCount {
		t.Fatalf("buckets: got %d want %d", len(s.Buckets10s), seriesBucketCount)
	}
	var sum int64
	for _, b := range s.Buckets10s {
		sum += b.Requests
	}
	if sum != 3 {
		t.Fatalf("bucket requests sum: %d want 3", sum)
	}
}

func TestCollectorInspectDenyReclassify(t *testing.T) {
	t.Parallel()
	c := metrics.NewCollector()
	c.RecordDecision(true, 10*time.Microsecond, metrics.DecideParts{}, "", "")
	c.RecordInspectDeny()

	s := c.TrafficSnapshot()
	if s.Allowed5m != 0 || s.Denied5m != 1 {
		t.Fatalf("after inspect deny: allow=%d deny=%d", s.Allowed5m, s.Denied5m)
	}
	if s.Buckets10s[len(s.Buckets10s)-1].Allow != 0 || s.Buckets10s[len(s.Buckets10s)-1].Deny != 1 {
		t.Fatalf("bucket allow/deny not reclassified")
	}
	var sumReq int64
	for _, b := range s.Buckets10s {
		sumReq += b.Requests
	}
	if sumReq != 1 {
		t.Fatalf("requests count unchanged: %d", sumReq)
	}
}
