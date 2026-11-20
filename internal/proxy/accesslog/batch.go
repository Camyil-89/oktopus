package accesslog

import (
	"context"
	"sync"
	"time"
)

// FlushFunc сохраняет пачку записей в БД. Срез доступен до возврата из callback.
type FlushFunc func(ctx context.Context, batch []Entry) error

// BatcherConfig настраивает асинхронный буфер.
type BatcherConfig struct {
	FlushInterval    time.Duration  // базовое окно для высокой нагрузки, по умолчанию 3s
	MaxFlushInterval time.Duration  // окно при высокой нагрузке, по умолчанию 9s
	MaxBatch         int            // предельный размер одной отправки, по умолчанию 32_768
	RequestsPerSec   func() float64 // недавний RPS всех прокси; nil оставляет фиксированный режим
	OnFlushError     func(error)
}

// Batcher накапливает Entry в памяти и периодически вызывает flush.
type Batcher struct {
	cfg   BatcherConfig
	flush FlushFunc

	inMu    sync.Mutex
	inbox   []Entry
	inboxAt time.Time
	ready   [][]Entry
	pending int
	profile batchProfile
	spare   []Entry
	closed  bool
	wake    chan struct{}
	done    chan struct{}
}

type batchProfile struct {
	wait    time.Duration
	maxRows int
	reuse   bool
}

const (
	defaultFlushInterval    = 3 * time.Second
	defaultMaxFlushInterval = 9 * time.Second
	defaultMaxBatch         = 32_768
	flushTimeout            = 30 * time.Second
)

// NewBatcher запускает фоновую горутину. Остановка и финальный сброс в БД — через Close().
func NewBatcher(flush FlushFunc, cfg BatcherConfig) *Batcher {
	if flush == nil {
		flush = func(context.Context, []Entry) error { return nil }
	}
	if cfg.FlushInterval <= 0 {
		cfg.FlushInterval = defaultFlushInterval
	}
	if cfg.MaxFlushInterval <= 0 {
		cfg.MaxFlushInterval = defaultMaxFlushInterval
	}
	cfg.MaxFlushInterval = max(cfg.FlushInterval, cfg.MaxFlushInterval)
	if cfg.MaxBatch <= 0 {
		cfg.MaxBatch = defaultMaxBatch
	}
	b := &Batcher{
		cfg:     cfg,
		flush:   flush,
		wake:    make(chan struct{}, 1),
		done:    make(chan struct{}),
		profile: cfg.profile(0),
	}
	go b.loop()
	return b
}

func (b *Batcher) Record(e Entry) {
	if e.At.IsZero() {
		e.At = time.Now()
	}
	b.inMu.Lock()
	if b.closed {
		b.inMu.Unlock()
		return
	}
	first := len(b.inbox) == 0
	if first {
		b.inboxAt = time.Now()
		b.inbox, b.spare = b.spare, nil
	}
	b.inbox = append(b.inbox, e)
	b.pending++
	full := len(b.inbox) >= b.profile.maxRows
	if full {
		b.ready = append(b.ready, b.inbox)
		b.inbox = nil
	}
	b.inMu.Unlock()
	if first || full {
		b.notify()
	}
}

func (b *Batcher) notify() {
	select {
	case b.wake <- struct{}{}:
	default:
	}
}

// Close останавливает приём, сбрасывает очередь в БД и дожидается завершения.
func (b *Batcher) Close() {
	b.inMu.Lock()
	b.closed = true
	b.inMu.Unlock()
	b.notify()
	<-b.done
}

// QueueDepth — записи в очереди без пачки, которая уже отправляется.
func (b *Batcher) QueueDepth() int {
	if b == nil {
		return 0
	}
	b.inMu.Lock()
	defer b.inMu.Unlock()
	return b.pending
}

func (b *Batcher) flushBatch(batch []Entry) {
	if len(batch) == 0 {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), flushTimeout)
	defer cancel()
	if err := b.flush(ctx, batch); err != nil && b.cfg.OnFlushError != nil {
		b.cfg.OnFlushError(err)
	}
	if b.profile.reuse {
		clear(batch)
		b.inMu.Lock()
		// Keep at most one spare while work is arriving. The idle path releases it.
		if !b.closed && b.pending > 0 && cap(batch) > cap(b.spare) {
			b.spare = batch[:0]
		}
		b.inMu.Unlock()
	}
}

// nextBatch returns either work, the time to the next deadline, or completion after Close.
func (b *Batcher) nextBatch() (batch []Entry, delay time.Duration, done bool) {
	var rps float64
	if b.cfg.RequestsPerSec != nil {
		rps = b.cfg.RequestsPerSec()
	}
	b.inMu.Lock()
	defer b.inMu.Unlock()
	b.profile = b.cfg.profile(rps)
	if !b.profile.reuse || b.pending == 0 {
		b.spare = nil
	}
	if len(b.ready) != 0 {
		batch = b.ready[0]
		b.ready[0] = nil
		b.ready = b.ready[1:]
		if len(b.ready) == 0 {
			b.ready = nil
		}
		b.pending -= len(batch)
		return batch, 0, false
	}
	if len(b.inbox) == 0 {
		return nil, 0, b.closed
	}
	if !b.closed && len(b.inbox) < b.profile.maxRows {
		if delay := b.profile.wait - time.Since(b.inboxAt); delay > 0 {
			return nil, delay, false
		}
	}
	batch = b.inbox
	b.inbox = nil
	b.pending -= len(batch)
	return batch, 0, false
}

func (cfg BatcherConfig) profile(rps float64) batchProfile {
	base := cfg.FlushInterval
	p := batchProfile{wait: base, maxRows: cfg.MaxBatch}
	if cfg.RequestsPerSec == nil {
		return p
	}
	switch {
	case rps < 100:
		p.wait = min(time.Second, base)
		p.maxRows = min(512, cfg.MaxBatch)
	case rps <= 1_000:
		p.wait = min(2*time.Second, base)
		p.maxRows = min(4_096, cfg.MaxBatch)
	default:
		p.reuse = true
		target := rps * float64(time.Millisecond)
		if target >= float64(cfg.MaxFlushInterval) {
			p.wait = cfg.MaxFlushInterval
		} else {
			p.wait = max(base, time.Duration(target))
		}
	}
	if cfg.MaxFlushInterval == base {
		p.wait = base
	}
	return p
}

func (b *Batcher) loop() {
	defer close(b.done)
	timer := time.NewTimer(b.cfg.FlushInterval)
	timer.Stop()
	for {
		batch, delay, done := b.nextBatch()
		if done {
			return
		}
		if len(batch) != 0 {
			b.flushBatch(batch)
			continue
		}
		var deadline <-chan time.Time
		if delay > 0 {
			timer.Reset(delay)
			deadline = timer.C
		}
		select {
		case <-b.wake:
		case <-deadline:
		}
		timer.Stop()
	}
}
