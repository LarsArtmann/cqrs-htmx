package usermgmt

import (
	"context"
	"sync"
	"time"
)

// mockUserStore is a configurable test double for UserStore.
// Each method field can be set to return a specific result.
// Unset methods return zero values / nil.
type mockUserStore struct {
	mu            sync.RWMutex
	FindByIDFn    func(ctx context.Context, id UserID) (*User, error)
	FindByEmailFn func(ctx context.Context, email string) (*User, error)
	SaveFn        func(ctx context.Context, user *User) error
	CreateFn      func(ctx context.Context, user *User) error
	DeleteFn      func(ctx context.Context, id UserID) error
}

func (m *mockUserStore) FindByID(ctx context.Context, id UserID) (*User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.FindByIDFn != nil {
		return m.FindByIDFn(ctx, id)
	}
	return nil, ErrUserNotFound
}

func (m *mockUserStore) FindByEmail(ctx context.Context, email string) (*User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.FindByEmailFn != nil {
		return m.FindByEmailFn(ctx, email)
	}
	return nil, ErrUserNotFound
}

func (m *mockUserStore) Save(ctx context.Context, user *User) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.SaveFn != nil {
		return m.SaveFn(ctx, user)
	}
	return nil
}

func (m *mockUserStore) Create(ctx context.Context, user *User) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.CreateFn != nil {
		return m.CreateFn(ctx, user)
	}
	return nil
}

func (m *mockUserStore) Delete(ctx context.Context, id UserID) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.DeleteFn != nil {
		return m.DeleteFn(ctx, id)
	}
	return nil
}

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
