package usermgmt

import (
	"context"
	"testing"
	"time"
)

func TestAccountLockout_RecordFailure_ResetsExpiredLockout(t *testing.T) {
	l := NewAccountLockout(LockoutConfig{MaxAttempts: 2, Duration: 50 * time.Millisecond})

	l.RecordFailure("a@b.com")
	l.RecordFailure("a@b.com")
	if !l.IsLocked("a@b.com") {
		t.Fatal("expected locked")
	}

	time.Sleep(100 * time.Millisecond)

	if l.IsLocked("a@b.com") {
		t.Error("expected lockout to have expired")
	}

	if l.RecordFailure("a@b.com") {
		t.Error("expected not locked on first failure after expiry reset")
	}
}

func TestAccountLockout_Reset(t *testing.T) {
	l := NewAccountLockout(LockoutConfig{MaxAttempts: 2, Duration: time.Hour})
	l.RecordFailure("r@test.com")
	l.Reset("r@test.com")
	if l.IsLocked("r@test.com") {
		t.Error("expected not locked after reset")
	}
}

func TestAccountLockout_IsLocked_NotLocked(t *testing.T) {
	l := NewAccountLockout()
	if l.IsLocked("never@test.com") {
		t.Error("expected not locked for unknown email")
	}
}

func TestRecordFailure_TriggersLock(t *testing.T) {
	l := NewAccountLockout(LockoutConfig{MaxAttempts: 3, Duration: time.Hour})
	if l.RecordFailure("a@b.com") {
		t.Error("expected not locked on first failure")
	}
	if l.RecordFailure("a@b.com") {
		t.Error("expected not locked on second failure")
	}
	if !l.RecordFailure("a@b.com") {
		t.Error("expected locked on third failure (threshold)")
	}
}

func TestService_Login_AccountLocked(t *testing.T) {
	svc, _ := NewService(ServiceConfig{
		BcryptCost: minBcryptCost,
		Lockout:    NewAccountLockout(LockoutConfig{MaxAttempts: 2, Duration: time.Hour}),
	})
	ctx := context.Background()

	registerTestUser(t, svc, "u1", "locked@test.com", "secret12")

	//nolint:errcheck // intentionally triggering lockout
	svc.Login(ctx, LoginRequest{Email: "locked@test.com", Password: "wrong1"})
	//nolint:errcheck // intentionally triggering lockout
	svc.Login(ctx, LoginRequest{Email: "locked@test.com", Password: "wrong2"})

	_, err := svc.Login(ctx, LoginRequest{Email: "locked@test.com", Password: "secret12"})
	assertErrorIs(t, err, ErrAccountLocked, "ErrAccountLocked")
}

func TestService_Login_LockoutReset(t *testing.T) {
	lockout := NewAccountLockout(LockoutConfig{MaxAttempts: 2, Duration: time.Hour})
	svc, _ := NewService(ServiceConfig{
		BcryptCost: minBcryptCost,
		Lockout:    lockout,
	})
	ctx := context.Background()
	registerTestUser(t, svc, "lr1", "lr@test.com", "secret12")

	_, _ = svc.Login(ctx, LoginRequest{Email: "lr@test.com", Password: "wrong1"})
	_, err := svc.Login(ctx, LoginRequest{Email: "lr@test.com", Password: "secret12"})
	if err != nil {
		t.Errorf("expected successful login after one failure, got %v", err)
	}
}
