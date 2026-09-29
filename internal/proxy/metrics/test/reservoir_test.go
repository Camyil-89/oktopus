package metrics_test

import (
	"testing"

	"oktopus/internal/proxy/metrics"
)

func TestReservoirAppend(t *testing.T) {
	t.Parallel()
	var s []int
	for i := int64(1); i <= 1000; i++ {
		s = metrics.ReservoirAppend(s, int(i), 10, i)
	}
	if len(s) != 10 {
		t.Fatalf("len: got %d want 10", len(s))
	}
}
