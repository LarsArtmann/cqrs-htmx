package usermgmt

import (
	"context"
	"database/sql"
	"log/slog"
	"sync"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/projection/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

const (
	defaultSessionTTL    = 24 * time.Hour
	maxDisplayNameLength = 100
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
	webauthn                 WebAuthnProvider
	webauthnSessions         WebAuthnSessionStore
	stopWebAuthnEviction     func()
	auditLog                 *AuditLog
	verificationTokens       VerificationTokenStore
	stopVerificationEviction func()
	verificationTTL          time.Duration
	sendVerificationEmail    SendVerificationEmailFunc
	totp                     TOTPProvider
	pendingTOTP              PendingTOTPStore
	totpPendingTTL           time.Duration
	stopPendingTOTPEviction  func()
	oauth2                   OAuth2Provider
	oauth2Svc                *OAuth2Service
	oauth2States             OAuth2StateStore
	stopOAuth2Eviction       func()
	oauth2StateTTL           time.Duration
	stopLockoutEviction      func()
	tokenPepper              TokenPepper
	projectionHost           *projectionhost.Host
	checkpointStore          event.CheckpointStore
	projections              []projection.Projection
	maxUsers                 int
	// registrationMu serializes the check-then-register critical section in
	// Register and OAuth2Service.matchOrCreateUser so concurrent requests cannot
	// both pass the MaxUsers check before either dispatch lands in the read model.
	registrationMu sync.Mutex
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
	// WebAuthn, if provided, enables passwordless passkey authentication.
	// Import usermgmt/webauthn to obtain a *Provider:
	//
	// 	wa, _ := webauthn.New(webauthn.Config{RPID: "localhost", RPDisplayName: "MyApp"})
	// 	svc, _ := usermgmt.NewService(usermgmt.ServiceConfig{WebAuthn: wa})
	WebAuthn WebAuthnProvider
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
	// WebAuthnSessionTTL is the time-to-live for in-flight WebAuthn challenge
	// sessions. Defaults to 5 minutes. Ignored when WebAuthn is nil or when
	// WebAuthnSessionStore is provided (the custom store manages its own TTL).
	WebAuthnSessionTTL time.Duration
	// VerificationTokenStore, if provided, replaces the default in-memory email
	// verification token store. Use this for multi-instance deployments.
	// Ignored when EmailVerification is nil.
	VerificationTokenStore VerificationTokenStore
	// PendingTOTPStore, if provided, replaces the default in-memory pending TOTP
	// secret store. Use this for multi-instance deployments.
	// Ignored when TOTP is nil.
	PendingTOTPStore PendingTOTPStore
	// TOTPPendingSecretTTL is the time-to-live for in-flight TOTP setup
	// secrets (between EnableTOTP and VerifyTOTPSetup). Defaults to 5 minutes.
	// Ignored when TOTP is nil or when PendingTOTPStore is provided.
	TOTPPendingSecretTTL time.Duration

	// OAuth2, if provided, enables OAuth2/OIDC login with external identity
	// providers (Google, GitHub, etc.). Import usermgmt/oauth2 to obtain a
	// *Provider.
	OAuth2 OAuth2Provider
	// OAuth2StateStore, if provided, replaces the default in-memory state
	// token store. Use this for multi-instance deployments (e.g., Redis).
	// Ignored when OAuth2Config is nil.
	OAuth2StateStore OAuth2StateStore
	// OAuth2StateTTL is the time-to-live for OAuth2 state tokens used in the
	// PKCE flow. Defaults to 10 minutes. Ignored when OAuth2 is nil or when
	// OAuth2StateStore is provided (the custom store manages its own TTL).
	OAuth2StateTTL time.Duration

	// ReadModelDB, when set, creates SQL-backed read models (User, Membership,
	// Tenant, Bot) that persist across restarts. Use [OptimizeSQLiteDB] to tune
	// the connection before passing it here. When nil, in-memory read models
	// are used (data lost on restart).
	ReadModelDB *sql.DB

	// ReadModelDialect selects the constructor family used for ReadModelDB:
	// "" / "sqlite" / "sqlite3" (default, historical behavior), "postgres" /
	// "pgx", or "mysql". Ignored when ReadModelDB is nil.
	ReadModelDialect string

	// SecurityHooks configures opt-in event signing and encryption.
	// See SecurityHooks for field documentation.
	SecurityHooks

	// CheckpointStore, when set, enables checkpoint-based projection replay:
	// on restart, only events since the last checkpoint are replayed instead
	// of the full journal. Must be backed by persistent storage to survive
	// restarts. When nil, full journal replay is used.
	CheckpointStore event.CheckpointStore

	// OnProjectionFailed, when set, is called when a projection worker
	// exhausts its restart budget and enters a terminal failure state.
	// Use this for alerting (e.g., emit a metric, page on-call). The
	// callback receives the projection name and the last error message.
	// Optional — when nil, terminal failures are silent (logs only).
	OnProjectionFailed func(projectionName, lastError string)

	// SnapshotConfig optionally enables aggregate snapshotting for
	// high-event-volume aggregates (>10K events/aggregate). When the Store
	// field is nil (the default), repositories replay the full journal on
	// every Load — zero behavior change. See SnapshotConfig for usage.
	SnapshotConfig

	// TokenPepper is the server-side secret used for HMAC-SHA256 bot token hashing.
	// Required for bot registration and API token authentication. Set this to a
	// 32+ byte random value that is stored outside the database (e.g., in a
	// secrets manager or environment variable). When nil, RegisterBot and
	// ResolveBotByToken return errors.
	TokenPepper TokenPepper

	// DrainTimeout is the maximum time to wait for projection workers to finish
	// their initial journal drain during startup. Defaults to 30 seconds when
	// zero. Increase this for deployments with large event journals or slow
	// storage (e.g., SQLite on contended disk). A transient error is returned
	// when the drain cannot complete within this budget. Ignored when
	// AsyncStartup is true.
	DrainTimeout time.Duration

	// AsyncStartup controls whether NewService blocks until projection workers
	// finish their initial journal drain. When false (the default), NewService
	// blocks until all projections catch up — preserving the historical
	// synchronous startup (the HTTP server cannot bind until drain completes).
	//
	// Set to true for async startup: NewService returns immediately after the
	// projection host starts, so the HTTP server binds while projections replay
	// the journal in the background. This eliminates multi-minute restart
	// outages on deployments with large event journals. Gate reads behind a
	// readiness check (cqrshtmx.ProjectionReadinessCheck) — point your reverse
	// proxy's health check at /health, which returns 503 until every projection
	// reaches "live" state, then 200.
	//
	// Recommended for production deployments. With async startup, projection
	// failures during drain surface via ProjectionStatuses(), the /health
	// endpoint, or OnProjectionFailed — not as a NewService error.
	AsyncStartup bool

	// MaxUsers, when greater than zero, limits the total number of users that
	// can register. When the user count reaches MaxUsers, all further
	// registration attempts return ErrRegistrationClosed — including the
	// implicit registration of new users via OAuth2 first-login
	// auto-provisioning. Existing users can always log in. Zero (the default)
	// means unlimited registration. Set to 1 for single-user deployments to
	// automatically lock registration after the first user is created.
	MaxUsers int
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
		return nil, errorfamily.NewTransient("usermgmt.event_store.wrap", "wrap event store").WithCause(err)
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
			return errorfamily.NewTransient("usermgmt.middleware.apply_publish", "apply publish middleware").
				WithCause(err)
		}
	}

	if len(handlerMW) > 0 {
		if err := bus.Use(handlerMW...); err != nil {
			return errorfamily.NewTransient("usermgmt.middleware.apply_handler", "apply handler middleware").
				WithCause(err)
		}
	}

	return nil
}

// journalFromStore extracts the projection-replay journal from the store.
// Falls back to a fresh in-memory store when the store does not implement
// event.Journal — this is intentional for dev/test wrappers that embed
// event.Store without promoting Journal methods. Production consumers
// should use a journal-capable store (SQLite/Postgres/Pebble).
func journalFromStore(store event.Store) event.Journal {
	if j, ok := store.(event.Journal); ok {
		return j
	}

	//cqrs-lint:ignore(C018) dev/test fallback: consumers using journal-capable stores (SQLite/Postgres) are unaffected; wrapped stores in tests need the fallback
	return memory.NewMemoryStore()
}

// NewService creates a Service from the given config, applying defaults for nil/zero fields.
// It sets up event-sourced infrastructure (store, bus, repository, projections) with
// in-memory defaults if not provided.
//

func NewService(config ServiceConfig) (*Service, error) {
	setup, err := NewEventSourcedSetup(EventSourcedConfig{
		EventStore:         config.EventStore,
		EventBus:           config.EventBus,
		ReadModelDB:        config.ReadModelDB,
		ReadModelDialect:   config.ReadModelDialect,
		AuditLog:           config.AuditLog,
		CheckpointStore:    config.CheckpointStore,
		OnProjectionFailed: config.OnProjectionFailed,
		SecurityHooks:      config.SecurityHooks,
		SnapshotConfig:     config.SnapshotConfig,
		DrainTimeout:       config.DrainTimeout,
		AsyncStartup:       config.AsyncStartup,
	})
	if err != nil {
		return nil, err
	}

	// Use custom Authz if provided (with a fresh projection); otherwise use setup's.
	authz := setup.casbinProjection.authz
	casbinProjection := setup.casbinProjection
	if config.Authz != nil {
		authz = config.Authz
		casbinProjection, err = NewCasbinProjection(authz)
		if err != nil {
			return nil, errorfamily.NewTransient("usermgmt.authz.create_casbin_projection", "create casbin projection").
				WithCause(err)
		}
	}

	if config.SessionStore == nil {
		config.SessionStore = NewInMemorySessionStore()
	}
	if config.SessionTTL == 0 {
		config.SessionTTL = defaultSessionTTL
	}

	logger := config.Logger
	if logger == nil {
		logger = slog.Default()
	}

	dispatcher := command.NewDispatcher()
	if err := RegisterCommands(dispatcher, setup.Repository); err != nil {
		return nil, errorfamily.NewTransient("usermgmt.command.register", "register commands").WithCause(err)
	}
	if err := RegisterMembershipCommands(dispatcher, setup.MembershipRepository); err != nil {
		return nil, errorfamily.NewTransient("usermgmt.command.register_membership", "register membership commands").
			WithCause(err)
	}
	if err := RegisterTenantCommands(dispatcher, setup.TenantRepository); err != nil {
		return nil, errorfamily.NewTransient("usermgmt.command.register_tenant", "register tenant commands").
			WithCause(err)
	}
	if err := RegisterBotCommands(dispatcher, setup.BotRepository); err != nil {
		return nil, errorfamily.NewTransient("usermgmt.command.register_bot", "register bot commands").WithCause(err)
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
		sessions:            config.SessionStore,
		sessionTTL:          config.SessionTTL,
		logger:              logger,
		lockout:             config.Lockout,
		bus:                 setup.Bus,
		store:               setup.Store,
		auditLog:            config.AuditLog,
		projectionHost:      setup.projectionHost,
		checkpointStore:     setup.checkpointStore,
		projections:         setup.projections,
	}

	if config.WebAuthn != nil {
		svc.webauthn = config.WebAuthn
		sessionStore := config.WebAuthnSessionStore
		if sessionStore == nil {
			mem := newWebAuthnSessionStore(config.WebAuthnSessionTTL)
			sessionStore = mem
			svc.stopWebAuthnEviction = mem.startEviction()
		}
		svc.webauthnSessions = sessionStore
	}

	if config.EmailVerification != nil {
		vStore := config.VerificationTokenStore
		if vStore == nil {
			mem := newVerificationTokenStore()
			vStore = mem
			svc.stopVerificationEviction = mem.startEviction()
		}
		svc.verificationTokens = vStore
		svc.verificationTTL = config.EmailVerification.TokenTTL
		if svc.verificationTTL == 0 {
			svc.verificationTTL = VerificationTokenTTL
		}
		svc.sendVerificationEmail = config.EmailVerification.SendEmail
	}

	svc.totp = config.TOTP
	if config.TOTP != nil {
		tStore := config.PendingTOTPStore
		if tStore == nil {
			mem := newPendingTOTPStore()
			tStore = &mem
			svc.stopPendingTOTPEviction = mem.startEviction()
		}
		svc.pendingTOTP = tStore
		svc.totpPendingTTL = config.TOTPPendingSecretTTL
		if svc.totpPendingTTL == 0 {
			svc.totpPendingTTL = defaultTOTPPendingTTL
		}
	}

	if config.OAuth2 != nil {
		stateStore := config.OAuth2StateStore
		if stateStore == nil {
			stateStore = newOAuth2StateStore()
		}
		svc.oauth2 = config.OAuth2
		svc.oauth2States = stateStore
		svc.oauth2StateTTL = config.OAuth2StateTTL
		if svc.oauth2StateTTL == 0 {
			svc.oauth2StateTTL = defaultOAuthStateTTL
		}
		svc.stopOAuth2Eviction = startPeriodicEviction(stateStore.EvictExpired, oauthStateEvictionInterval)
		svc.oauth2Svc = NewOAuth2Service(
			config.OAuth2, stateStore, svc.oauth2StateTTL,
			config.MaxUsers, &svc.registrationMu,
			svc.readModel, svc.dispatcher, svc.sessions, svc.sessionTTL, svc.logger,
			svc.classifyDispatchError,
		)
	}

	svc.wireLockoutEviction()

	svc.tokenPepper = config.TokenPepper
	svc.maxUsers = config.MaxUsers

	return svc, nil
}

func (s *Service) wireLockoutEviction() {
	if s.lockout == nil {
		return
	}
	evictor, ok := s.lockout.(interface{ EvictStale() int })
	if !ok {
		return
	}
	s.stopLockoutEviction = startPeriodicEviction(evictor.EvictStale, lockoutEvictionInterval)
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
		&s.stopLockoutEviction,
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
		return errorfamily.WrapTransient(
			err,
			"usermgmt.service.graceful_close",
			"context cancelled during graceful close",
		)
	}
	return nil
}

// closeInfra closes the bus and store if they implement io.Closer.
// Returns the first error encountered.
func (s *Service) closeInfra() error {
	if s.projectionHost != nil {
		if err := s.projectionHost.Stop(); err != nil {
			slog.Warn(
				"usermgmt.Service: failed to stop projection host during close",
				slog.String("error", err.Error()),
			)
			_ = errorfamily.WrapTransient(err, "usermgmt.service.stop_projections", "stop projection host")
		}
	}
	if c, ok := s.bus.(interface{ Close() error }); ok {
		if err := c.Close(); err != nil {
			return errorfamily.WrapTransient(err, "usermgmt.service.close_bus", "close event bus")
		}
	}
	if c, ok := s.store.(interface{ Close() error }); ok {
		if err := c.Close(); err != nil {
			return errorfamily.WrapTransient(err, "usermgmt.service.close_store", "close event store")
		}
	}
	return nil
}

// ReadModel returns the user read model for direct queries.
func (s *Service) ReadModel() *UserReadModel { return s.readModel }

// AuditLog returns the configured audit log, or nil if not configured.
func (s *Service) AuditLog() *AuditLog { return s.auditLog }

// ProjectionHost returns the internal projection host that manages read-model
// and Casbin projections. Returns nil if the service was constructed without
// event-sourced setup.
//
// Consumers building observability dashboards (e.g. dashboardui) can use this
// to wire projection health panels without constructing a separate host.
func (s *Service) ProjectionHost() *projectionhost.Host { return s.projectionHost }

func (s *Service) emailFromEvent(evt event.Event) string {
	user, ok := s.readModel.FindByID(evt.StreamID())
	if !ok {
		return ""
	}
	return user.Email
}
