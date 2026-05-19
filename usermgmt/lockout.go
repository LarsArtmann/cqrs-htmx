package usermgmt

import (
	"sync"
	"time"
)

const (
	defaultMaxAttempts = 5
	defaultLockoutDur  = 15 * time.Minute
)

type LockoutConfig struct {
	MaxAttempts int
	Duration    time.Duration
}

type AccountLockout struct {
	mu       sync.RWMutex
	config   LockoutConfig
	attempts map[string]int
	lockedAt map[string]time.Time
}

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
		attempts: make(map[string]int),
		lockedAt: make(map[string]time.Time),
	}
}

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

func (l *AccountLockout) Reset(email string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.attempts, email)
	delete(l.lockedAt, email)
}
