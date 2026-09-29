package accesslog_test

import (
	"context"
	"testing"
	"time"

	"oktopus/internal/proxy/accesslog"
)

func TestBatcherRecordSetsAtBeforeFlush(t *testing.T) {
	t.Parallel()

	decided := time.Date(2026, 3, 30, 12, 0, 0, 0, time.UTC)
	var got []accesslog.Entry
	b := accesslog.NewBatcher(func(_ context.Context, batch []accesslog.Entry) error {
		got = append(got, batch...)
		return nil
	}, accesslog.BatcherConfig{FlushInterval: time.Hour, MaxBatch: 10})

	b.Record(accesslog.Entry{Action: accesslog.ActionDeny, At: decided})
	b.Close()

	if len(got) != 1 {
		t.Fatalf("batch len: %d", len(got))
	}
	if !got[0].At.Equal(decided) {
		t.Fatalf("At=%v want %v", got[0].At, decided)
	}
}
