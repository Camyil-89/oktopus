package middleware

import (
	"sync"
	"time"
)

const (
	lockoutFailureWindow    = 30 * time.Second
	lockoutFailureThreshold = 3
	lockoutDuration         = 5 * time.Minute
)

// LoginLockout глобальная блокировка входа после серии неудачных попыток с любого клиента.
type LoginLockout struct {
	mu           sync.Mutex
	failures     []time.Time
	blockedUntil time.Time
}

// Blocked сообщает, активна ли блокировка входа в момент now.
func (l *LoginLockout) Blocked(now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.pruneExpired(now)
	return now.Before(l.blockedUntil)
}

// RecordFailure учитывает неудачную попытку входа; при пороге включает блокировку.
func (l *LoginLockout) RecordFailure(now time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if now.Before(l.blockedUntil) {
		return
	}
	cutoff := now.Add(-lockoutFailureWindow)
	kept := make([]time.Time, 0, len(l.failures)+1)
	for _, t := range l.failures {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	kept = append(kept, now)
	l.failures = kept
	if len(l.failures) >= lockoutFailureThreshold {
		l.blockedUntil = now.Add(lockoutDuration)
		l.failures = nil
	}
}

func (l *LoginLockout) pruneExpired(now time.Time) {
	if l.blockedUntil.IsZero() {
		return
	}
	if now.Before(l.blockedUntil) {
		return
	}
	l.blockedUntil = time.Time{}
	l.failures = nil
}
