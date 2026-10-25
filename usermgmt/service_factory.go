package usermgmt

import (
	"fmt"
	"log/slog"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

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

	// Silent-data-loss-shaped misconfiguration: a store without Journal
	// methods gives the projection host an EMPTY journal — projections
	// would replay nothing and rebuilds would see no history. Dev/test
	// wrappers that embed event.Store land here intentionally; warn so
	// production consumers find out at startup, not after their first
	// rebuild.
	slog.Warn(
		"usermgmt: event store does not implement event.Journal — "+
			"falling back to an EMPTY in-memory journal; projections will not replay. "+
			"Use a journal-capable store (SQLite/Postgres/Pebble) in production.",
		"store_type", fmt.Sprintf("%T", store),
	)

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
		Authz:              config.Authz,
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

	// The setup built (and registered on the projection host) exactly one
	// casbin projection around the effective engine — config.Authz when
	// provided, the internal default otherwise. Event-derived policies reach
	// whichever engine is exposed here; there is no second, unregistered
	// projection to drift out of sync.
	authz := setup.casbinProjection.authz
	casbinProjection := setup.casbinProjection

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
	dispatcher.Use(commandAuditMiddleware()...)
	dispatcher.Use(config.CommandMiddleware...)
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
		authAudit:           authAuditor{logger: logger, store: setup.Store, bus: setup.Bus},
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
			svc.store, svc.bus,
		)
	}

	svc.wireLockoutEviction()

	svc.tokenPepper = config.TokenPepper
	svc.maxUsers = config.MaxUsers

	return svc, nil
}
