package ratelimit

import (
	"sync"
	"time"
)

const maxWaitSlice = 250 * time.Millisecond

type bucket struct {
	mu     sync.Mutex
	tokens int64
	last   time.Time
	seeded bool
}

func newBucket(initial int64) *bucket {
	b := &bucket{last: time.Now()}
	if initial > 0 {
		b.tokens = initial
		b.seeded = true
	}
	return b
}

func (b *bucket) wait(n int64, lim rateLimits) error {
	if lim.unlimited() || n <= 0 {
		return nil
	}
	remaining := n
	for remaining > 0 {
		chunk := remaining
		if lim.maxBytes > 0 && chunk > lim.maxBytes {
			chunk = lim.maxBytes
		}
		for {
			sleep := b.tryTake(chunk, lim)
			if sleep == 0 {
				remaining -= chunk
				break
			}
			if sleep > maxWaitSlice {
				sleep = maxWaitSlice
			}
			time.Sleep(sleep)
		}
	}
	return nil
}

func (b *bucket) tryTake(n int64, lim rateLimits) time.Duration {
	restore := lim.restoreBPS
	max := lim.maxBytes
	now := time.Now()

	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.seeded {
		b.seeded = true
		if lim.initial > 0 {
			b.tokens = lim.initial
		}
		b.last = now
	}

	if restore > 0 && max > 0 {
		elapsed := now.Sub(b.last)
		if elapsed > 0 {
			add := int64(float64(restore) * elapsed.Seconds())
			if add > 0 {
				b.tokens += add
				if b.tokens > max {
					b.tokens = max
				}
				b.last = now
			}
		}
	}

	if b.tokens >= n {
		b.tokens -= n
		return 0
	}
	need := n - b.tokens
	if restore <= 0 {
		return time.Second
	}
	sec := float64(need) / float64(restore)
	if sec < 0.001 {
		sec = 0.001
	}
	return time.Duration(sec * float64(time.Second))
}

func (b *bucket) topUp(lim rateLimits, dt time.Duration) {
	if lim.unlimited() {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.seeded {
		b.seeded = true
		if lim.initial > 0 {
			b.tokens = lim.initial
		}
		b.last = time.Now()
	}
	add := int64(float64(lim.restoreBPS) * dt.Seconds())
	if add <= 0 {
		return
	}
	b.tokens += add
	if b.tokens > lim.maxBytes {
		b.tokens = lim.maxBytes
	}
	b.last = time.Now()
}
