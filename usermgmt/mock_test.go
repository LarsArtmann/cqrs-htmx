package usermgmt

import (
	"context"
	"errors"
	"sync"
	"time"
)

// mockSessionStore is a configurable test double for SessionStore.
type mockSessionStore struct {
	mu               sync.RWMutex
	CreateFn         func(ctx context.Context, userID UserID, ttl time.Duration) (*Session, error)
	FindFn           func(ctx context.Context, token string) (*Session, error)
	DeleteFn         func(ctx context.Context, token string) error
	DeleteByUserIDFn func(ctx context.Context, userID UserID) error
}

func (m *mockSessionStore) Create(
	ctx context.Context, userID UserID, ttl time.Duration,
) (*Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.CreateFn != nil {
		return m.CreateFn(ctx, userID, ttl)
	}
	s, _ := NewSession(userID, ttl)
	return s, nil
}

func (m *mockSessionStore) Find(ctx context.Context, token string) (*Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.FindFn != nil {
		return m.FindFn(ctx, token)
	}
	return nil, ErrSessionNotFound
}

func (m *mockSessionStore) Delete(ctx context.Context, token string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.DeleteFn != nil {
		return m.DeleteFn(ctx, token)
	}
	return nil
}

func (m *mockSessionStore) DeleteByUserID(ctx context.Context, userID UserID) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.DeleteByUserIDFn != nil {
		return m.DeleteByUserIDFn(ctx, userID)
	}
	return nil
}

// failingDeleteByUserIDSessionStore returns a SessionStore whose
// DeleteByUserID always returns the given error. Used to verify callers
// handle session-store failures gracefully.
func failingDeleteByUserIDSessionStore(msg string) *mockSessionStore {
	return &mockSessionStore{
		DeleteByUserIDFn: func(context.Context, UserID) error {
			return errors.New(msg)
		},
	}
}
