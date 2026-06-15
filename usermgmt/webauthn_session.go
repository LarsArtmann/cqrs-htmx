package usermgmt

import (
	"sync"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
)

// WebAuthnConfig configures the WebAuthn Relying Party for passwordless auth.
type WebAuthnConfig struct {
	RPID          string   // e.g. "example.com"
	RPDisplayName string   // e.g. "My App"
	RPOrigins     []string // e.g. []string{"https://example.com"}
}

// webauthnSessionStore holds the temporary challenge data for in-flight WebAuthn ceremonies.
// Each entry expires after a short TTL (default 5 minutes).
type webauthnSessionStore struct {
	mu       sync.Mutex
	sessions map[string]*webauthn.SessionData
}

func newWebAuthnSessionStore() *webauthnSessionStore {
	return &webauthnSessionStore{
		sessions: make(map[string]*webauthn.SessionData),
	}
}

func (s *webauthnSessionStore) Save(key string, data *webauthn.SessionData) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[key] = data
}

func (s *webauthnSessionStore) Get(key string) (*webauthn.SessionData, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, ok := s.sessions[key]
	if !ok {
		return nil, ErrSessionDataNotFound
	}
	if !data.Expires.IsZero() && time.Now().After(data.Expires) {
		delete(s.sessions, key)
		return nil, ErrSessionDataNotFound
	}
	return data, nil
}

func (s *webauthnSessionStore) Delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, key)
}
