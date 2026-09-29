package startup

import (
	"context"
	"sync"
	"time"
)

// TimingsDTO — метрики запуска процесса serve для API.
type TimingsDTO struct {
	StartedAt     string `json:"started_at,omitempty"`
	ProxyReadyAt  string `json:"proxy_ready_at,omitempty"`
	ACLDBSyncedAt string `json:"acl_db_synced_at,omitempty"`
	ProxyReadyMs  *int64 `json:"proxy_ready_ms,omitempty"`
	ACLDBSyncedMs *int64 `json:"acl_db_sync_ms,omitempty"`
}

var (
	mu              sync.RWMutex
	processStarted  time.Time
	proxyReady      time.Time
	aclDBSynced     time.Time
	startedRecorded bool
	proxyReadyOnce  sync.Once
	aclSyncOnce     sync.Once

	remoteListPollStartupOnce sync.Once
	remoteListPollStartupCh   = make(chan struct{})
)

// MarkProcessStarted фиксирует момент успешного старта serve (после db.Open).
func MarkProcessStarted(t time.Time) {
	mu.Lock()
	defer mu.Unlock()
	if startedRecorded {
		return
	}
	processStarted = t.UTC()
	startedRecorded = true
}

func releaseRemoteListPollStartup() {
	remoteListPollStartupOnce.Do(func() {
		close(remoteListPollStartupCh)
	})
}

// AllowRemoteListPollStartup — разрешить фоновый опрос remote-списков (если прокси выключен).
func AllowRemoteListPollStartup() {
	releaseRemoteListPollStartup()
}

// WaitRemoteListPollStartup блокируется до AllowRemoteListPollStartup или MarkProxyReady.
func WaitRemoteListPollStartup(ctx context.Context) error {
	select {
	case <-remoteListPollStartupCh:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// MarkProxyReady — первый раз, когда прокси начал принимать соединения.
func MarkProxyReady() {
	proxyReadyOnce.Do(func() {
		mu.Lock()
		proxyReady = time.Now().UTC()
		mu.Unlock()
		releaseRemoteListPollStartup()
	})
}

// MarkACLDBSynced — ACL опубликован в прокси (для метрик запуска, опционально).
func MarkACLDBSynced() {
	aclSyncOnce.Do(func() {
		mu.Lock()
		aclDBSynced = time.Now().UTC()
		mu.Unlock()
	})
}

// Timings возвращает снимок метрик для JSON API.
func Timings() TimingsDTO {
	mu.RLock()
	defer mu.RUnlock()
	var dto TimingsDTO
	if startedRecorded {
		dto.StartedAt = processStarted.Format(time.RFC3339Nano)
	}
	if !proxyReady.IsZero() {
		dto.ProxyReadyAt = proxyReady.Format(time.RFC3339Nano)
		if startedRecorded {
			ms := proxyReady.Sub(processStarted).Milliseconds()
			dto.ProxyReadyMs = &ms
		}
	}
	if !aclDBSynced.IsZero() {
		dto.ACLDBSyncedAt = aclDBSynced.Format(time.RFC3339Nano)
		if startedRecorded {
			ms := aclDBSynced.Sub(processStarted).Milliseconds()
			dto.ACLDBSyncedMs = &ms
		}
	}
	return dto
}
