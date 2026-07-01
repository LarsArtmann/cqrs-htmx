package usermgmt

import (
	"context"
	"database/sql"
	"log/slog"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/larsartmann/go-cqrs-lite/command/v3"
	"github.com/larsartmann/go-cqrs-lite/decider/v3"
	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v3"
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
	lockout                  LockoutStore
	bus                      event.Bus
	store                    event.Store
	webauthn                 *webauthn.WebAuthn
	webauthnSessions         WebAuthnSessionStore
	stopWebAuthnEviction     func()
	auditLog                 *AuditLog
	verificationTokens       VerificationTokenStore
	stopVerificationEviction func()
	verificationTTL          time.Duration
	sendVerificationEmail    SendVerificationEmailFunc
	totp                     TOTPProvider
	pendingTOTP              PendingTOTPStore
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
	// EventStore is the event persistence backend. Defaults to storage/memory.MemoryStore.
	EventStore event.Store
	// EventBus is the event pub/sub backend. Defaults to watermill.EventBus.
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
	// Defaults to none. Implement [LockoutStore] for distributed lockout.
	Lockout LockoutStore
	// WebAuthnConfig configures passwordless authentication. Required for login.
	WebAuthnConfig *WebAuthnConfig
	// AuditLog, if provided, is registered as a projection to record all
	// user-related events for compliance and security monitoring.
	AuditLog *AuditLog
	// EmailVerification, if provided, enables the email verification flow.
	// When nil, SendVerificationEmail and VerifyEmail return ErrEmailVerificationNotConfigured.
	EmailVerification *EmailVerificationConfig
	// TOTP, if provided, enables TOTP multi-factor authentication.
	// Import usermgmt/totp to obtain a *Provider:
	//
	// 	svc, _ := usermgmt.NewService(usermgmt.ServiceConfig{
	// 		TOTP: totp.New(totp.Config{Issuer: "MyApp"}),
	// 	})
	TOTP TOTPProvider
	// WebAuthnSessionStore, if provided, replaces the default in-memory WebAuthn
	// challenge store. Use this for multi-instance deployments (e.g., Redis).
	// Ignored when WebAuthnConfig is nil.
	WebAuthnSessionStore WebAuthnSessionStore
	// VerificationTokenStore, if provided, replaces the default in-memory email
	// verification token store. Use this for multi-instance deployments.
	// Ignored when EmailVerification is nil.
	VerificationTokenStore VerificationTokenStore
	// PendingTOTPStore, if provided, replaces the default in-memory pending TOTP
	// secret store. Use this for multi-instance deployments.
	// Ignored when TOTP is nil.
	PendingTOTPStore PendingTOTPStore

	// OAuth2Config, if provided, enables OAuth2/OIDC login with external
	// identity providers (Google, GitHub, etc.).
	OAuth2Config *OAuth2Config
	// OAuth2StateStore, if provided, replaces the default in-memory state
	// token store. Use this for multi-instance deployments (e.g., Redis).
	// Ignored when OAuth2Config is nil.
	OAuth2StateStore OAuth2StateStore

	// ReadModelDB, when set, creates SQL-backed read models (User, Membership,
	// Tenant, Bot) that persist across restarts. Use [OptimizeSQLiteDB] to tune
	// the connection before passing it here. When nil, in-memory read models
	// are used (data lost on restart).
	ReadModelDB *sql.DB

	// SecurityHooks configures opt-in event signing and encryption.
	// See SecurityHooks for field documentation.
	SecurityHooks

	// CheckpointStore, when set, enables checkpoint-based projection replay:
	// on restart, only events since the last checkpoint are replayed instead
	// of the full journal. Must be backed by persistent storage to survive
	// restarts. When nil, full journal replay is used.
	CheckpointStore event.CheckpointStore

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
// *storage/memory.MemoryStore nor event.Journal.
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

func NewService(cfg ServiceConfig) (*Service, error) {
	setup, err := NewEventSourcedSetup(EventSourcedConfig{
		EventStore:      cfg.EventStore,
		EventBus:        cfg.EventBus,
		ReadModelDB:     cfg.ReadModelDB,
		AuditLog:        cfg.AuditLog,
		CheckpointStore: cfg.CheckpointStore,
		SecurityHooks:   cfg.SecurityHooks,
	})
	if err != nil {
		return nil, err
	}

	// Use custom Authz if provided (with a fresh projection); otherwise use setup's.
	authz := setup.casbinProjection.authz
	casbinProjection := setup.casbinProjection
	if cfg.Authz != nil {
		authz = cfg.Authz
		casbinProjection, err = NewCasbinProjection(authz)
		if err != nil {
			return nil, event.NewTransient("internal", "create casbin projection").WithCause(err)
		}
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
	if err := RegisterCommands(dispatcher, setup.Repository); err != nil {
		return nil, event.NewTransient("internal", "register commands").WithCause(err)
	}
	if err := RegisterMembershipCommands(dispatcher, setup.MembershipRepository); err != nil {
		return nil, event.NewTransient("internal", "register membership commands").WithCause(err)
	}
	if err := RegisterTenantCommands(dispatcher, setup.TenantRepository); err != nil {
		return nil, event.NewTransient("internal", "register tenant commands").WithCause(err)
	}
	if err := RegisterBotCommands(dispatcher, setup.BotRepository); err != nil {
		return nil, event.NewTransient("internal", "register bot commands").WithCause(err)
	}

	//nolint:exhaustruct // fields set conditionally below
	svc := &Service{
		repository:          setup.Repository,
		membershipRepo:      setup.MembershipRepository,
		tenantRepo:          setup.TenantRepository,
		botRepo:             setup.BotRepository,
		dispatcher:          dispatcher,
		readModel:           setup.ReadModel,
		membershipReadModel: setup.MembershipReadModel,
		tenantReadModel:     setup.TenantReadModel,
		botReadModel:        setup.BotReadModel,
		casbinProjection:    casbinProjection,
		authz:               authz,
		sessions:            cfg.SessionStore,
		sessionTTL:          cfg.SessionTTL,
		logger:              logger,
		lockout:             cfg.Lockout,
		bus:                 setup.Bus,
		store:               setup.Store,
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
		sessionStore := cfg.WebAuthnSessionStore
		if sessionStore == nil {
			mem := newWebAuthnSessionStore()
			sessionStore = mem
			svc.stopWebAuthnEviction = mem.startEviction()
		}
		svc.webauthnSessions = sessionStore
	}

	if cfg.EmailVerification != nil {
		vStore := cfg.VerificationTokenStore
		if vStore == nil {
			mem := newVerificationTokenStore()
			vStore = mem
			svc.stopVerificationEviction = mem.startEviction()
		}
		svc.verificationTokens = vStore
		svc.verificationTTL = cfg.EmailVerification.TokenTTL
		if svc.verificationTTL == 0 {
			svc.verificationTTL = VerificationTokenTTL
		}
		svc.sendVerificationEmail = cfg.EmailVerification.SendEmail
	}

	svc.totp = cfg.TOTP
	if cfg.TOTP != nil {
		tStore := cfg.PendingTOTPStore
		if tStore == nil {
			mem := newPendingTOTPStore()
			tStore = &mem
			svc.stopPendingTOTPEviction = mem.startEviction()
		}
		svc.pendingTOTP = tStore
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

	svc.tokenPepper = cfg.TokenPepper

	return svc, nil
}

// Authz returns the underlying authorization engine for direct policy queries.
func (s *Service) Authz() *Authz { return s.authz }

// Stop gracefully shuts down background resources associated with the Service,
// such as the WebAuthn session eviction goroutine. It is safe to call multiple
// times and is a no-op when no background resources are running.
//
// Stop does NOT close the event bus or event store. Use [Service.Close] for
// full lifecycle shutdown.
func (s *Service) Stop() {
	s.stopEvictions()
}

// stopEvictions stops all background eviction goroutines. Idempotent.
func (s *Service) stopEvictions() {
	for _, stop := range []*func(){
		&s.stopWebAuthnEviction,
		&s.stopVerificationEviction,
		&s.stopPendingTOTPEviction,
		&s.stopOAuth2Eviction,
	} {
		if *stop != nil {
			(*stop)()
			*stop = nil
		}
	}
}

// Close performs a full lifecycle shutdown: stops all background eviction
// goroutines, then closes the event bus and event store (if they implement
// io.Closer). It is safe to call multiple times.
//
// For context-bounded shutdown prefer [Service.GracefulClose].
func (s *Service) Close() error {
	s.stopEvictions()
	return s.closeInfra()
}

// GracefulClose is identical to [Service.Close] but bounded by ctx.
// If the context expires before resources are closed, the context error is
// returned alongside any resource errors.
func (s *Service) GracefulClose(ctx context.Context) error {
	s.stopEvictions()
	if err := s.closeInfra(); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return event.WrapTransient(err, "usermgmt.service.graceful_close", "context cancelled during graceful close")
	}
	return nil
}

// closeInfra closes the bus and store if they implement io.Closer.
// Returns the first error encountered.
func (s *Service) closeInfra() error {
	if c, ok := s.bus.(interface{ Close() error }); ok {
		if err := c.Close(); err != nil {
			return event.WrapTransient(err, "usermgmt.service.close_bus", "close event bus")
		}
	}
	if c, ok := s.store.(interface{ Close() error }); ok {
		if err := c.Close(); err != nil {
			return event.WrapTransient(err, "usermgmt.service.close_store", "close event store")
		}
	}
	return nil
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

func (s *Service) emailFromEvent(evt event.Event) string {
	user, ok := s.readModel.FindByID(evt.AggregateID())
	if !ok {
		return ""
	}
	return user.Email
}
