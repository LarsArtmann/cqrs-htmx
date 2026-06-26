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

// webauthnEvictionInterval is how often the background cleanup goroutine
// scans for expired WebAuthn challenge sessions.
const webauthnEvictionInterval = 5 * time.Minute

// webauthnSessionStore holds the temporary challenge data for in-flight WebAuthn ceremonies.
// Each entry expires after the WebAuthn session's Expires field (set by go-webauthn).
// Expired entries are removed lazily on Get and proactively by a background goroutine.
type webauthnSessionStore struct {
	mu       sync.Mutex
	sessions map[string]*webauthn.SessionData
}

func newWebAuthnSessionStore() *webauthnSessionStore {
	//nolint:exhaustruct // mu zero-value is correct (sync.Mutex)
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
	deleteKeyed(&s.mu, s.sessions, key)
}

// EvictExpired removes all sessions whose Expires time has passed.
// Returns the number of sessions evicted.
func (s *webauthnSessionStore) EvictExpired() int {
	return evictExpired(&s.mu, s.sessions, func(_ string, data *webauthn.SessionData) bool {
		return !data.Expires.IsZero() && time.Now().After(data.Expires)
	})
}

// startEviction launches a background goroutine that periodically removes
// expired WebAuthn challenge sessions. Returns a stop function that must be
// called to terminate the goroutine (e.g. on shutdown or in tests).
func (s *webauthnSessionStore) startEviction() (stop func()) {
	return startPeriodicEviction(s.EvictExpired, webauthnEvictionInterval)
}
