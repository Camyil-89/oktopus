package metrics_test

import (
	"testing"
	"testing/synctest"
	"time"

	"github.com/google/uuid"

	"oktopus/internal/proxy/metrics"
)

func TestRecentRequestsPerSecAggregatesCompletedSeconds(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		registry := metrics.NewRegistry()
		first := registry.Collector(uuid.New())
		second := registry.Collector(uuid.New())
		first.RecordDecision(true, 0, metrics.DecideParts{}, "", "")
		second.RecordDecision(false, 0, metrics.DecideParts{}, "", "")
		if got := registry.RecentRequestsPerSec(); got != 0 {
			t.Fatalf("unfinished bucket: got %v RPS, want 0", got)
		}
		time.Sleep(time.Second)
		if got := registry.RecentRequestsPerSec(); got != 0.4 {
			t.Fatalf("completed bucket: got %v RPS, want 0.4", got)
		}
		time.Sleep(5 * time.Second)
		if got := registry.RecentRequestsPerSec(); got != 0 {
			t.Fatalf("expired bucket: got %v RPS, want 0", got)
		}
	})
}
