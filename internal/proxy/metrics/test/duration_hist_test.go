package metrics_test

import (
	"testing"
	"time"

	"oktopus/internal/proxy/metrics"
)

func TestDurationHistQuantileUniform(t *testing.T) {
	t.Parallel()
	c := metrics.NewCollector()
	const n = 10_000
	for i := 0; i < n; i++ {
		c.RecordDecision(true, 20*time.Microsecond, metrics.DecideParts{}, "", "")
	}
	s := c.TrafficSnapshot()
	if got := s.DecideDurationUsP955m; got < 18 || got > 26 {
		t.Fatalf("p95 us: got %d want ~20", got)
	}
	if got := s.DecideDurationUsP995m; got < 18 || got > 26 {
		t.Fatalf("p99 us: got %d want ~20", got)
	}
}

func TestMeanNsToUsExact(t *testing.T) {
	t.Parallel()
	got := metrics.MeanNsToUsExact(20_500, 10)
	if got < 2.04 || got > 2.06 {
		t.Fatalf("mean us: got %v want ~2.05", got)
	}
}
