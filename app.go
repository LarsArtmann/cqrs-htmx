// Package cqrshtmx provides HTMX-aware CQRS handler integration with Casbin authorization.
package cqrshtmx

import (
	"context"
	"net/http"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/larsartmann/go-cqrs-lite/core/command"
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
		return nil, errors.New("[cqrs-htmx] at least one of Commands or Queries must be non-nil")
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
		beforeDispatch:  cfg.BeforeDispatch,
		afterDispatch:   cfg.AfterDispatch,
	}, nil
}

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

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a.commands == nil {
			a.errorHandler(w, r, ErrCommandsNil)
			return
		}

		r = a.enrichUserID(r)
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

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a.queries == nil {
			a.errorHandler(w, r, ErrQueriesNil)
			return
		}

		r = a.enrichUserID(r)
		a.handleQueryDispatch(w, r, qryType, cfg)
	})
}

// Middleware returns an HTTP middleware that enriches the request context
// with user identity. Apply this once to your router/mux.
func (a *App) Middleware() func(http.Handler) http.Handler {
	return ContextEnrichmentMiddleware(a.userIDExtractor)
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

	userIDStr := a.userIDExtractor(r)
	if userIDStr == "" {
		return r
	}

	userID, err := ParseUserID(userIDStr)
	if err != nil {
		return r
	}

	return r.WithContext(WithUserID(r.Context(), userID))
}

// dispatchWithTimeout runs a command dispatch with the App's timeout, if configured.
func (a *App) dispatchWithTimeout(ctx context.Context, fn func(context.Context) error) error {
	if a.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, a.timeout)
		defer cancel()
	}
	return fn(ctx)
}

// dispatchQueryWithTimeout runs a query dispatch with the App's timeout, if configured.
func (a *App) dispatchQueryWithTimeout(
	ctx context.Context,
	_ query.Type,
	qry query.Query,
) (any, error) {
	if a.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, a.timeout)
		defer cancel()
	}

	result, dispatchErr := a.queries.Dispatch(ctx, qry) //nolint:wrapcheck,golines // error is wrapped by caller
	return result, dispatchErr
}

func buildHandlerConfig(opts []HandlerOption) *handlerConfig {
	cfg := &handlerConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	return cfg
}
