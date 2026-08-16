package setup

import (
	"github.com/larsartmann/cqrs-htmx/adminui/v4"
	"github.com/larsartmann/cqrs-htmx/dashboardui/v4"
	"github.com/larsartmann/cqrs-htmx/loginpage/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
	memorystorage "github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
	"github.com/larsartmann/go-cqrs-lite/watermill/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// New creates a fully wired [Bundle] from a single [Config].
//
// It creates the shared event infrastructure (or uses provided stores), constructs the
// usermgmt.Service with all auth strategies, and builds the admin/dashboard/login UI panels.
// The returned Bundle is ready to [Bundle.Mount] and serve.
//
// All defaults are overridable via Config fields. Stores default to in-memory.
//
// When [Config.Service] is set, New adopts that service instead of building
// one — see the field documentation for the ownership contract.
func New(cfg Config) (*Bundle, error) {
	cfg = cfg.withDefaults()

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	var (
		svc         *usermgmt.Service
		ownsService bool
		store       event.Store
		bus         event.Bus
	)

	if cfg.Service != nil {
		svc = cfg.Service
		store = svc.Journal()
		bus = svc.EventBus()
	} else {
		ownsService = true

		store = cfg.EventStore
		if store == nil {
			store = memorystorage.NewMemoryStore()
		}

		bus = cfg.EventBus
		if bus == nil {
			//cqrs-lint:ignore(B024) go-cqrs-lite bus wraps handlers with recovery internally
			bus = watermill.NewEventBus()
		}

		built, err := usermgmt.NewService(usermgmt.ServiceConfig{
			EventStore:         store,
			EventBus:           bus,
			ReadModelDB:        cfg.ReadModelDB,
			AuditLog:           usermgmt.NewAuditLog(),
			TOTP:               cfg.TOTP,
			WebAuthn:           cfg.WebAuthn,
			OAuth2:             cfg.OAuth2,
			SessionTTL:         cfg.SessionTTL,
			Logger:             cfg.Logger,
			OnProjectionFailed: cfg.OnProjectionFailed,
			AsyncStartup:       cfg.AsyncStartup,
		})
		if err != nil {
			return nil, errorfamily.WrapRejection(err, "setup.service_creation_failed", "failed to create usermgmt service")
		}

		svc = built
	}

	bundle := &Bundle{ //nolint:exhaustruct // Admin/Dashboard/Login/SSE assigned conditionally below
		Service:     svc,
		Auth:        usermgmt.NewAuthHandler(svc),
		Stores:      &Stores{EventStore: store, EventBus: bus},
		ownsService: ownsService,
		config:      cfg,
	}

	if err := bundle.attachPanels(store, bus); err != nil {
		bundle.cleanup()

		return nil, err
	}

	bundle.attachSSE()

	return bundle, nil
}

// attachPanels builds and attaches the enabled UI panels (admin, dashboard,
// login) to the bundle. On failure the caller must call [Bundle.cleanup] to
// release everything created so far.
func (b *Bundle) attachPanels(store event.Store, bus event.Bus) error {
	if err := b.attachAdmin(); err != nil {
		return err
	}

	if err := b.attachDashboard(store, bus); err != nil {
		return err
	}

	return b.attachLogin()
}

// attachAdmin builds the admin panel unless disabled by Config.DisableAdmin.
func (b *Bundle) attachAdmin() error {
	if b.config.DisableAdmin {
		return nil
	}

	admin, err := adminui.New(adminui.Config{
		Service:     b.Service,
		Title:       b.config.Title,
		BasePath:    b.config.AdminPath,
		Mode:        b.config.AdminMode,
		TenantID:    b.config.TenantID,
		Authorizer:  b.config.AdminAuthorizer,
		AccentColor: b.config.AccentColor,
		LogoutURL:   b.config.LogoutURL,
		SSEURL:      b.config.SSEURL,
	})
	if err != nil {
		return errorfamily.WrapRejection(err, "setup.admin_creation_failed", "failed to create admin panel")
	}

	b.Admin = admin

	return nil
}

// attachDashboard builds the CQRS/ES observability dashboard unless disabled
// by Config.DisableDashboard, wired from the shared stores.
func (b *Bundle) attachDashboard(store event.Store, bus event.Bus) error {
	if b.config.DisableDashboard {
		return nil
	}

	dash, err := dashboardui.New(buildDashboardConfig(b.config, store, bus, b.Service.ProjectionHost()))
	if err != nil {
		return errorfamily.WrapRejection(
			err,
			"setup.dashboard_creation_failed",
			"failed to create CQRS dashboard",
		)
	}

	b.Dashboard = dash

	return nil
}

// attachLogin builds the login page unless disabled by Config.DisableLogin.
func (b *Bundle) attachLogin() error {
	if b.config.DisableLogin {
		return nil
	}

	login, err := loginpage.New(loginpage.Config{
		Service:        b.Service,
		Title:          b.config.Title,
		Redirect:       b.config.LoginRedirect,
		AccentColor:    b.config.AccentColor,
		NoRegistration: b.config.LoginNoRegistration,
	})
	if err != nil {
		return errorfamily.WrapRejection(err, "setup.login_creation_failed", "failed to create login page")
	}

	b.Login = login

	return nil
}

// cleanup closes all resources created by [New], in reverse creation order.
// Called on any creation failure to prevent leaks; [Bundle.Close] handles the
// success path. An adopted service is never closed here — its lifecycle
// belongs to the caller who provided it.
func (b *Bundle) cleanup() {
	if b.Dashboard != nil {
		//cqrs-lint:ignore(C015) Dashboard.Close has no error return — nothing to check
		b.Dashboard.Close()
	}

	if b.ownsService && b.Service != nil {
		//cqrs-lint:ignore(C023,C015) cleanup on a failed construction path: the creation error is the primary failure; Close errors here are secondary
		_ = b.Service.Close()
	}
}

// MustNew is like [New] but panics on error. Use for tests and demos where failure is a bug.
func MustNew(cfg Config) *Bundle {
	b, err := New(cfg)
	if err != nil {
		panic(err)
	}

	return b
}

// buildDashboardConfig constructs the dashboardui.Config from the setup Config,
// wiring shared stores and applying dashboard-specific defaults.
func buildDashboardConfig(
	cfg Config,
	store event.Store,
	bus event.Bus,
	projectionHost *projectionhost.Host,
) dashboardui.Config {
	dashCfg := dashboardui.Config{
		Title:          cfg.Title + " · CQRS Dashboard",
		BasePath:       cfg.DashboardPath,
		EventSource:    store,
		EventBus:       bus,
		ProjectionHost: projectionHost,
		PageSize:       cfg.DashboardPageSize,
		LogoutURL:      cfg.LogoutURL,
		AccentColor:    cfg.AccentColor,
		Authorizer:     cfg.DashboardAuthorizer,
	}

	if journal, ok := store.(event.Journal); ok {
		dashCfg.Journal = journal
	}

	if cfg.DashboardReadOnly != nil {
		dashCfg.ReadOnly = *cfg.DashboardReadOnly
	} else {
		dashCfg.ReadOnly = true
	}

	return dashCfg
}
