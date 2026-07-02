package usermgmt

import (
	"sync"
	"time"
)

// webauthnSessionTTL is the default lifetime of a WebAuthn challenge session.
// WebAuthn ceremonies are short-lived; 5 minutes matches typical go-webauthn defaults.
const webauthnSessionTTL = 5 * time.Minute

// webauthnEvictionInterval is how often the background cleanup goroutine
// scans for expired WebAuthn challenge sessions.
const webauthnEvictionInterval = 5 * time.Minute

// webauthnSessionEntry holds the opaque session data and its expiry time.
type webauthnSessionEntry struct {
	data      []byte
	expiresAt time.Time
}

// webauthnSessionStore holds the temporary challenge data for in-flight WebAuthn ceremonies.
// Each entry expires after webauthnSessionTTL.
// Expired entries are removed lazily on Get and proactively by a background goroutine.
type webauthnSessionStore struct {
	mu       sync.Mutex
	sessions map[string]webauthnSessionEntry
	ttl      time.Duration
}

func newWebAuthnSessionStore() *webauthnSessionStore {
	return &webauthnSessionStore{ //nolint:exhaustruct // mu zero-value is correct (sync.Mutex)
		sessions: make(map[string]webauthnSessionEntry),
		ttl:      webauthnSessionTTL,
	}
}

func (s *webauthnSessionStore) Save(key string, data []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[key] = webauthnSessionEntry{
		data:      data,
		expiresAt: time.Now().Add(s.ttl),
	}
}

func (s *webauthnSessionStore) Get(key string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.sessions[key]
	if !ok {
		return nil, ErrSessionDataNotFound
	}
	if time.Now().After(entry.expiresAt) {
		delete(s.sessions, key)
		return nil, ErrSessionDataNotFound
	}
	return entry.data, nil
}

func (s *webauthnSessionStore) Delete(key string) {
	deleteKeyed(&s.mu, s.sessions, key)
}

// EvictExpired removes all sessions whose TTL has expired.
// Returns the number of sessions evicted.
func (s *webauthnSessionStore) EvictExpired() int {
	return evictExpired(&s.mu, s.sessions, func(_ string, entry webauthnSessionEntry) bool {
		return time.Now().After(entry.expiresAt)
	})
}

// startEviction launches a background goroutine that periodically removes
// expired WebAuthn challenge sessions. Returns a stop function that must be
// called to terminate the goroutine (e.g. on shutdown or in tests).
func (s *webauthnSessionStore) startEviction() (stop func()) {
	return startPeriodicEviction(s.EvictExpired, webauthnEvictionInterval)
}
