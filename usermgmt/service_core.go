package usermgmt

import (
	"context"
	"log/slog"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/larsartmann/go-cqrs-lite/command/v3"
	"github.com/larsartmann/go-cqrs-lite/decider/v3"
	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v3"
	"github.com/larsartmann/go-cqrs-lite/watermill/v3"
)

const (
	defaultSessionTTL    = 24 * time.Hour
	maxDisplayNameLength = 100
	maxEmailLength       = 254 // RFC 5321 max
)

// Service orchestrates user registration, authentication, authorization, and session management
// using event-sourced CQRS under the hood.
type Service struct {
	repository               *decider.Repository[UserState]
	membershipRepo           *decider.Repository[MembershipState]
	tenantRepo               *decider.Repository[TenantState]
	botRepo                  *decider.Repository[BotState]
	dispatcher               *command.Dispatcher
	readModel                *UserReadModel
	membershipReadModel      *MembershipReadModel
	tenantReadModel          *TenantReadModel
	botReadModel             *BotReadModel
	casbinProjection         *CasbinProjection
	authz                    *Authz
	sessions                 SessionStore
	sessionTTL               time.Duration
	logger                   *slog.Logger
	lockout                  *AccountLockout
	eventHandler             EventHandler
	bus                      event.Bus
	store                    event.Store
	webauthn                 *webauthn.WebAuthn
	webauthnSessions         *webauthnSessionStore
	stopWebAuthnEviction     func()
	auditLog                 *AuditLog
	verificationTokens       *verificationTokenStore
	stopVerificationEviction func()
	verificationTTL          time.Duration
	sendVerificationEmail    SendVerificationEmailFunc
	totpConfig               *TOTPConfig
	pendingTOTP              pendingTOTPStore
	stopPendingTOTPEviction  func()
	oauth2Providers          map[string]*oauth2Provider
	oauth2States             OAuth2StateStore
	stopOAuth2Eviction       func()
	oauth2StateTTL           time.Duration
	tokenPepper              TokenPepper
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
	// OAuth2Config, if provided, enables OAuth2/OIDC login with external
	// identity providers (Google, GitHub, etc.).
	OAuth2Config *OAuth2Config
	// OAuth2StateStore, if provided, replaces the default in-memory state
	// token store. Use this for multi-instance deployments (e.g., Redis).
	// Ignored when OAuth2Config is nil.
	OAuth2StateStore OAuth2StateStore

	// SecurityHooks configures opt-in event signing and encryption.
	// See SecurityHooks for field documentation.
	SecurityHooks

	// TokenPepper is the server-side secret used for HMAC-SHA256 bot token hashing.
	// Required for bot registration and API token authentication. Set this to a
	// 32+ byte random value that is stored outside the database (e.g., in a
	// secrets manager or environment variable). When nil, RegisterBot and
	// ResolveBotByToken return errors.
	TokenPepper TokenPepper
}

// wrapEventStore applies the optional StoreWrapper (e.g. transparent encryption)
// before the repository and projections are wired. A nil wrapper or nil result
// leaves the store unchanged.
func wrapEventStore(wrapper func(event.Store) (event.Store, error), store event.Store) (event.Store, error) {
	if wrapper == nil {
		return store, nil
	}

	wrapped, err := wrapper(store)
	if err != nil {
		return nil, event.NewTransient("internal", "wrap event store").WithCause(err)
	}

	if wrapped != nil {
		return wrapped, nil
	}

	return store, nil
}

// applyBusMiddleware applies PublishMiddleware and HandlerMiddleware to the bus
// before projections subscribe, so the middleware wraps all current and future
// handlers. PublishMiddleware (sign/encrypt on publish) runs outermost-first;
// HandlerMiddleware (verify/decrypt on handle) likewise.
func applyBusMiddleware(publishMW []event.PublishMiddleware, handlerMW []event.Middleware, bus event.Bus) error {
	if len(publishMW) > 0 {
		if err := bus.UsePublish(publishMW...); err != nil {
			return event.NewTransient("internal", "apply publish middleware").WithCause(err)
		}
	}

	if len(handlerMW) > 0 {
		if err := bus.Use(handlerMW...); err != nil {
			return event.NewTransient("internal", "apply handler middleware").WithCause(err)
		}
	}

	return nil
}

// journalFromStore extracts the projection-replay journal from the store.
// Falls back to a fresh in-memory store when the store implements neither
// *memory.MemoryStore nor event.Journal.
func journalFromStore(store event.Store) event.Journal {
	if memStore, ok := store.(*memory.MemoryStore); ok {
		return memStore
	}

	if j, ok := store.(event.Journal); ok {
		return j
	}

	return memory.NewMemoryStore()
}

// NewService creates a Service from the given config, applying defaults for nil/zero fields.
// It sets up event-sourced infrastructure (store, bus, repository, projections) with
// in-memory defaults if not provided.
//
//nolint:gocognit // inherent to multi-aggregate service wiring
func NewService(cfg ServiceConfig) (*Service, error) {
	store := cfg.EventStore
	if store == nil {
		store = memory.NewMemoryStore()
	}

	store, err := wrapEventStore(cfg.StoreWrapper, store)
	if err != nil {
		return nil, err
	}

	bus := cfg.EventBus
	if bus == nil {
		bus = watermill.NewEventBus()
	}

	if err := applyBusMiddleware(cfg.PublishMiddleware, cfg.HandlerMiddleware, bus); err != nil {
		return nil, err
	}

	journal := journalFromStore(store)

	repo, err := decider.NewRepository(store, bus, UserDecider())
	if err != nil {
		return nil, event.NewTransient("internal", "create decider repository").WithCause(err)
	}

	membershipRepo, err := decider.NewRepository(store, bus, MembershipDecider())
	if err != nil {
		return nil, event.NewTransient("internal", "create membership repository").WithCause(err)
	}

	tenantRepo, err := decider.NewRepository(store, bus, TenantDecider())
	if err != nil {
		return nil, event.NewTransient("internal", "create tenant repository").WithCause(err)
	}

	botRepo, err := decider.NewRepository(store, bus, BotDecider())
	if err != nil {
		return nil, event.NewTransient("internal", "create bot repository").WithCause(err)
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
	membershipReadModel := NewMembershipReadModel()
	tenantReadModel := NewTenantReadModel()
	botReadModel := NewBotReadModel()

	if err := StartProjections(
		journal,
		bus,
		readModel,
		membershipReadModel,
		tenantReadModel,
		botReadModel,
		casbinProjection,
		cfg.AuditLog,
	); err != nil {
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
	if err := RegisterMembershipCommands(dispatcher, membershipRepo); err != nil {
		return nil, event.NewTransient("internal", "register membership commands").WithCause(err)
	}
	if err := RegisterTenantCommands(dispatcher, tenantRepo); err != nil {
		return nil, event.NewTransient("internal", "register tenant commands").WithCause(err)
	}
	if err := RegisterBotCommands(dispatcher, botRepo); err != nil {
		return nil, event.NewTransient("internal", "register bot commands").WithCause(err)
	}

	//nolint:exhaustruct // fields set conditionally below
	svc := &Service{
		repository:          repo,
		membershipRepo:      membershipRepo,
		tenantRepo:          tenantRepo,
		botRepo:             botRepo,
		dispatcher:          dispatcher,
		readModel:           readModel,
		membershipReadModel: membershipReadModel,
		tenantReadModel:     tenantReadModel,
		botReadModel:        botReadModel,
		casbinProjection:    casbinProjection,
		authz:               authz,
		sessions:            cfg.SessionStore,
		sessionTTL:          cfg.SessionTTL,
		logger:              logger,
		lockout:             cfg.Lockout,
		eventHandler:        cfg.EventHandler,
		bus:                 bus,
		store:               store,
		auditLog:            cfg.AuditLog,
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
		svc.stopVerificationEviction = svc.verificationTokens.startEviction()
		svc.verificationTTL = cfg.EmailVerification.TokenTTL
		if svc.verificationTTL == 0 {
			svc.verificationTTL = VerificationTokenTTL
		}
		svc.sendVerificationEmail = cfg.EmailVerification.SendEmail
	}

	svc.totpConfig = cfg.TOTPConfig
	if cfg.TOTPConfig != nil {
		svc.pendingTOTP = newPendingTOTPStore()
		svc.stopPendingTOTPEviction = svc.pendingTOTP.startEviction()
	}

	if cfg.OAuth2Config != nil && len(cfg.OAuth2Config.Providers) > 0 {
		stateStore := cfg.OAuth2StateStore
		if stateStore == nil {
			stateStore = newOAuth2StateStore()
		}
		if err := svc.initOAuth2(cfg.OAuth2Config, stateStore); err != nil {
			return nil, err
		}
	}

	if cfg.EventHandler != nil {
		svc.bridgeEventHandler(bus)
	}

	svc.tokenPepper = cfg.TokenPepper

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
	if s.stopVerificationEviction != nil {
		s.stopVerificationEviction()
		s.stopVerificationEviction = nil
	}
	if s.stopPendingTOTPEviction != nil {
		s.stopPendingTOTPEviction()
		s.stopPendingTOTPEviction = nil
	}
	if s.stopOAuth2Eviction != nil {
		s.stopOAuth2Eviction()
		s.stopOAuth2Eviction = nil
	}
}

// initOAuth2 initializes the OAuth2 providers, state store, and background eviction.
func (s *Service) initOAuth2(cfg *OAuth2Config, stateStore OAuth2StateStore) error {
	s.oauth2Providers = make(map[string]*oauth2Provider, len(cfg.Providers))
	s.oauth2States = stateStore
	s.stopOAuth2Eviction = startPeriodicEviction(stateStore.EvictExpired, oauthStateEvictionInterval)
	s.oauth2StateTTL = cfg.StateTTL
	if s.oauth2StateTTL == 0 {
		s.oauth2StateTTL = defaultOAuthStateTTL
	}
	for name, provCfg := range cfg.Providers {
		prov, err := initOAuth2Provider(context.Background(), name, provCfg)
		if err != nil {
			return event.NewTransient("internal", "init oauth2 provider "+name).WithCause(err)
		}
		s.oauth2Providers[name] = prov
	}
	return nil
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
