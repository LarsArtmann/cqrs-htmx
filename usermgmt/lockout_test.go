package usermgmt

import (
	"testing"
	"time"
)

func TestAccountLockout_NotLockedByDefault(t *testing.T) {
	l := NewAccountLockout()
	if l.IsLocked("user@test.com") {
		t.Error("expected account to not be locked by default")
	}
}

func TestAccountLockout_LockAfterMaxAttempts(t *testing.T) {
	l := NewAccountLockout(LockoutConfig{MaxAttempts: 3, Duration: time.Hour})

	for i := range 3 {
		locked := l.RecordFailure("user@test.com")
		if i < 2 && locked {
			t.Errorf("locked prematurely at attempt %d", i+1)
		}
	}

	if !l.IsLocked("user@test.com") {
		t.Error("expected account to be locked after 3 failures")
	}
}

func TestAccountLockout_Reset(t *testing.T) {
	l := NewAccountLockout(LockoutConfig{MaxAttempts: 2, Duration: time.Hour})

	l.RecordFailure("user@test.com")
	l.RecordFailure("user@test.com")
	if !l.IsLocked("user@test.com") {
		t.Fatal("expected locked")
	}

	l.Reset("user@test.com")
	if l.IsLocked("user@test.com") {
		t.Error("expected unlocked after reset")
	}
}

func TestAccountLockout_ExpiredLockClears(t *testing.T) {
	l := NewAccountLockout(LockoutConfig{MaxAttempts: 1, Duration: 10 * time.Millisecond})

	l.RecordFailure("user@test.com")
	if !l.IsLocked("user@test.com") {
		t.Fatal("expected locked")
	}

	time.Sleep(20 * time.Millisecond)
	if l.IsLocked("user@test.com") {
		t.Error("expected lock to expire")
	}
}

func TestAccountLockout_EvictStale(t *testing.T) {
	l := NewAccountLockout(LockoutConfig{MaxAttempts: 1, Duration: 10 * time.Millisecond})

	l.RecordFailure("stale@test.com")
	time.Sleep(20 * time.Millisecond)

	evicted := l.EvictStale()
	if evicted == 0 {
		t.Error("expected at least 1 eviction")
	}
}

func TestAccountLockout_NormalizeEmail(t *testing.T) {
	l := NewAccountLockout(LockoutConfig{MaxAttempts: 1, Duration: time.Hour})

	l.RecordFailure("USER@TEST.COM")
	if !l.IsLocked("user@test.com") {
		t.Error("expected lock to be case-insensitive")
	}
}

func TestAccountLockout_RecordFailureAfterLockExpiry(t *testing.T) {
	l := NewAccountLockout(LockoutConfig{MaxAttempts: 1, Duration: 10 * time.Millisecond})

	locked := l.RecordFailure("user@test.com")
	if !locked {
		t.Fatal("expected to be locked after 1 attempt with MaxAttempts=1")
	}
	time.Sleep(20 * time.Millisecond)

	if l.IsLocked("user@test.com") {
		t.Error("lock should have expired")
	}

	// After lock expiry, the next failure should lock again (counter resets on lock expiry)
	locked = l.RecordFailure("user@test.com")
	if !locked {
		t.Error("expected lock again after expiry + new failure")
	}
}

func TestAccountLockout_CustomConfig(t *testing.T) {
	l := NewAccountLockout(LockoutConfig{MaxAttempts: 5, Duration: 30 * time.Minute})
	if l.config.MaxAttempts != 5 {
		t.Errorf("MaxAttempts = %d, want 5", l.config.MaxAttempts)
	}
	if l.config.Duration != 30*time.Minute {
		t.Errorf("Duration = %v, want 30m", l.config.Duration)
	}
}

func TestAccountLockout_DefaultConfig(t *testing.T) {
	l := NewAccountLockout()
	if l.config.MaxAttempts != defaultMaxAttempts {
		t.Errorf("default MaxAttempts = %d, want %d", l.config.MaxAttempts, defaultMaxAttempts)
	}
	if l.config.Duration != defaultLockoutDur {
		t.Errorf("default Duration = %v, want %v", l.config.Duration, defaultLockoutDur)
	}
}
