package metrics_test

import (
	"testing"
	"time"

	"oktopus/internal/proxy/metrics"
)

func TestRecentRingDropsOlderThan1000(t *testing.T) {
	t.Parallel()
	c := metrics.NewCollector()
	const fast = 500 * time.Nanosecond
	const slow = 5 * time.Millisecond
	for i := 0; i < 2000; i++ {
		c.RecordDecision(true, fast, metrics.DecideParts{}, "", "")
	}
	for i := 0; i < 100; i++ {
		c.RecordDecision(true, slow, metrics.DecideParts{}, "", "")
	}

	s := c.TrafficSnapshot()
	if s.DecideDurationUsP955m < 4000 {
		t.Fatalf("p95 should reflect slow tail in last 1000, got %d", s.DecideDurationUsP955m)
	}
}
