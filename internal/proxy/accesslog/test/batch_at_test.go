package accesslog_test

import (
	"context"
	"slices"
	"testing"
	"testing/synctest"
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

func TestBatcherFlushesAtThreshold(t *testing.T) {
	t.Parallel()

	flushed := make(chan []accesslog.Entry, 1)
	b := accesslog.NewBatcher(func(_ context.Context, batch []accesslog.Entry) error {
		flushed <- slices.Clone(batch)
		return nil
	}, accesslog.BatcherConfig{FlushInterval: time.Hour, MaxBatch: 2})
	t.Cleanup(b.Close)

	b.Record(accesslog.Entry{RuleRef: "first"})
	if got := b.QueueDepth(); got != 1 {
		t.Fatalf("queue depth = %d, want 1", got)
	}
	b.Record(accesslog.Entry{RuleRef: "second"})
	select {
	case batch := <-flushed:
		if len(batch) != 2 || batch[0].RuleRef != "first" || batch[1].RuleRef != "second" {
			t.Fatalf("unexpected batch: %+v", batch)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("batch did not flush at the configured threshold")
	}

	if got := b.QueueDepth(); got != 0 {
		t.Fatalf("queue depth after flush = %d, want 0", got)
	}
}

func TestBatcherAdaptsToTraffic(t *testing.T) {
	for _, tc := range []struct {
		name      string
		rps       float64
		wantDelay time.Duration
	}{
		{"low rate", 99, time.Second},
		{"moderate starts at 100", 100, 2 * time.Second},
		{"moderate includes 1000", 1_000, 2 * time.Second},
		{"high starts above 1000", 1_001, 3 * time.Second},
		{"high rate", 3_500, 3500 * time.Millisecond},
		{"maximum wait", 20_000, 9 * time.Second},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				start := time.Now()
				flushed := make(chan time.Duration, 1)
				b := accesslog.NewBatcher(func(_ context.Context, _ []accesslog.Entry) error {
					select {
					case flushed <- time.Since(start):
					default:
					}
					return nil
				}, accesslog.BatcherConfig{RequestsPerSec: func() float64 { return tc.rps }})
				defer b.Close()

				b.Record(accesslog.Entry{})
				time.Sleep(tc.wantDelay - time.Millisecond)
				select {
				case got := <-flushed:
					t.Fatalf("early flush at %v, want %v", got, tc.wantDelay)
				default:
				}
				time.Sleep(time.Millisecond)
				synctest.Wait()
				select {
				case got := <-flushed:
					if got != tc.wantDelay {
						t.Errorf("flush delay = %v, want %v", got, tc.wantDelay)
					}
				default:
					t.Error("batch was not flushed")
				}
			})
		})
	}
}

func TestBatcherCapsBatchesWhileFlushIsBlocked(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		release := make(chan struct{})
		var sizes []int
		b := accesslog.NewBatcher(func(_ context.Context, batch []accesslog.Entry) error {
			sizes = append(sizes, len(batch))
			<-release
			return nil
		}, accesslog.BatcherConfig{FlushInterval: time.Hour, MaxBatch: 2})

		for range 2 {
			b.Record(accesslog.Entry{})
		}
		synctest.Wait()
		for range 7 {
			b.Record(accesslog.Entry{})
		}
		depth := b.QueueDepth()
		close(release)
		b.Close()

		if depth != 7 {
			t.Errorf("queue depth = %d, want 7", depth)
		}
		if want := []int{2, 2, 2, 2, 1}; !slices.Equal(sizes, want) {
			t.Errorf("batch sizes = %v, want %v", sizes, want)
		}
	})
}

func TestBatcherReusesBufferDuringHighTraffic(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		type batchSnapshot struct {
			start *accesslog.Entry
			refs  [2]string
		}
		flushed := make(chan batchSnapshot, 1)
		proceed := make(chan struct{})
		b := accesslog.NewBatcher(func(_ context.Context, batch []accesslog.Entry) error {
			flushed <- batchSnapshot{&batch[0], [2]string{batch[0].RuleRef, batch[1].RuleRef}}
			<-proceed
			return nil
		}, accesslog.BatcherConfig{MaxBatch: 2, RequestsPerSec: func() float64 { return 3_500 }})
		defer func() {
			close(proceed)
			b.Close()
		}()
		b.Record(accesslog.Entry{RuleRef: "a1"})
		b.Record(accesslog.Entry{RuleRef: "a2"})
		first := <-flushed
		b.Record(accesslog.Entry{RuleRef: "b1"})
		proceed <- struct{}{}
		synctest.Wait()
		b.Record(accesslog.Entry{RuleRef: "b2"})
		second := <-flushed
		b.Record(accesslog.Entry{RuleRef: "c1"})
		b.Record(accesslog.Entry{RuleRef: "c2"})
		proceed <- struct{}{}
		third := <-flushed
		if first.start == second.start || first.start != third.start {
			t.Fatal("third batch must reuse the completed first batch's buffer")
		}
		if third.refs != [2]string{"c1", "c2"} {
			t.Fatalf("reused buffer contains %v", third.refs)
		}
	})
}

func BenchmarkBatcherFlush3000(b *testing.B) {
	e := accesslog.Entry{At: time.Now(), FullURLRequest: "http://origin/load"}
	b.ReportAllocs()
	for b.Loop() {
		batcher := accesslog.NewBatcher(nil, accesslog.BatcherConfig{FlushInterval: time.Hour})
		for range 3000 {
			batcher.Record(e)
		}
		batcher.Close()
	}
}
