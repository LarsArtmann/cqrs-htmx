package usermgmt

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/event"
)

// UserStore is the persistence interface for User aggregates.
type UserStore interface {
	FindByID(ctx context.Context, id UserID) (*User, error)
	FindByEmail(ctx context.Context, email string) (*User, error)
	Save(ctx context.Context, user *User) error
	Create(ctx context.Context, user *User) error
	Delete(ctx context.Context, id UserID) error
}

// SessionStore is the persistence interface for Session entities.
type SessionStore interface {
	Create(ctx context.Context, userID UserID, ttl time.Duration) (*Session, error)
	Find(ctx context.Context, token string) (*Session, error)
	Delete(ctx context.Context, token string) error
	DeleteByUserID(ctx context.Context, userID UserID) error
}

// InMemoryUserStore is a thread-safe, in-memory implementation of UserStore.
// It maintains an email index for O(1) lookups by email.
//
// Warning: Not suitable for production. Data is lost on process restart and memory
// grows unbounded as users are added. Use a persistent store (SQL, etc.) in production.
type InMemoryUserStore struct {
	mu     sync.RWMutex
	users  map[UserID]*User
	emails map[string]UserID
}

// NewInMemoryUserStore creates an empty InMemoryUserStore.
func NewInMemoryUserStore() *InMemoryUserStore {
	return &InMemoryUserStore{
		users:  make(map[UserID]*User),
		emails: make(map[string]UserID),
	}
}

// FindByID returns the user with the given ID, or ErrUserNotFound.
func (s *InMemoryUserStore) FindByID(_ context.Context, id UserID) (*User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.users[id]
	if !ok {
		return nil, ErrUserNotFound
	}
	return u, nil
}

// FindByEmail returns the user with the given email, or ErrUserNotFound.
func (s *InMemoryUserStore) FindByEmail(_ context.Context, email string) (*User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, ok := s.emails[email]
	if !ok {
		return nil, ErrUserNotFound
	}
	return s.users[id], nil
}

// Save updates an existing user. It returns ErrEmailExists if another user
// already claims the email.
func (s *InMemoryUserStore) Save(_ context.Context, user *User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for email, id := range s.emails {
		if id == user.ID && email != user.Email {
			delete(s.emails, email)
		}
	}
	if otherID, taken := s.emails[user.Email]; taken && otherID != user.ID {
		return ErrEmailExists
	}
	user.UpdatedAt = time.Now().UTC()
	s.users[user.ID] = user
	s.emails[user.Email] = user.ID
	return nil
}

// Create atomically inserts a new user. It returns ErrEmailExists if the email
// is already taken, or an error if the user ID already exists.
func (s *InMemoryUserStore) Create(_ context.Context, user *User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.emails[user.Email]; ok {
		return ErrEmailExists
	}
	if _, ok := s.users[user.ID]; ok {
		return ErrUserIDExists
	}
	user.UpdatedAt = time.Now().UTC()
	s.users[user.ID] = user
	s.emails[user.Email] = user.ID
	return nil
}

// Delete removes the user and its email index entry.
func (s *InMemoryUserStore) Delete(_ context.Context, id UserID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if u, ok := s.users[id]; ok {
		delete(s.emails, u.Email)
	}
	delete(s.users, id)
	return nil
}

// Count returns the number of stored users.
func (s *InMemoryUserStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.users)
}

// InMemorySessionStore is a thread-safe, in-memory implementation of SessionStore.
//
// Warning: Not suitable for production. Sessions are lost on process restart.
// Expired sessions accumulate unless EvictExpired is called periodically.
// Use a persistent store (Redis, SQL, etc.) in production.
type InMemorySessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

// NewInMemorySessionStore creates an empty InMemorySessionStore.
func NewInMemorySessionStore() *InMemorySessionStore {
	return &InMemorySessionStore{
		sessions: make(map[string]*Session),
	}
}

// Create generates a new session for the user with the given TTL.
func (s *InMemorySessionStore) Create(
	_ context.Context, userID UserID, ttl time.Duration,
) (*Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	session, err := NewSession(userID, ttl)
	if err != nil {
		return nil, event.NewTransient("session_create_failed",
			fmt.Sprintf("create session for user %q", userID)).WithCause(err)
	}
	s.sessions[session.Token] = session
	return session, nil
}

// Find returns the session for the given token, or ErrSessionNotFound.
func (s *InMemorySessionStore) Find(_ context.Context, token string) (*Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, ok := s.sessions[token]
	if !ok {
		return nil, ErrSessionNotFound
	}
	return session, nil
}

// Delete removes the session with the given token.
func (s *InMemorySessionStore) Delete(_ context.Context, token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, token)
	return nil
}

// DeleteByUserID removes all sessions belonging to the given user.
func (s *InMemorySessionStore) DeleteByUserID(_ context.Context, userID UserID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for token, session := range s.sessions {
		if session.UserID == userID {
			delete(s.sessions, token)
		}
	}
	return nil
}

// EvictExpired removes all expired sessions from the store and returns the
// number of sessions evicted. This is a maintenance method for the in-memory
// store — call periodically to prevent unbounded memory growth.
func (s *InMemorySessionStore) EvictExpired() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	evicted := 0
	for token, session := range s.sessions {
		if now.After(session.ExpiresAt) {
			delete(s.sessions, token)
			evicted++
		}
	}
	return evicted
}

// Count returns the number of active sessions.
func (s *InMemorySessionStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.sessions)
}
