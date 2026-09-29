package middleware

import (
	"testing"
	"time"
)

func TestLoginLockout_threeFailuresInWindowBlocks(t *testing.T) {
	var l LoginLockout
	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	l.RecordFailure(base)
	l.RecordFailure(base.Add(5 * time.Second))
	if l.Blocked(base.Add(5 * time.Second)) {
		t.Fatal("expected no block after 2 failures")
	}

	l.RecordFailure(base.Add(10 * time.Second))
	if !l.Blocked(base.Add(10 * time.Second)) {
		t.Fatal("expected block after 3 failures in 30s")
	}
	third := base.Add(10 * time.Second)
	if !l.Blocked(third.Add(lockoutDuration - time.Second)) {
		t.Fatal("expected block until 5 minutes elapse")
	}
	if l.Blocked(third.Add(lockoutDuration)) {
		t.Fatal("expected block lifted after 5 minutes")
	}
}

func TestLoginLockout_oldFailuresExpireFromWindow(t *testing.T) {
	var l LoginLockout
	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	l.RecordFailure(base)
	l.RecordFailure(base.Add(5 * time.Second))
	l.RecordFailure(base.Add(lockoutFailureWindow + time.Second))
	if l.Blocked(base.Add(lockoutFailureWindow + time.Second)) {
		t.Fatal("third failure outside 30s window should not trigger block alone")
	}
}
