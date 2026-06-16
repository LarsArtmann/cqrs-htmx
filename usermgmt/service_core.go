package usermgmt

import (
	"context"
	"log/slog"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/larsartmann/go-cqrs-lite/command/v2"
	"github.com/larsartmann/go-cqrs-lite/decider/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/memory/v2"
)

const (
	defaultSessionTTL    = 24 * time.Hour
	maxDisplayNameLength = 100
)

// Service orchestrates user registration, authentication, authorization, and session management
// using event-sourced CQRS under the hood.
type Service struct {
	repository            *decider.Repository[UserState]
	dispatcher            *command.Dispatcher
	readModel             *UserReadModel
	casbinProjection      *CasbinProjection
	authz                 *Authz
	sessions              SessionStore
	sessionTTL            time.Duration
	logger                *slog.Logger
	lockout               *AccountLockout
	eventHandler          EventHandler
	bus                   event.Bus
	store                 event.Store
	webauthn              *webauthn.WebAuthn
	webauthnSessions      *webauthnSessionStore
	stopWebAuthnEviction  func()
	auditLog              *AuditLog
	verificationTokens    *verificationTokenStore
	verificationTTL       time.Duration
	sendVerificationEmail SendVerificationEmailFunc
	totpConfig            *TOTPConfig
	pendingTOTP           pendingTOTPStore
}

// ServiceConfig holds optional dependencies for NewService.
// Zero-valued fields are replaced with sensible defaults.
type ServiceConfig struct {
	// EventStore is the event persistence backend. Defaults to MemoryStore.
	EventStore event.Store
	// EventBus is the event pub/sub backend. Defaults to MemoryBus.
	EventBus event.Bus
	// Authz is the authorization engine. Defaults to a new Authz with default policies.
	Authz *Authz
	// SessionStore is the session persistence backend. Defaults to InMemorySessionStore.
	SessionStore SessionStore
	// SessionTTL is the default time-to-live for new sessions. Defaults to 24 hours.
	SessionTTL time.Duration
	// Logger is used for structured auth event logging. Defaults to slog.Default().
	Logger *slog.Logger
	// Lockout, if provided, enables account lockout after repeated login failures.
	Lockout *AccountLockout
	// EventHandler, if provided, is called after successful domain operations.
	EventHandler EventHandler
	// WebAuthnConfig configures passwordless authentication. Required for login.
	WebAuthnConfig *WebAuthnConfig
	// AuditLog, if provided, is registered as a projection to record all
	// user-related events for compliance and security monitoring.
	AuditLog *AuditLog
	// EmailVerification, if provided, enables the email verification flow.
	// When nil, SendVerificationEmail and VerifyEmail return ErrEmailVerificationNotConfigured.
	EmailVerification *EmailVerificationConfig
	// TOTPConfig, if provided, enables TOTP multi-factor authentication.
	TOTPConfig *TOTPConfig
}

// NewService creates a Service from the given config, applying defaults for nil/zero fields.
// It sets up event-sourced infrastructure (store, bus, repository, projections) with
// in-memory defaults if not provided.
func NewService(cfg ServiceConfig) (*Service, error) {
	store := cfg.EventStore
	if store == nil {
		store = memory.NewMemoryStore()
	}

	bus := cfg.EventBus
	if bus == nil {
		bus = memory.NewMemoryBus()
	}

	var journal event.Journal
	memStore, ok := store.(*memory.MemoryStore)
	if ok {
		journal = memStore
	} else {
		if j, jok := store.(event.Journal); jok {
			journal = j
		} else {
			journal = memory.NewMemoryStore()
		}
	}

	repo, err := decider.NewRepository(store, bus, UserDecider())
	if err != nil {
		return nil, event.NewTransient("internal", "create decider repository").WithCause(err)
	}

	authz := cfg.Authz
	if authz == nil {
		authz, err = NewAuthz()
		if err != nil {
			return nil, event.NewTransient("internal", "create authz").WithCause(err)
		}
	}

	casbinProjection, err := NewCasbinProjection(authz)
	if err != nil {
		return nil, event.NewTransient("internal", "create casbin projection").WithCause(err)
	}

	readModel := NewUserReadModel()

	if err := StartProjections(journal, bus, readModel, casbinProjection, cfg.AuditLog); err != nil {
		return nil, event.NewTransient("internal", "start projections").WithCause(err)
	}

	if cfg.SessionStore == nil {
		cfg.SessionStore = NewInMemorySessionStore()
	}
	if cfg.SessionTTL == 0 {
		cfg.SessionTTL = defaultSessionTTL
	}

	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	dispatcher := command.NewDispatcher()
	if err := RegisterCommands(dispatcher, repo); err != nil {
		return nil, event.NewTransient("internal", "register commands").WithCause(err)
	}

	//nolint:exhaustruct // fields set conditionally below
	svc := &Service{
		repository:       repo,
		dispatcher:       dispatcher,
		readModel:        readModel,
		casbinProjection: casbinProjection,
		authz:            authz,
		sessions:         cfg.SessionStore,
		sessionTTL:       cfg.SessionTTL,
		logger:           logger,
		lockout:          cfg.Lockout,
		eventHandler:     cfg.EventHandler,
		bus:              bus,
		store:            store,
		auditLog:         cfg.AuditLog,
	}

	if cfg.WebAuthnConfig != nil {
		//nolint:exhaustruct // only required fields set; others use go-webauthn defaults
		wa, err := webauthn.New(&webauthn.Config{
			RPID:          cfg.WebAuthnConfig.RPID,
			RPDisplayName: cfg.WebAuthnConfig.RPDisplayName,
			RPOrigins:     cfg.WebAuthnConfig.RPOrigins,
		})
		if err != nil {
			return nil, event.NewTransient("internal", "create webauthn instance").WithCause(err)
		}
		svc.webauthn = wa
		svc.webauthnSessions = newWebAuthnSessionStore()
		svc.stopWebAuthnEviction = svc.webauthnSessions.startEviction()
	}

	if cfg.EmailVerification != nil {
		svc.verificationTokens = newVerificationTokenStore()
		svc.verificationTTL = cfg.EmailVerification.TokenTTL
		if svc.verificationTTL == 0 {
			svc.verificationTTL = VerificationTokenTTL
		}
		svc.sendVerificationEmail = cfg.EmailVerification.SendEmail
	}

	svc.totpConfig = cfg.TOTPConfig
	if cfg.TOTPConfig != nil {
		svc.pendingTOTP = newPendingTOTPStore()
	}

	if cfg.EventHandler != nil {
		svc.bridgeEventHandler(bus)
	}

	return svc, nil
}

// Authz returns the underlying authorization engine for direct policy queries.
func (s *Service) Authz() *Authz { return s.authz }

// Stop gracefully shuts down background resources associated with the Service,
// such as the WebAuthn session eviction goroutine. It is safe to call multiple
// times and is a no-op when no background resources are running.
func (s *Service) Stop() {
	if s.stopWebAuthnEviction != nil {
		s.stopWebAuthnEviction()
		s.stopWebAuthnEviction = nil
	}
}

// ReadModel returns the user read model for direct queries.
func (s *Service) ReadModel() *UserReadModel { return s.readModel }

// AuditLog returns the configured audit log, or nil if not configured.
func (s *Service) AuditLog() *AuditLog { return s.auditLog }

// bridgeEventHandler subscribes to the bus and translates events to the old EventHandler callback.
func (s *Service) bridgeEventHandler(bus event.Subscriber) {
	if err := bus.Subscribe(eventUserRegistered, func(_ context.Context, evt event.Event) error {
		s.emit(userIDFromAggID(evt.AggregateID()), UserRegisteredEvent{
			Email:      s.emailFromEvent(evt),
			OccurredAt: evt.OccurredAt(),
		})
		return nil
	}); err != nil {
		s.logger.Warn("usermgmt: failed to subscribe to UserRegistered events", "error", err)
	}
	if err := bus.Subscribe(eventRolesUpdated, func(_ context.Context, evt event.Event) error {
		s.emit(userIDFromAggID(evt.AggregateID()), RolesUpdatedEvent{
			OccurredAt: evt.OccurredAt(),
		})
		return nil
	}); err != nil {
		s.logger.Warn("usermgmt: failed to subscribe to RolesUpdated events", "error", err)
	}
}

func userIDFromAggID(aggID interface{ String() string }) UserID {
	return NewUserID(aggID.String())
}

func (s *Service) emailFromEvent(evt event.Event) string {
	user, ok := s.readModel.FindByID(evt.AggregateID())
	if !ok {
		return ""
	}
	return user.Email
}
