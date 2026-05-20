package usermgmt

import (
	"sync"
	"time"

	"github.com/cockroachdb/errors"
)

// UserStore is the persistence interface for User aggregates.
type UserStore interface {
	FindByID(id UserID) (*User, error)
	FindByEmail(email string) (*User, error)
	Save(user *User) error
	Create(user *User) error
	Delete(id UserID) error
}

// SessionStore is the persistence interface for Session entities.
type SessionStore interface {
	Create(userID UserID, ttl time.Duration) (*Session, error)
	Find(token string) (*Session, error)
	Delete(token string) error
	DeleteByUserID(userID UserID) error
}

// InMemoryUserStore is a thread-safe, in-memory implementation of UserStore.
// It maintains an email index for O(1) lookups by email.
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
func (s *InMemoryUserStore) FindByID(id UserID) (*User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.users[id]
	if !ok {
		return nil, ErrUserNotFound
	}
	return u, nil
}

// FindByEmail returns the user with the given email, or ErrUserNotFound.
func (s *InMemoryUserStore) FindByEmail(email string) (*User, error) {
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
func (s *InMemoryUserStore) Save(user *User) error {
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
func (s *InMemoryUserStore) Create(user *User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.emails[user.Email]; ok {
		return ErrEmailExists
	}
	if _, ok := s.users[user.ID]; ok {
		return errors.Newf("user ID %s already exists", user.ID)
	}
	user.UpdatedAt = time.Now().UTC()
	s.users[user.ID] = user
	s.emails[user.Email] = user.ID
	return nil
}

// Delete removes the user and its email index entry.
func (s *InMemoryUserStore) Delete(id UserID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if u, ok := s.users[id]; ok {
		delete(s.emails, u.Email)
	}
	delete(s.users, id)
	return nil
}

// InMemorySessionStore is a thread-safe, in-memory implementation of SessionStore.
type InMemorySessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*Session
	ttl      time.Duration
}

// NewInMemorySessionStore creates an empty InMemorySessionStore with the default TTL.
func NewInMemorySessionStore() *InMemorySessionStore {
	return &InMemorySessionStore{
		sessions: make(map[string]*Session),
		ttl:      defaultSessionTTL,
	}
}

// WithTTL sets the default session TTL and returns the receiver for chaining.
func (s *InMemorySessionStore) WithTTL(ttl time.Duration) *InMemorySessionStore {
	s.ttl = ttl
	return s
}

// Create generates a new session for the user with the given TTL.
func (s *InMemorySessionStore) Create(userID UserID, ttl time.Duration) (*Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	session, err := NewSession(userID, ttl)
	if err != nil {
		return nil, errors.Wrapf(err, "create session for user %q", userID)
	}
	s.sessions[session.Token] = session
	return session, nil
}

// Find returns the session for the given token, or ErrSessionNotFound.
func (s *InMemorySessionStore) Find(token string) (*Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, ok := s.sessions[token]
	if !ok {
		return nil, ErrSessionNotFound
	}
	return session, nil
}

// Delete removes the session with the given token.
func (s *InMemorySessionStore) Delete(token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, token)
	return nil
}

// DeleteByUserID removes all sessions belonging to the given user.
func (s *InMemorySessionStore) DeleteByUserID(userID UserID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for token, session := range s.sessions {
		if session.UserID == userID {
			delete(s.sessions, token)
		}
	}
	return nil
}
