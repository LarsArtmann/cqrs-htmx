package usermgmt

import (
	"database/sql"
	"log/slog"
	"sync"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/projection/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
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
	authAudit                authAuditor
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

	// CommandMiddleware is applied to the internal command dispatcher for
	// every dispatched usermgmt command (all 20 domain commands). It runs
	// INSIDE the built-in audit chain (enrichment, actor context, causation),
	// so consumer middleware sees fully enriched commands. Use the factories
	// from go-cqrs-lite/middleware/v4 (Recovery, Retry, CircuitBreaker,
	// TypedMetrics, Tracing, ...) — see docs/guides/leveraging-go-cqrs-lite.md
	// for a recommended production chain. Nil (the default) keeps the
	// dispatcher bare apart from the audit chain — backward compatible.
	CommandMiddleware []command.Middleware

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
	//
	// Limitation: the gate serializes check-then-register per PROCESS (via
	// registrationMu). Horizontally scaled replicas each hold their own
	// mutex, so N replicas can transiently admit up to N extra users before
	// their read models converge. Deployments that must never exceed
	// MaxUsers exactly should run a single writer replica (the common
	// single-user homelab case this gate targets).
	MaxUsers int
}

// wrapEventStore applies the optional StoreWrapper (e.g. transparent encryption)
// before the repository and projections are wired. A nil wrapper or nil result
// leaves the store unchanged.
