package usermgmt

import (
	"context"
	"sync"
	"time"
)

// SessionStore is the persistence interface for Session entities.
type SessionStore interface {
	Create(ctx context.Context, session *Session) error
	Find(ctx context.Context, token string) (*Session, error)
	Delete(ctx context.Context, token string) error
	DeleteByUserID(ctx context.Context, userID UserID) error
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

// Create stores a pre-built session.
func (s *InMemorySessionStore) Create(_ context.Context, session *Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[session.Token] = session
	return nil
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
