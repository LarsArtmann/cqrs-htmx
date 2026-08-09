package setup

import (
	"github.com/larsartmann/cqrs-htmx/adminui/v4"
	"github.com/larsartmann/cqrs-htmx/dashboardui/v4"
	"github.com/larsartmann/cqrs-htmx/loginpage/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
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
func New(cfg Config) (*Bundle, error) {
	cfg = cfg.withDefaults()

	// 1. Shared event infrastructure.
	store := cfg.EventStore
	if store == nil {
		store = memorystorage.NewMemoryStore()
	}

	bus := cfg.EventBus
	if bus == nil {
		bus = watermill.NewEventBus()
	}

	// 2. User management service — injects the shared stores.
	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{ //nolint:exhaustruct // setup applies selective defaults
		EventStore:  store,
		EventBus:    bus,
		ReadModelDB: cfg.ReadModelDB,
		AuditLog:    usermgmt.NewAuditLog(),
		TOTP:        cfg.TOTP,
		WebAuthn:    cfg.WebAuthn,
		OAuth2:      cfg.OAuth2,
	})
	if err != nil {
		return nil, errorfamily.WrapRejection(err, "setup.service_creation_failed", "failed to create usermgmt service")
	}

	bundle := &Bundle{
		Service: svc,
		Auth:    usermgmt.NewAuthHandler(svc),
		Stores:  &Stores{EventStore: store, EventBus: bus},
		config:  cfg,
	}

	// 3. Admin panel.
	if !cfg.DisableAdmin {
		admin, err := adminui.New(adminui.Config{ //nolint:exhaustruct // defaults applied internally
			Service:     svc,
			Title:       cfg.Title,
			AccentColor: cfg.AccentColor,
		})
		if err != nil {
			_ = svc.Close()

			return nil, errorfamily.WrapRejection(err, "setup.admin_creation_failed", "failed to create admin panel")
		}

		bundle.Admin = admin
	}

	// 4. CQRS/ES observability dashboard — wired from the shared stores.
	if !cfg.DisableDashboard {
		dashCfg := dashboardui.Config{
			Title:          cfg.Title + " · CQRS Dashboard",
			EventSource:    store,
			EventBus:       bus,
			ProjectionHost: svc.ProjectionHost(),
			PageSize:       25,
		}
		if journal, ok := store.(event.Journal); ok {
			dashCfg.Journal = journal
		}

		dash, err := dashboardui.New(dashCfg)
		if err != nil {
			_ = svc.Close()

			return nil, errorfamily.WrapRejection(
				err,
				"setup.dashboard_creation_failed",
				"failed to create CQRS dashboard",
			)
		}

		bundle.Dashboard = dash
	}

	// 5. Login page.
	if !cfg.DisableLogin {
		login, err := loginpage.New(loginpage.Config{ //nolint:exhaustruct // defaults applied internally
			Service:  svc,
			Title:    cfg.Title,
			Redirect: cfg.LoginRedirect,
		})
		if err != nil {
			_ = svc.Close()

			return nil, errorfamily.WrapRejection(err, "setup.login_creation_failed", "failed to create login page")
		}

		bundle.Login = login
	}

	return bundle, nil
}

// MustNew is like [New] but panics on error. Use for tests and demos where failure is a bug.
func MustNew(cfg Config) *Bundle {
	b, err := New(cfg)
	if err != nil {
		panic(err)
	}

	return b
}
