package middleware

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	lockoutFailureWindow    = 30 * time.Second
	lockoutFailureThreshold = 3
	lockoutDuration         = 5 * time.Minute
)

// LoginLockout блокировка входа по IP после серии неудачных попыток (состояние только в памяти процесса).
type LoginLockout struct {
	mu   sync.Mutex
	byIP map[string]*ipLockout
}

type ipLockout struct {
	failures     []time.Time
	blockedUntil time.Time
}

// ClientIP извлекает IP клиента из RemoteAddr запроса.
func ClientIP(r *http.Request) string {
	if r == nil {
		return ""
	}
	return ipKeyFromRemoteAddr(r.RemoteAddr)
}

func ipKeyFromRemoteAddr(remoteAddr string) string {
	remoteAddr = strings.TrimSpace(remoteAddr)
	if remoteAddr == "" {
		return ""
	}
	host := remoteAddr
	if h, _, err := net.SplitHostPort(remoteAddr); err == nil {
		host = h
	}
	host = strings.Trim(host, "[]")
	if ip := net.ParseIP(host); ip != nil {
		return ip.String()
	}
	return host
}

// Blocked сообщает, активна ли блокировка входа для ip в момент now.
func (l *LoginLockout) Blocked(now time.Time, ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	s := l.state(ip)
	s.pruneExpired(now)
	blocked := now.Before(s.blockedUntil)
	if s.idle() {
		l.drop(ip)
	}
	return blocked
}

// RecordFailure учитывает неудачную попытку входа с ip; при пороге включает блокировку для этого ip.
func (l *LoginLockout) RecordFailure(now time.Time, ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	s := l.state(ip)
	if now.Before(s.blockedUntil) {
		return
	}
	cutoff := now.Add(-lockoutFailureWindow)
	kept := make([]time.Time, 0, len(s.failures)+1)
	for _, t := range s.failures {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	kept = append(kept, now)
	s.failures = kept
	if len(s.failures) >= lockoutFailureThreshold {
		s.blockedUntil = now.Add(lockoutDuration)
		s.failures = nil
	}
}

func (l *LoginLockout) state(ip string) *ipLockout {
	if l.byIP == nil {
		l.byIP = make(map[string]*ipLockout)
	}
	s, ok := l.byIP[ip]
	if !ok {
		s = &ipLockout{}
		l.byIP[ip] = s
	}
	return s
}

func (l *LoginLockout) drop(ip string) {
	if l.byIP != nil {
		delete(l.byIP, ip)
	}
}

func (s *ipLockout) idle() bool {
	return s.blockedUntil.IsZero() && len(s.failures) == 0
}

func (s *ipLockout) pruneExpired(now time.Time) {
	if s.blockedUntil.IsZero() {
		return
	}
	if now.Before(s.blockedUntil) {
		return
	}
	s.blockedUntil = time.Time{}
	s.failures = nil
}
