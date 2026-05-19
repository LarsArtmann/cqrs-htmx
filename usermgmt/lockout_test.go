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
	_, _ = svc.Register(ctx, RegisterRequest{ID: "u1", Email: "a@b.com", Password: "secret12"})

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
