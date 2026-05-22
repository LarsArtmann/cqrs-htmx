package usermgmt

import (
	"sync"
	"time"
)

const (
	defaultMaxAttempts = 5
	defaultLockoutDur  = 15 * time.Minute
)

// LockoutConfig controls the account lockout behaviour.
type LockoutConfig struct {
	// MaxAttempts is the number of failed logins before the account is locked.
	MaxAttempts uint
	// Duration is how long the lockout lasts before it automatically resets.
	Duration time.Duration
}

// AccountLockout tracks failed login attempts per email and enforces temporary lockouts.
// Lockout state is held in-memory and lost on process restart. The internal maps grow
// with unique email addresses — call EvictStale periodically to remove expired lockouts
// and abandoned attempt counters, or use a distributed store in production.
type AccountLockout struct {
	mu       sync.RWMutex
	config   LockoutConfig
	attempts map[string]uint
	lockedAt map[string]time.Time
}

// NewAccountLockout creates an AccountLockout. An optional LockoutConfig can be
// provided; zero-valued fields fall back to defaults (5 attempts, 15 minutes).
func NewAccountLockout(cfg ...LockoutConfig) *AccountLockout {
	config := LockoutConfig{
		MaxAttempts: defaultMaxAttempts,
		Duration:    defaultLockoutDur,
	}
	if len(cfg) > 0 {
		if cfg[0].MaxAttempts > 0 {
			config.MaxAttempts = cfg[0].MaxAttempts
		}
		if cfg[0].Duration > 0 {
			config.Duration = cfg[0].Duration
		}
	}
	return &AccountLockout{
		config:   config,
		attempts: make(map[string]uint),
		lockedAt: make(map[string]time.Time),
	}
}

// IsLocked reports whether the account for the given email is currently locked.
// Expired lockouts are automatically cleared.
func (l *AccountLockout) IsLocked(email string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	lockedAt, ok := l.lockedAt[email]
	if !ok {
		return false
	}
	if time.Since(lockedAt) > l.config.Duration {
		delete(l.lockedAt, email)
		delete(l.attempts, email)
		return false
	}
	return true
}

// RecordFailure increments the failure counter for the email and returns true
// if the account has just been locked (i.e. the threshold was reached).
func (l *AccountLockout) RecordFailure(email string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if lockedAt, ok := l.lockedAt[email]; ok && time.Since(lockedAt) > l.config.Duration {
		delete(l.lockedAt, email)
		l.attempts[email] = 0
	}
	l.attempts[email]++
	if l.attempts[email] >= l.config.MaxAttempts {
		l.lockedAt[email] = time.Now()
		return true
	}
	return false
}

// Reset clears the failure counter and lockout for the given email.
func (l *AccountLockout) Reset(email string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.attempts, email)
	delete(l.lockedAt, email)
}

// EvictStale removes expired lockouts and stale attempt counters (entries with
// zero attempts that were never locked). Returns the number of entries evicted.
// Call periodically to prevent unbounded memory growth in high-cardinality scenarios.
func (l *AccountLockout) EvictStale() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	evicted := 0
	for email, lockedAt := range l.lockedAt {
		if now.Sub(lockedAt) > l.config.Duration {
			delete(l.lockedAt, email)
			delete(l.attempts, email)
			evicted++
		}
	}
	return evicted
}
