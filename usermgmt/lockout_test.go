package usermgmt

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestAccountLockout_NotLockedByDefault(t *testing.T) {
	l := NewAccountLockout()
	if l.IsLocked("test@example.com") {
		t.Error("expected not locked by default")
	}
}

func TestAccountLockout_LocksAfterMaxAttempts(t *testing.T) {
	l := NewAccountLockout(LockoutConfig{MaxAttempts: 3, Duration: time.Minute})

	for i := range 2 {
		if l.RecordFailure("test@example.com") {
			t.Errorf("expected not locked after attempt %d", i+1)
		}
	}

	if !l.RecordFailure("test@example.com") {
		t.Error("expected locked on 3rd failure")
	}

	if !l.IsLocked("test@example.com") {
		t.Error("expected account to be locked")
	}
}

func TestAccountLockout_ResetsOnSuccess(t *testing.T) {
	l := NewAccountLockout(LockoutConfig{MaxAttempts: 3, Duration: time.Minute})

	l.RecordFailure("test@example.com")
	l.RecordFailure("test@example.com")
	l.Reset("test@example.com")

	if l.IsLocked("test@example.com") {
		t.Error("expected not locked after reset")
	}
}

func TestAccountLockout_ExpiresAfterDuration(t *testing.T) {
	l := NewAccountLockout(LockoutConfig{MaxAttempts: 1, Duration: 50 * time.Millisecond})

	l.RecordFailure("test@example.com")
	if !l.IsLocked("test@example.com") {
		t.Error("expected locked")
	}

	time.Sleep(100 * time.Millisecond)
	if l.IsLocked("test@example.com") {
		t.Error("expected lockout to expire")
	}
}

func TestService_Login_AccountLockout(t *testing.T) {
	lockout := NewAccountLockout(LockoutConfig{MaxAttempts: 2, Duration: time.Minute})
	svc, _ := NewService(ServiceConfig{
		BcryptCost: minBcryptCost,
		Lockout:    lockout,
	})
	ctx := context.Background()
	_, _ = svc.Register(
		ctx,
		RegisterRequest{ID: NewUserID("u1"), Email: "a@b.com", Password: "secret12"},
	)

	_, err := svc.Login(ctx, LoginRequest{Email: "a@b.com", Password: "wrong1"})
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}

	_, err = svc.Login(ctx, LoginRequest{Email: "a@b.com", Password: "wrong2"})
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}

	_, err = svc.Login(ctx, LoginRequest{Email: "a@b.com", Password: "secret12"})
	if !errors.Is(err, ErrAccountLocked) {
		t.Errorf("expected ErrAccountLocked after max attempts, got %v", err)
	}
}

func TestAccountLockout_EvictStale(t *testing.T) {
	l := NewAccountLockout(LockoutConfig{MaxAttempts: 1, Duration: 50 * time.Millisecond})

	l.RecordFailure("expired@test.com")
	if !l.IsLocked("expired@test.com") {
		t.Fatal("expected locked")
	}

	time.Sleep(60 * time.Millisecond)

	l.RecordFailure("fresh@test.com")

	evicted := l.EvictStale()
	if evicted != 1 {
		t.Errorf("expected 1 eviction, got %d", evicted)
	}
	if l.IsLocked("expired@test.com") {
		t.Error("expected expired entry to be evicted")
	}
	if !l.IsLocked("fresh@test.com") {
		t.Error("expected fresh entry to remain")
	}

	if l.EvictStale() != 0 {
		t.Error("expected 0 evictions on clean state")
	}
}
