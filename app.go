// Package cqrshtmx provides HTMX-aware CQRS handler integration with Casbin authorization.
package cqrshtmx

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/dispatcher"
	"github.com/larsartmann/go-cqrs-lite/core/query"
)

// App wires CQRS dispatchers, Casbin authorization, and HTMX response handling
// into a cohesive integration layer.
//
// Create one with New, then use Command and Query to build HTTP handlers
// that automatically handle authorization, dispatch, and HTMX responses.
type App struct {
	commands        *command.Dispatcher
	queries         *query.Dispatcher
	enforcer        Enforcer
	userIDExtractor UserIDExtractor
	errorHandler    ErrorHandler
	loginRedirect   string
	timeout         time.Duration
	maxBodySize     int64

	beforeDispatch BeforeDispatchHook
	afterDispatch  AfterDispatchHook
}

// Config configures an App. Commands or Queries must be non-nil.
type Config struct {
	Commands        *command.Dispatcher
	Queries         *query.Dispatcher
	Enforcer        Enforcer
	UserIDExtractor UserIDExtractor
	ErrorHandler    ErrorHandler
	LoginRedirect   string

	// BeforeDispatch is called before dispatching a command or query.
	// It receives the request context and returns a (possibly modified) context.
	// Use this for setting up tracing spans, request IDs, or timers.
	BeforeDispatch BeforeDispatchHook

	// AfterDispatch is called after dispatching a command or query,
	// regardless of success or failure. The error is nil on success.
	// Use this for logging, metrics, or teardown.
	AfterDispatch AfterDispatchHook

	// Timeout sets a maximum execution time for command and query handlers.
	// A zero or negative value means no timeout (default).
	// When set, dispatch is wrapped in a context with timeout.
	Timeout time.Duration

	// MaxBodySize sets the maximum allowed request body size in bytes.
	// A zero or negative value means no limit (default).
	// When set, request bodies larger than this will be rejected with
	// 413 Request Entity Too Large.
	MaxBodySize int64
}

// BeforeDispatchHook is called before dispatching a command or query.
// It receives the request context and returns a (possibly modified) context
// that will be used for the remainder of the handler.
type BeforeDispatchHook func(ctx context.Context, r *http.Request) context.Context

// AfterDispatchHook is called after dispatching a command or query,
// regardless of whether dispatch succeeded or failed.
// Use this for logging, metrics, or cleanup.
type AfterDispatchHook func(ctx context.Context, r *http.Request, err error)

// New creates an App from the given Config.
// Returns an error if both Commands and Queries are nil.
func New(cfg Config) (*App, error) {
	if cfg.Commands == nil && cfg.Queries == nil {
		return nil, event.NewInfrastructure(
			"config_invalid",
			"[cqrs-htmx] at least one of Commands or Queries must be non-nil",
		)
	}

	loginRedirect := cfg.LoginRedirect
	if loginRedirect == "" {
		loginRedirect = defaultLoginRedirect
	}

	eh := cfg.ErrorHandler
	if eh == nil {
		eh = func(w http.ResponseWriter, r *http.Request, err error) {
			DefaultErrorHandlerWithRedirect(w, r, err, loginRedirect)
		}
	}

	return &App{
		commands:        cfg.Commands,
		queries:         cfg.Queries,
		enforcer:        cfg.Enforcer,
		userIDExtractor: cfg.UserIDExtractor,
		errorHandler:    eh,
		loginRedirect:   loginRedirect,
		timeout:         cfg.Timeout,
		maxBodySize:     cfg.MaxBodySize,
		beforeDispatch:  cfg.BeforeDispatch,
		afterDispatch:   cfg.AfterDispatch,
	}, nil
}

// MustNew is like New but panics on error. Useful for init-time setup where
// failure is a programmer error.
func MustNew(cfg Config) *App {
	app, err := New(cfg)
	if err != nil {
		panic(err)
	}
	return app
}

// HasCommands returns true if the App has a command dispatcher configured.
func (a *App) HasCommands() bool { return a.commands != nil }

// HasQueries returns true if the App has a query dispatcher configured.
func (a *App) HasQueries() bool { return a.queries != nil }

// Command returns an http.HandlerFunc that dispatches a command.
//
// The handler flow:
//  1. Extract user ID from context (via UserIDExtractor middleware)
//  2. Check Casbin authorization (if Authorize option is set)
//  3. Decode the HTTP request into a command.Command (via DecodeJSON, etc.)
//  4. Dispatch the command through the command.Dispatcher
//  5. Apply HTMX response headers (redirect, trigger, push URL)
func (a *App) Command(cmdType command.Type, opts ...HandlerOption) http.HandlerFunc {
	cfg := buildHandlerConfig(opts)
	if cfg.maxBodySize == 0 {
		cfg.maxBodySize = a.maxBodySize
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a.commands == nil {
			a.errorHandler(w, r, errCommandsNil)
			return
		}

		r = a.enrichUserID(r)
		//nolint:contextcheck // ctx is extracted from r inside dispatchContext
		a.handleCommandDispatch(w, r, cmdType, cfg)
	})
}

// Query returns an http.HandlerFunc that dispatches a query.
//
// The handler flow:
//  1. Extract user ID from context (via UserIDExtractor middleware)
//  2. Check Casbin authorization (if Authorize option is set)
//  3. Decode the HTTP request into a query.Query
//  4. Dispatch the query through the query.Dispatcher
//  5. Render the result (via Render option)
//  6. Apply HTMX response headers
func (a *App) Query(qryType query.Type, opts ...HandlerOption) http.HandlerFunc {
	cfg := buildHandlerConfig(opts)
	if cfg.maxBodySize == 0 {
		cfg.maxBodySize = a.maxBodySize
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a.queries == nil {
			a.errorHandler(w, r, errQueriesNil)
			return
		}

		r = a.enrichUserID(r)
		//nolint:contextcheck // ctx is extracted from r inside dispatchContext
		a.handleQueryDispatch(w, r, qryType, cfg)
	})
}

// Middleware returns an HTTP middleware that enriches the request context
// with user identity. Apply this once to your router/mux.
func (a *App) Middleware() func(http.Handler) http.Handler {
	return ContextEnrichmentMiddleware(a.userIDExtractor)
}

// CommandCatalogEntries returns all registered command catalog entries.
// Returns nil if no command dispatcher is configured.
func (a *App) CommandCatalogEntries() map[command.Type]dispatcher.HandlerMeta {
	if a.commands == nil {
		return nil
	}
	return a.commands.CatalogEntries()
}

// QueryCatalogEntries returns all registered query catalog entries.
// Returns nil if no query dispatcher is configured.
func (a *App) QueryCatalogEntries() map[query.Type]dispatcher.HandlerMeta {
	if a.queries == nil {
		return nil
	}
	return a.queries.CatalogEntries()
}

// enrichUserID extracts the user ID if not already present in context.
// This avoids duplicate extraction when both Middleware() and handlers are used.
func (a *App) enrichUserID(r *http.Request) *http.Request {
	if !UserIDFromContext(r.Context()).IsZero() {
		return r
	}

	if a.userIDExtractor == nil {
		return r
	}

	userID, err := a.userIDExtractor(r)
	if err != nil {
		slog.Warn(
			"cqrs-htmx: UserIDExtractor returned error",
			slog.String("error", err.Error()),
		)
		return r
	}
	if userID.IsZero() {
		return r
	}

	return r.WithContext(WithUserID(r.Context(), userID))
}

// noopCancel is a pre-allocated no-op cancel function returned by timeoutCtx
// when no timeout is configured, avoiding a closure allocation per dispatch.
var noopCancel = func() {} //nolint:gochecknoglobals // pre-allocated to avoid per-request closure allocation

// timeoutCtx returns a context with the handler's timeout applied, if configured.
// Falls back to the App's timeout if the handler has no override.
// The caller must call the returned cancel function when done.
func (a *App) timeoutCtx(
	ctx context.Context,
	cfg *handlerConfig,
) (context.Context, context.CancelFunc) {
	t := cfg.timeout
	if t <= 0 {
		t = a.timeout
	}
	if t <= 0 {
		return ctx, noopCancel
	}
	return context.WithTimeout(ctx, t)
}

// afterDispatchHook calls the afterDispatch hook if configured.
func (a *App) afterDispatchHook(ctx context.Context, r *http.Request, err error) {
	if a.afterDispatch != nil {
		a.afterDispatch(ctx, r, err)
	}
}

// HealthHandler returns an HTTP handler that reports dispatcher availability.
// Returns 200 OK with JSON when healthy, 503 Service Unavailable when not.
//
// Usage:
//
//	mux.Handle("/health", app.HealthHandler())
func (a *App) HealthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		hasDispatchers := a.commands != nil || a.queries != nil
		if !hasDispatchers {
			w.Header().Set("Content-Type", ContentTypeJSON)
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"status":"unhealthy","error":"no dispatchers configured"}`))
			return
		}

		w.Header().Set("Content-Type", ContentTypeJSON)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}
}

func buildHandlerConfig(opts []HandlerOption) *handlerConfig {
	cfg := &handlerConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	return cfg
}
