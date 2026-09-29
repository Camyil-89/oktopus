package metrics_test

import (
	"testing"
	"time"

	"oktopus/internal/proxy/bytecount"
	"oktopus/internal/proxy/metrics"
)

func TestObserveBytesInBuckets(t *testing.T) {
	t.Parallel()
	sec := time.Now().Unix()
	bytecount.Rotate(sec)
	bytecount.ObserveUp(1000)
	bytecount.ObserveDown(2000)
	bytecount.Rotate(sec + 1)

	s := metrics.FreshTrafficSnapshot()
	var up, down int64
	for _, b := range s.Buckets10s {
		up += b.BytesUp
		down += b.BytesDown
	}
	if up != 1000 || down != 2000 {
		t.Fatalf("bucket totals: up=%d down=%d", up, down)
	}
}
