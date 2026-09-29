package accesslog

import (
	"context"
	"sync"
	"time"
)

// FlushFunc сохраняет пачку записей в БД (или другое хранилище).
type FlushFunc func(ctx context.Context, batch []Entry) error

// BatcherConfig настраивает асинхронный буфер.
type BatcherConfig struct {
	FlushInterval time.Duration // по умолчанию 3s
	MaxBatch      int           // по умолчанию 500
	OnFlushError  func(error)
}

// Batcher накапливает Entry в памяти и периодически вызывает flush.
type Batcher struct {
	cfg   BatcherConfig
	flush FlushFunc

	inMu  sync.Mutex
	inbox []Entry
	wake  chan struct{}

	depthMu      sync.Mutex
	pendingCount int

	stopCh chan struct{}
	wg     sync.WaitGroup
}

const flushTimeout = 30 * time.Second

// NewBatcher запускает фоновую горутину. Остановка и финальный сброс в БД — через Close().
func NewBatcher(flush FlushFunc, cfg BatcherConfig) *Batcher {
	if flush == nil {
		flush = func(context.Context, []Entry) error { return nil }
	}
	if cfg.FlushInterval <= 0 {
		cfg.FlushInterval = 3 * time.Second
	}
	if cfg.MaxBatch <= 0 {
		cfg.MaxBatch = 5000
	}
	b := &Batcher{
		cfg:    cfg,
		flush:  flush,
		wake:   make(chan struct{}, 1),
		stopCh: make(chan struct{}),
	}
	b.wg.Add(1)
	go b.loop()
	return b
}

func (b *Batcher) Record(e Entry) {
	if e.At.IsZero() {
		e.At = time.Now()
	}
	b.inMu.Lock()
	b.inbox = append(b.inbox, e)
	b.inMu.Unlock()
	select {
	case b.wake <- struct{}{}:
	default:
	}
}

// Close останавливает приём, сбрасывает очередь в БД и дожидается завершения.
func (b *Batcher) Close() {
	close(b.stopCh)
	b.wg.Wait()
}

// QueueDepth — записи в очереди (inbox + буфер перед flush).
func (b *Batcher) QueueDepth() int {
	if b == nil {
		return 0
	}
	b.inMu.Lock()
	inbox := len(b.inbox)
	b.inMu.Unlock()
	b.depthMu.Lock()
	pending := b.pendingCount
	b.depthMu.Unlock()
	return inbox + pending
}

func (b *Batcher) setPendingCount(n int) {
	b.depthMu.Lock()
	b.pendingCount = n
	b.depthMu.Unlock()
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
}

func (b *Batcher) pullInbox() []Entry {
	b.inMu.Lock()
	defer b.inMu.Unlock()
	if len(b.inbox) == 0 {
		return nil
	}
	out := b.inbox
	b.inbox = nil
	return out
}

func (b *Batcher) loop() {
	defer b.wg.Done()
	ticker := time.NewTicker(b.cfg.FlushInterval)
	defer ticker.Stop()

	pending := make([]Entry, 0, b.cfg.MaxBatch)
	flush := func() {
		if len(pending) == 0 {
			return
		}
		batch := pending
		pending = make([]Entry, 0, b.cfg.MaxBatch)
		b.setPendingCount(len(pending))
		b.flushBatch(batch)
	}

	appendInbox := func() {
		if chunk := b.pullInbox(); len(chunk) > 0 {
			pending = append(pending, chunk...)
			b.setPendingCount(len(pending))
		}
	}

	for {
		select {
		case <-b.stopCh:
			appendInbox()
			flush()
			return
		case <-ticker.C:
			appendInbox()
			flush()
		case <-b.wake:
			appendInbox()
			if len(pending) >= b.cfg.MaxBatch {
				flush()
			}
		}
	}
}
