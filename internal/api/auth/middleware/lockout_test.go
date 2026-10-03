package middleware

import (
	"testing"
	"time"
)

const testIP = "203.0.113.10"

func TestLoginLockout_threeFailuresInWindowBlocks(t *testing.T) {
	var l LoginLockout
	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	l.RecordFailure(base, testIP)
	l.RecordFailure(base.Add(5*time.Second), testIP)
	if l.Blocked(base.Add(5*time.Second), testIP) {
		t.Fatal("expected no block after 2 failures")
	}

	l.RecordFailure(base.Add(10*time.Second), testIP)
	if !l.Blocked(base.Add(10*time.Second), testIP) {
		t.Fatal("expected block after 3 failures in 30s")
	}
	third := base.Add(10 * time.Second)
	if !l.Blocked(third.Add(lockoutDuration-time.Second), testIP) {
		t.Fatal("expected block until 5 minutes elapse")
	}
	if l.Blocked(third.Add(lockoutDuration), testIP) {
		t.Fatal("expected block lifted after 5 minutes")
	}
}

func TestLoginLockout_oldFailuresExpireFromWindow(t *testing.T) {
	var l LoginLockout
	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	l.RecordFailure(base, testIP)
	l.RecordFailure(base.Add(5*time.Second), testIP)
	l.RecordFailure(base.Add(lockoutFailureWindow+time.Second), testIP)
	if l.Blocked(base.Add(lockoutFailureWindow+time.Second), testIP) {
		t.Fatal("third failure outside 30s window should not trigger block alone")
	}
}

func TestLoginLockout_perIPIsolation(t *testing.T) {
	var l LoginLockout
	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	attacker := "198.51.100.1"
	other := "198.51.100.2"

	for i := 0; i < lockoutFailureThreshold; i++ {
		l.RecordFailure(base.Add(time.Duration(i)*time.Second), attacker)
	}
	if !l.Blocked(base.Add(10*time.Second), attacker) {
		t.Fatal("expected attacker IP blocked")
	}
	if l.Blocked(base.Add(10*time.Second), other) {
		t.Fatal("expected other IP not blocked")
	}
}
