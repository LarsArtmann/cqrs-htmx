package usermgmt

import (
	"fmt"
	"sync"
	"time"
)

type UserStore interface {
	FindByID(id UserID) (*User, error)
	FindByEmail(email string) (*User, error)
	Save(user *User) error
	Create(user *User) error
	Delete(id UserID) error
}

type SessionStore interface {
	Create(userID UserID, ttl time.Duration) (*Session, error)
	Find(token string) (*Session, error)
	Delete(token string) error
	DeleteByUserID(userID UserID) error
}

type InMemoryUserStore struct {
	mu     sync.RWMutex
	users  map[UserID]*User
	emails map[string]UserID
}

func NewInMemoryUserStore() *InMemoryUserStore {
	return &InMemoryUserStore{
		users:  make(map[UserID]*User),
		emails: make(map[string]UserID),
	}
}

func (s *InMemoryUserStore) FindByID(id UserID) (*User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.users[id]
	if !ok {
		return nil, ErrUserNotFound
	}
	return u, nil
}

func (s *InMemoryUserStore) FindByEmail(email string) (*User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, ok := s.emails[email]
	if !ok {
		return nil, ErrUserNotFound
	}
	return s.users[id], nil
}

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

func (s *InMemoryUserStore) Create(user *User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.emails[user.Email]; ok {
		return ErrEmailExists
	}
	if _, ok := s.users[user.ID]; ok {
		return fmt.Errorf("user ID %s already exists", user.ID)
	}
	user.UpdatedAt = time.Now().UTC()
	s.users[user.ID] = user
	s.emails[user.Email] = user.ID
	return nil
}

func (s *InMemoryUserStore) Delete(id UserID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if u, ok := s.users[id]; ok {
		delete(s.emails, u.Email)
	}
	delete(s.users, id)
	return nil
}

type InMemorySessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*Session
	ttl      time.Duration
}

func NewInMemorySessionStore() *InMemorySessionStore {
	return &InMemorySessionStore{
		sessions: make(map[string]*Session),
		ttl:      defaultSessionTTL,
	}
}

func (s *InMemorySessionStore) WithTTL(ttl time.Duration) *InMemorySessionStore {
	s.ttl = ttl
	return s
}

func (s *InMemorySessionStore) Create(userID UserID, ttl time.Duration) (*Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	session, err := NewSession(userID, ttl)
	if err != nil {
		return nil, fmt.Errorf("create session for user %q: %w", userID, err)
	}
	s.sessions[session.Token] = session
	return session, nil
}

func (s *InMemorySessionStore) Find(token string) (*Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, ok := s.sessions[token]
	if !ok {
		return nil, ErrSessionNotFound
	}
	return session, nil
}

func (s *InMemorySessionStore) Delete(token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, token)
	return nil
}

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
