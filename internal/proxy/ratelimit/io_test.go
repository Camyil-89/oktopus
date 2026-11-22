package ratelimit

import (
	"bytes"
	"io"
	"testing"
	"time"
)

const (
	rateLimitTestBytes = 4 << 10
	rateLimitTestBPS   = 64 << 10
)

func newIOTestFlow() *Flow {
	rt := NewRuntime([]PoolDef{
		{},
		{
			Class:         1,
			IndRestoreBPS: rateLimitTestBPS,
			IndMaxBytes:   rateLimitTestBPS,
			Initial:       1024,
		},
	})
	return rt.Flow(1, nil)
}

func TestReaderLimitsDownloadRate(t *testing.T) {
	payload := bytes.Repeat([]byte("d"), rateLimitTestBytes)
	reader := newIOTestFlow().Reader(bytes.NewReader(payload))

	start := time.Now()
	got, err := io.ReadAll(reader)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("read limited download: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("download body mismatch: got %d bytes, want %d", len(got), len(payload))
	}
	assertRateLimitWait(t, elapsed)
}

func TestWriterLimitsUploadRate(t *testing.T) {
	payload := bytes.Repeat([]byte("u"), rateLimitTestBytes)
	var dst bytes.Buffer
	writer := newIOTestFlow().Writer(&dst)

	start := time.Now()
	n, err := writer.Write(payload)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("write limited upload: %v", err)
	}
	if n != len(payload) {
		t.Fatalf("upload wrote %d bytes, want %d", n, len(payload))
	}
	if !bytes.Equal(dst.Bytes(), payload) {
		t.Fatalf("upload body mismatch: got %d bytes, want %d", dst.Len(), len(payload))
	}
	assertRateLimitWait(t, elapsed)
}

func assertRateLimitWait(t *testing.T, elapsed time.Duration) {
	t.Helper()
	// The 1 KiB initial burst leaves about 3 KiB to replenish at 64 KiB/s
	// (roughly 47 ms); use a lower bound with ample scheduling tolerance.
	if elapsed < 25*time.Millisecond {
		t.Fatalf("rate limit did not delay transfer enough: %v", elapsed)
	}
	if elapsed > time.Second {
		t.Fatalf("rate limit delayed a 4 KiB transfer unexpectedly long: %v", elapsed)
	}
}
