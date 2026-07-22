package usermgmt

import (
	"strings"
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

// normalizeEmail lowercases and trims whitespace from an email address.
func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
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
	normalized := normalizeEmail(email)

	l.mu.RLock()
	lockedAt, ok := l.lockedAt[normalized]
	l.mu.RUnlock()
	if !ok {
		return false
	}
	if time.Since(lockedAt) > l.config.Duration {
		l.mu.Lock()
		defer l.mu.Unlock()
		// Double-check after acquiring write lock.
		if lockedAt2, ok2 := l.lockedAt[normalized]; ok2 &&
			time.Since(lockedAt2) > l.config.Duration {
			delete(l.lockedAt, normalized)
			delete(l.attempts, normalized)
		}
		return false
	}
	return true
}

// normalizeAndLock normalizes the email and acquires the write lock.
// Returns the normalized email and an unlock function the caller must defer.
func (l *AccountLockout) normalizeAndLock(email string) (string, func()) {
	normalized := normalizeEmail(email)
	l.mu.Lock()
	return normalized, l.mu.Unlock
}

// RecordFailure increments the failure counter for the email and returns true
// if the account has just been locked (i.e. the threshold was reached).
func (l *AccountLockout) RecordFailure(email string) bool {
	normalized, unlock := l.normalizeAndLock(email)
	defer unlock()
	if lockedAt, ok := l.lockedAt[normalized]; ok && time.Since(lockedAt) > l.config.Duration {
		delete(l.lockedAt, normalized)
		l.attempts[normalized] = 0
	}
	l.attempts[normalized]++
	if l.attempts[normalized] >= l.config.MaxAttempts {
		l.lockedAt[normalized] = time.Now()
		return true
	}
	return false
}

// Reset clears the failure counter and lockout for the given email.
func (l *AccountLockout) Reset(email string) {
	normalized, unlock := l.normalizeAndLock(email)
	defer unlock()
	delete(l.attempts, normalized)
	delete(l.lockedAt, normalized)
}

// EvictStale removes expired lockouts and stale attempt counters (entries with
// zero attempts that were never locked). Returns the number of entries evicted.
// Call periodically to prevent unbounded memory growth in high-cardinality scenarios.
func (l *AccountLockout) EvictStale() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	return deleteExpired(l.lockedAt, func(email string, lockedAt time.Time) bool {
		if now.Sub(lockedAt) <= l.config.Duration {
			return false
		}
		delete(l.attempts, email)
		return true
	})
}
