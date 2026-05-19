package usermgmt

import (
	"fmt"
	"sync"
	"time"
)

type UserStore interface {
	FindByID(id string) (*User, error)
	FindByEmail(email string) (*User, error)
	Save(user *User) error
	Delete(id string) error
}

type SessionStore interface {
	Create(userID string, ttl time.Duration) (*Session, error)
	Find(token string) (*Session, error)
	Delete(token string) error
	DeleteByUserID(userID string) error
}

type InMemoryUserStore struct {
	mu    sync.RWMutex
	users map[string]*User
}

func NewInMemoryUserStore() *InMemoryUserStore {
	return &InMemoryUserStore{users: make(map[string]*User)}
}

func (s *InMemoryUserStore) FindByID(id string) (*User, error) {
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
	for _, u := range s.users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, ErrUserNotFound
}

func (s *InMemoryUserStore) Save(user *User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	user.UpdatedAt = time.Now().UTC()
	s.users[user.ID] = user
	return nil
}

func (s *InMemoryUserStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
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

func (s *InMemorySessionStore) Create(userID string, ttl time.Duration) (*Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	session, err := NewSession(userID, ttl)
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
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

func (s *InMemorySessionStore) DeleteByUserID(userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for token, session := range s.sessions {
		if session.UserID == userID {
			delete(s.sessions, token)
		}
	}
	return nil
}
