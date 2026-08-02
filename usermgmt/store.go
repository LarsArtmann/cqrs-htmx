package usermgmt

import (
	"context"
	"sync"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
)

// Store is a generic persistence contract for ID-addressable entities.
//
// Domain-specific store interfaces can embed Store when they follow a standard
// CRUD-by-ID pattern (FindByID/Save/Create/Delete) and add their own
// specialized queries on top.
//
// The type parameters are intentionally permissive: T is the entity type and
// ID is the identifier type. The audit that introduced this interface noted
// that implementations should remain specific — Store only captures the
// shared contract.
type Store[T any, ID comparable] interface {
	FindByID(ctx context.Context, id ID) (T, error)
	Save(ctx context.Context, entity T) error
	Create(ctx context.Context, entity T) error
	Delete(ctx context.Context, id ID) error
}

// InMemoryStore is a thread-safe, generic in-memory implementation of Store.
//
// It is intended for tests and small prototypes. It is not suitable for
// production: data is lost on process restart and there is no persistence.
//
// The idOf function extracts the entity ID from a value. This keeps the store
// generic while still allowing typed IDs (e.g. brandid.ID) or simple scalars.
type InMemoryStore[T any, ID comparable] struct {
	mu    sync.RWMutex
	items map[ID]T
	idOf  func(T) ID
}

// NewInMemoryStore creates an empty InMemoryStore. idOf must return the stable
// identifier for an entity; it is called under lock during Save, Create, and FindByID.
func NewInMemoryStore[T any, ID comparable](idOf func(T) ID) *InMemoryStore[T, ID] {
	return &InMemoryStore[T, ID]{items: make(map[ID]T), idOf: idOf}
}

// FindByID returns the entity with the given ID. If no entity exists, it
// returns the zero value of T and a nil error.
func (s *InMemoryStore[T, ID]) FindByID(_ context.Context, id ID) (T, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	v, ok := s.items[id]
	if !ok {
		var zero T

		return zero, nil
	}

	return v, nil
}

// Save inserts or updates the entity under its ID.
func (s *InMemoryStore[T, ID]) Save(_ context.Context, entity T) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[s.idOf(entity)] = entity

	return nil
}

// Create inserts the entity. If an entity with the same ID already exists,
// it returns an error.
func (s *InMemoryStore[T, ID]) Create(_ context.Context, entity T) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := s.idOf(entity)
	if _, exists := s.items[id]; exists {
		return errorfamily.NewConflict("usermgmt.store.entity_exists", "entity already exists")
	}

	s.items[id] = entity

	return nil
}

// Delete removes the entity with the given ID.
func (s *InMemoryStore[T, ID]) Delete(_ context.Context, id ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.items, id)

	return nil
}

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
//cqrs-lint:ignore(S007) library default; consumers configure persistent session stores for production
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
	deleteKeyed(&s.mu, s.sessions, token)
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
	now := time.Now().UTC()
	return evictExpired(&s.mu, s.sessions, func(_ string, sess *Session) bool {
		return now.After(sess.ExpiresAt)
	})
}
