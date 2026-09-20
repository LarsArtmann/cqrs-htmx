// Package main defines the samber/do composition root for this demo.
//
// This file is the single most important file in the example: it shows every
// canonical samber/do v2 pattern applied to cqrs-htmx. Each provider is
// annotated with the pattern it demonstrates.
package main

import (
	"context"
	"log/slog"
	"net/http"

	auditlog "github.com/larsartmann/cqrs-htmx/auditlog/v4"
	health "github.com/larsartmann/cqrs-htmx/health/v4"
	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	totp "github.com/larsartmann/cqrs-htmx/usermgmt/totp/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	errorfamily "github.com/larsartmann/go-error-family"
	gohealth "github.com/larsartmann/go-health"
	healthdashboard "github.com/larsartmann/go-health-dashboard"
	doauditlog "github.com/larsartmann/samber-do-auditlog"
	"github.com/larsartmann/samber-do-auditlog/live"
	"github.com/samber/do/v2"
)

// AppConfig is the eager-foundation struct registered via ProvideValue.
// It holds values that must exist before any lazy service is constructed.
type AppConfig struct {
	Addr       string
	TOTPIssuer string
}

// Container wraps the samber/do injector. It is the composition root — the
// ONLY type that is allowed to hold a reference to do.Injector (DO-8 rule).
type Container struct {
	injector do.Injector

	// AuditViewer is the live audit-log dashboard (an http.Handler). It is
	// built BEFORE the injector because its plugin must be passed as an
	// injector option (do.NewWithOpts) — see NewContainer.
	AuditViewer http.Handler
}

// NewContainer creates the injector, registers all providers, and returns a
// cleanup function that MUST be deferred by the caller (DO-2 rule: every
// do.New must have a matching Shutdown).
func NewContainer(cfg AppConfig) (*Container, func()) {
	// The auditlog bridge (auditlog.WithAuditLog) returns injector OPTIONS:
	// the plugin must be present at construction time to record service
	// invocations, so it is built first and passed to do.NewWithOpts. This
	// is the one call that replaces a bare do.New() when an application
	// wants the audit trail.
	auditSetup, err := auditlog.WithAuditLog(
		doauditlog.Config{},           //nolint:exhaustruct // demo defaults; WithAuditLog enables recording
		live.Config{Prefix: "/audit"}, //nolint:exhaustruct // demo defaults
	)
	if err != nil {
		slog.Error("failed to build auditlog setup", "error", err)
		panic(err)
	}

	injector := do.NewWithOpts(auditSetup.Opts)

	registerProviders(injector, cfg)

	// Eagerly invoke the lifecycle wrapper so that injector.Shutdown() will
	// call usermgmt.Service.Close() even if no consumer ever resolved the
	// Service. This mirrors the ProvideValue pattern for resource-holding
	// services that have background goroutines.
	if _, err := do.Invoke[*serviceLifecycle](injector); err != nil {
		slog.Error("failed to initialize service lifecycle", "error", err)
	}

	// Eagerly invoke the SSE hub lifecycle for the same shutdown-ordering
	// reason as the service lifecycle above.
	if _, err := do.Invoke[*broadcasterLifecycle](injector); err != nil {
		slog.Error("failed to initialize broadcaster lifecycle", "error", err)
	}

	// Eagerly construct the health-gated services (go-health probe +
	// dashboard): a lazily-provided health dashboard reports green precisely
	// when it knows the least (samber-linter HW-4). Constructing them here
	// means the dashboard exists — and shows real projection state — from the
	// first second the server accepts traffic.
	if _, err := do.Invoke[*gohealth.Probe](injector); err != nil {
		slog.Error("failed to initialize health probe", "error", err)
	}
	if _, err := do.Invoke[*healthdashboard.Dashboard](injector); err != nil {
		slog.Error("failed to initialize health dashboard", "error", err)
	}

	return &Container{injector: injector, AuditViewer: auditSetup.Viewer}, func() {
		// injector.Shutdown() calls Shutdown() on every service that
		// implements do.Shutdowner* — in reverse invocation order.
		report := injector.Shutdown()
		if len(report.Services) > 0 {
			slog.Debug("DI container shut down", "services", len(report.Services))
		}
	}
}

// registerProviders wires every service into the container. Each registration
// demonstrates a specific samber/do pattern.
func registerProviders(injector do.Injector, cfg AppConfig) {
	// --- Eager foundation (ProvideValue) ---
	// Config must exist immediately because lazy providers depend on it
	// at construction time.
	do.ProvideValue(injector, cfg)

	// --- Lazy singletons (Provide) ---
	// Services are constructed on first invocation, exactly once, in
	// dependency order. This is the default for most services.

	do.Provide(injector, func(i do.Injector) (*slog.Logger, error) {
		return slog.Default(), nil
	})

	// TOTP auth provider — lazy because it only needs to exist when the
	// usermgmt.Service is first invoked. Named so we can show the named-
	// service pattern (multiple auth strategies could coexist).
	do.ProvideNamed(injector, "auth.totp", func(i do.Injector) (identitymodel.TOTPProvider, error) {
		appCfg, err := do.Invoke[AppConfig](i)
		if err != nil {
			return nil, err
		}
		issuer := appCfg.TOTPIssuer
		if issuer == "" {
			issuer = "cqrs-htmx Demo"
		}
		return totp.New(totp.Config{Issuer: issuer}), nil
	})

	// usermgmt.Service — the core event-sourced identity service.
	// Lazy singleton: built once on first invocation. Resolves the TOTP
	// provider from the container via InvokeNamed, demonstrating the
	// named-service resolution pattern.
	do.Provide(injector, func(i do.Injector) (*usermgmt.Service, error) {
		totpProvider, err := do.InvokeNamed[identitymodel.TOTPProvider](i, "auth.totp")
		if err != nil {
			return nil, err
		}
		return usermgmt.NewService(
			usermgmt.ServiceConfig{ //nolint:exhaustruct // demo uses in-memory defaults
				AuditLog: usermgmt.NewAuditLog(),
				TOTP:     totpProvider,
			},
		)
	})

	// serviceLifecycle — adapts *usermgmt.Service to samber/do's
	// ShutdownerWithContextAndError interface. injector.Shutdown() will
	// call Shutdown(ctx) on this wrapper, which delegates to Close().
	// This is the canonical pattern for integrating third-party types that
	// have their own Close/Shutdown methods but don't implement samber/do's
	// lifecycle interfaces directly.
	do.Provide(injector, func(i do.Injector) (*serviceLifecycle, error) {
		svc, err := do.Invoke[*usermgmt.Service](i)
		if err != nil {
			return nil, err
		}
		return &serviceLifecycle{svc: svc}, nil
	})

	// Projection health probe — the health/v4 bridge builds a go-health
	// Probe with one check per projection worker from the usermgmt.Service
	// (a ProjectionStatusProvider). health.Recorder(svc) additionally merges
	// the injector's own service checks when a go-health ecosystem wants a
	// single recorder for app + infrastructure health.
	do.Provide(injector, func(i do.Injector) (*gohealth.Probe, error) {
		svc, err := do.Invoke[*usermgmt.Service](i)
		if err != nil {
			return nil, err
		}
		return health.NewProbe(svc) //nolint:wrapcheck // demo: bridge errors surface as-is
	})

	// Health dashboard UI — renders the probe as an HTML page + SSE stream.
	// Mounted by main.go at /health-ui.
	do.Provide(injector, func(i do.Injector) (*healthdashboard.Dashboard, error) {
		probe, err := do.Invoke[*gohealth.Probe](i)
		if err != nil {
			return nil, err
		}
		return health.NewDashboard(probe), nil
	})

	// Broadcaster lifecycle — registers the SSE hub behind a wrapper that
	// implements BOTH do.ShutdownerWithContextAndError and do.Healthchecker
	// (the raw *cqrshtmx.Broadcaster only implements samber/do's Shutdowner
	// half; its Health method uses a different, context-less shape). The
	// wrapper pattern is the canonical fix when a third-party type cannot
	// grow the missing interface method itself.
	do.Provide(injector, func(_ do.Injector) (*broadcasterLifecycle, error) {
		return &broadcasterLifecycle{hub: cqrshtmx.NewBroadcaster()}, nil
	})

	// cqrshtmx.App — lazy singleton that wires command/query dispatchers
	// with HTMX handler generation. Resolves the command dispatcher from
	// the container, showing provider-to-provider dependency resolution.
	do.Provide(injector, func(i do.Injector) (*cqrshtmx.App, error) {
		disp, err := do.Invoke[*command.Dispatcher](i)
		if err != nil {
			return nil, err
		}
		return cqrshtmx.New(cqrshtmx.Config{ //nolint:exhaustruct // demo
			Commands: disp,
		})
	})

	// Command dispatcher — lazy singleton. Registering handlers HERE (before
	// returning the dispatcher) is the canonical DI pattern: by the time the
	// App resolves this dispatcher, every command handler is registered.
	do.Provide(injector, func(_ do.Injector) (*command.Dispatcher, error) {
		disp := command.NewDispatcher()
		if err := command.RegisterTyped(disp, "Hello",
			func(_ context.Context, cmd *helloCmd) error {
				if cmd.Name == "" {
					return errorfamily.NewRejection("samber_do_demo.hello.name_empty",
						"name must not be empty")
				}
				return nil
			}); err != nil {
			return nil, err
		}
		return disp, nil
	})
}

// --- Demo command ---

// helloRequest is the HTTP body for POST /command/hello.
type helloRequest struct {
	Name string `json:"name"`
}

// helloCmd is a custom command.Command dispatched through the DI-managed
// dispatcher and handled by the provider above.
type helloCmd struct {
	*command.BasicCommand
	Name string
}

// --- Lifecycle adapter ---

// serviceLifecycle wraps *usermgmt.Service to implement
// do.ShutdownerWithContextAndError. samber/do's injector.Shutdown() will
// automatically call this method, which delegates to Service.Close().
//
// This is the canonical pattern for adapting third-party Close()-based types
// into samber/do's lifecycle interface.
type serviceLifecycle struct {
	svc *usermgmt.Service
}

// Compile-time guards — catch missing interface methods at build time.
var _ do.ShutdownerWithContextAndError = (*serviceLifecycle)(nil)
var _ do.Healthchecker = (*serviceLifecycle)(nil)

func (l *serviceLifecycle) Shutdown(_ context.Context) error {
	return l.svc.Close()
}

// HealthCheck reports the service unhealthy while any projection worker is
// still draining or has exhausted its restart budget. It delegates to the
// library's own readiness gate so the DI health dashboard and the /health
// endpoint always agree on what "healthy" means (samber-linter HW-1).
func (l *serviceLifecycle) HealthCheck() error {
	return cqrshtmx.ProjectionReadinessCheck(l.svc).Check()
}

// broadcasterLifecycle adapts *cqrshtmx.Broadcaster to samber/do's
// Shutdowner + Healthchecker pair. The hub's own Health() state (closed /
// draining) becomes the health verdict, so a dead SSE feed surfaces on the
// DI health dashboard instead of rendering an unconditional pass.
type broadcasterLifecycle struct {
	hub *cqrshtmx.Broadcaster
}

var (
	_ do.ShutdownerWithContextAndError = (*broadcasterLifecycle)(nil)
	_ do.Healthchecker                 = (*broadcasterLifecycle)(nil)
)

func (l *broadcasterLifecycle) Shutdown(ctx context.Context) error {
	return l.hub.Shutdown(ctx)
}

func (l *broadcasterLifecycle) HealthCheck() error {
	return cqrshtmx.HubReadinessCheck(l.hub).Check()
}

// --- Typed accessors ---
// Callers should use typed methods like these rather than raw do.Invoke at
// every call site. This centralizes resolution and makes refactoring easier.

// Service resolves the usermgmt.Service from the container.
func (c *Container) Service() (*usermgmt.Service, error) {
	return do.Invoke[*usermgmt.Service](c.injector)
}

// Broadcaster resolves the SSE broadcaster (registered behind its
// lifecycle wrapper; the hub itself is the wrapper's field).
func (c *Container) Broadcaster() (*cqrshtmx.Broadcaster, error) {
	lifecycle, err := do.Invoke[*broadcasterLifecycle](c.injector)
	if err != nil {
		return nil, err
	}

	return lifecycle.hub, nil
}

// App resolves the cqrshtmx.App.
func (c *Container) App() (*cqrshtmx.App, error) {
	return do.Invoke[*cqrshtmx.App](c.injector)
}

// Logger resolves the configured logger.
func (c *Container) Logger() (*slog.Logger, error) {
	return do.Invoke[*slog.Logger](c.injector)
}

// HealthDashboard resolves the projection health dashboard UI (health/v4
// bridge over the usermgmt.Service's projection workers).
func (c *Container) HealthDashboard() (*healthdashboard.Dashboard, error) {
	return do.Invoke[*healthdashboard.Dashboard](c.injector)
}
