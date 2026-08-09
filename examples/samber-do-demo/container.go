// Package main defines the samber/do composition root for this demo.
//
// This file is the single most important file in the example: it shows every
// canonical samber/do v2 pattern applied to cqrs-htmx. Each provider is
// annotated with the pattern it demonstrates.
package main

import (
	"context"
	"log/slog"

	totp "github.com/larsartmann/cqrs-htmx/usermgmt/totp/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/go-cqrs-lite/command/v4"
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
}

// NewContainer creates the injector, registers all providers, and returns a
// cleanup function that MUST be deferred by the caller (DO-2 rule: every
// do.New must have a matching Shutdown).
func NewContainer(cfg AppConfig) (*Container, func()) {
	injector := do.New()

	registerProviders(injector, cfg)

	// Eagerly invoke the lifecycle wrapper so that injector.Shutdown() will
	// call usermgmt.Service.Close() even if no consumer ever resolved the
	// Service. This mirrors the ProvideValue pattern for resource-holding
	// services that have background goroutines.
	if _, err := do.Invoke[*serviceLifecycle](injector); err != nil {
		slog.Error("failed to initialize service lifecycle", "error", err)
	}

	return &Container{injector: injector}, func() {
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

	// Broadcaster — lazy singleton for SSE live updates.
	do.Provide(injector, func(_ do.Injector) (*cqrshtmx.Broadcaster, error) {
		return cqrshtmx.NewBroadcaster(), nil
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

	// Command dispatcher — lazy singleton. In a real app you'd register
	// handlers here too, before returning the dispatcher.
	do.Provide(injector, func(_ do.Injector) (*command.Dispatcher, error) {
		return command.NewDispatcher(), nil
	})
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

// Compile-time guard — catches missing interface methods at build time.
var _ do.ShutdownerWithContextAndError = (*serviceLifecycle)(nil)

func (l *serviceLifecycle) Shutdown(_ context.Context) error {
	return l.svc.Close()
}

// --- Typed accessors ---
// Callers should use typed methods like these rather than raw do.Invoke at
// every call site. This centralizes resolution and makes refactoring easier.

// Service resolves the usermgmt.Service from the container.
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
