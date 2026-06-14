package usermgmt

import (
	"log/slog"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
)

const (
	defaultSessionTTL    = 24 * time.Hour
	maxDisplayNameLength = 100
)

// Service orchestrates user registration, authentication, authorization, and session management.
type Service struct {
	authz        *Authz
	users        UserStore
	sessions     SessionStore
	sessionTTL   time.Duration
	bcryptCost   int
	logger       *slog.Logger
	lockout      *AccountLockout
	eventHandler EventHandler
}

// ServiceConfig holds optional dependencies for NewService.
// Zero-valued fields are replaced with sensible defaults.
type ServiceConfig struct {
	// Authz is the authorization engine. Defaults to a new Authz with default policies.
	Authz *Authz
	// UserStore is the user persistence backend. Defaults to InMemoryUserStore.
	UserStore UserStore
	// SessionStore is the session persistence backend. Defaults to InMemorySessionStore.
	SessionStore SessionStore
	// SessionTTL is the default time-to-live for new sessions. Defaults to 24 hours.
	SessionTTL time.Duration
	// BcryptCost is the bcrypt hashing cost. Defaults to 12. Values below 4 are clamped.
	BcryptCost int
	// Logger is used for structured auth event logging. Defaults to slog.Default().
	Logger *slog.Logger
	// Lockout, if provided, enables account lockout after repeated login failures.
	Lockout *AccountLockout
	// EventHandler, if provided, is called after successful domain operations.
	// See events.go for the event types that may be passed.
	EventHandler EventHandler
}

// NewService creates a Service from the given config, applying defaults for nil/zero fields.
func NewService(cfg ServiceConfig) (*Service, error) {
	if cfg.UserStore == nil {
		cfg.UserStore = NewInMemoryUserStore()
	}
	if cfg.SessionStore == nil {
		cfg.SessionStore = NewInMemorySessionStore()
	}
	if cfg.SessionTTL == 0 {
		cfg.SessionTTL = defaultSessionTTL
	}
	if cfg.Authz == nil {
		a, err := NewAuthz()
		if err != nil {
			return nil, event.NewTransient("internal", "create authz").WithCause(err)
		}
		cfg.Authz = a
	}

	cost := cfg.BcryptCost
	if cost < minBcryptCost {
		cost = defaultBcryptCost
	}

	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	return &Service{
		authz:        cfg.Authz,
		users:        cfg.UserStore,
		sessions:     cfg.SessionStore,
		sessionTTL:   cfg.SessionTTL,
		bcryptCost:   cost,
		logger:       logger,
		lockout:      cfg.Lockout,
		eventHandler: cfg.EventHandler,
	}, nil
}

// Authz returns the underlying authorization engine for direct policy queries.
// Mutations to the returned Authz affect the Service's authorization behavior.
func (s *Service) Authz() *Authz { return s.authz }
