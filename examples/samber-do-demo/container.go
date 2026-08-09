// Package main defines the samber/do composition root for this demo.
//
// This file is the single most important file in the example: it shows every
// canonical samber/do v2 pattern applied to cqrs-htmx. Each provider is
// annotated with the pattern it demonstrates.
package main

import (
	"context"
	"log/slog"

	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	totp "github.com/larsartmann/cqrs-htmx/usermgmt/totp/v4"
	"github.com/samber/do/v2"
)

// AppConfig is the eager-foundation struct registered via ProvideValue.
// It holds values that must exist before any lazy service is constructed.
type AppConfig struct {
	Addr       string
	TOTPIssuer string
	Logger     *slog.Logger
}

// Container wraps the samber/do injector. It is the composition root — the
// ONLY type that is allowed to hold a reference to do.Injector (DO-8 rule).
type Container struct {
	injector do.Injector
}

// NewContainer creates the injector, registers all providers, and returns a
// cleanup function that MUST be deferred by the caller (DO-2 rule: every
// do.New must have a matching Shutdown).
func NewContainer(cfg AppConfig) (*Container, func()) {
	injector := do.New()

	registerProviders(injector, cfg)

	return &Container{injector: injector}, func() {
		// injector.Shutdown() calls Shutdown() on every service that
		// implements do.Shutdowner* — in reverse invocation order.
		// This is how usermgmt.Service.Close() gets called automatically.
		report := injector.Shutdown()
		if len(report.Services) > 0 {
			cfg.Logger.Debug("DI container shut down", "services", len(report.Services))
		}
	}
}

// registerProviders wires every service into the container. Each registration
// demonstrates a specific samber/do pattern.
func registerProviders(injector do.Injector, cfg AppConfig) {
	// --- Eager foundation (ProvideValue) ---
	// Config and logger must exist immediately because lazy providers
	// depend on them at construction time.
	do.ProvideValue(injector, cfg)

	// --- Lazy singletons (Provide) ---
	// Services are constructed on first invocation, exactly once, in
	// dependency order. This is the default for most services.

	do.Provide(injector, func(i do.Injector) (*slog.Logger, error) {
		// Demonstrate overriding logger from config; falls back to default.
		appCfg, err := do.Invoke[AppConfig](i)
		if err != nil {
			return nil, err
		}
		if appCfg.Logger != nil {
			return appCfg.Logger, nil
		}
		return slog.Default(), nil
	})

	// TOTP auth provider — lazy because it only needs to exist when the
	// usermgmt.Service is first invoked. Named so we can show the named-
	// service pattern (multiple auth strategies could coexist).
	do.ProvideNamed(injector, "auth.totp", func(i do.Injector) (usermgmt.TOTPProvider, error) {
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
		totpProvider, err := do.InvokeNamed[usermgmt.TOTPProvider](i, "auth.totp")
		if err != nil {
			return nil, err
		}
		return usermgmt.NewService(usermgmt.ServiceConfig{ //nolint:exhaustruct // demo uses in-memory defaults
			AuditLog: usermgmt.NewAuditLog(),
			TOTP:     totpProvider,
		})
	})

	// Broadcaster — lazy singleton for SSE live updates.
	do.Provide(injector, func(_ do.Injector) (*cqrshtmx.Broadcaster, error) {
		return cqrshtmx.NewBroadcaster(), nil
	})

	// cqrshtmx.App — lazy singleton that wires command/query dispatchers
	// with HTMX handler generation. Depends on nothing from the container
	// directly (dispatchers are created inline for this demo), but in a
	// real app you'd resolve them via do.Invoke.
	do.Provide(injector, func(_ do.Injector) (*cqrshtmx.App, error) {
		return cqrshtmx.MustNew(cqrshtmx.Config{ //nolint:exhaustruct // demo
			// In a real app: resolve dispatchers from the container
			// cmdDisp, _ := do.Invoke[*command.Dispatcher](i)
			// qryDisp, _ := do.Invoke[*query.Dispatcher](i)
		}), nil
	})
}

// --- Lifecycle wrappers ---

// usermgmtServiceLifecycle wraps *usermgmt.Service to implement
// do.ShutdownerWithContextAndError. samber/do's injector.Shutdown() will
// automatically call this method, which delegates to Service.Close().
//
// This is the canonical pattern for integrating third-party types that have
// their own Close/Shutdown methods but don't implement samber/do's interfaces.
type usermgmtServiceLifecycle struct {
	svc *usermgmt.Service
}

// Compile-time guard — catches missing interface methods at build time.
var _ do.ShutdownerWithContextAndError = (*usermgmtServiceLifecycle)(nil)

func (l *usermgmtServiceLifecycle) Shutdown(ctx context.Context) error {
	return l.svc.Close()
}

// Service resolves the usermgmt.Service from the container. This is the
// accessor pattern — callers should use typed methods like this rather than
// raw do.Invoke at every call site.
func (c *Container) Service() (*usermgmt.Service, error) {
	return do.Invoke[*usermgmt.Service](c.injector)
}

// Broadcaster resolves the SSE broadcaster.
func (c *Container) Broadcaster() (*cqrshtmx.Broadcaster, error) {
	return do.Invoke[*cqrshtmx.Broadcaster](c.injector)
}

// App resolves the cqrshtmx.App.
func (c *Container) App() (*cqrshtmx.App, error) {
	return do.Invoke[*cqrshtmx.App](c.injector)
}

// Logger resolves the configured logger.
func (c *Container) Logger() (*slog.Logger, error) {
	return do.Invoke[*slog.Logger](c.injector)
}
