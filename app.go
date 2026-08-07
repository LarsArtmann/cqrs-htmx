package cqrshtmx

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
	errorfamily "github.com/larsartmann/go-error-family"
	servertiming "github.com/larsartmann/httputil/server_timing"
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
	serviceName     string

	beforeDispatch BeforeDispatchHook
	afterDispatch  AfterDispatchHook
	serverTiming   func(*http.Request) bool
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

	// ServerTiming, when set, enables the W3C Server-Timing response header for
	// requests where the predicate returns true. This wraps every App.Command()
	// and App.Query() handler — no separate middleware needed. Nil (default)
	// means disabled (zero overhead). Use ServerTimingMiddleware() for routes
	// outside the App.
	//
	//	app, _ := cqrshtmx.New(cqrshtmx.Config{
	//	    // ...
	//	    ServerTiming: func(r *http.Request) bool {
	//	        return r.URL.Query().Has("debug") // or: isAdmin, isLocalhost, etc.
	//	    },
	//	})
	ServerTiming func(*http.Request) bool

	// IncludeRequestIDInErrors makes the default error handler include the
	// request ID in error responses when one is present in the request context.
	// This helps clients correlate errors with server logs.
	// Only applies when ErrorHandler is not set (uses the default handler).
	IncludeRequestIDInErrors bool

	// IncludeInternalDetails keeps the raw error message in 5xx responses.
	// By default the App's error handler redacts server-fault detail
	// (Corruption/Infrastructure/Transient) to the error family's generic
	// public-safe message, so internal wiring and infrastructure addresses are
	// not leaked to clients. The full detail is always logged server-side.
	// Set to true for local development or trusted internal networks.
	// Only applies when ErrorHandler is not set (uses the default handler).
	IncludeInternalDetails bool

	// ServiceName identifies the service emitting events. It is included
	// in event metadata (event.WithSource) by App.EventOptions and propagates
	// to downstream event consumers. Must be a valid event source identifier
	// (non-empty, ASCII printable). Invalid values are silently dropped.
	ServiceName string
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
func New(config Config) (*App, error) {
	if config.Commands == nil && config.Queries == nil {
		return nil, errorfamily.NewInfrastructure(
			"config_invalid",
			"[cqrs-htmx] at least one of Commands or Queries must be non-nil",
		)
	}

	loginRedirect := config.LoginRedirect
	if loginRedirect == "" {
		loginRedirect = defaultLoginRedirect
	}

	errorHandler := config.ErrorHandler
	if errorHandler == nil {
		includeInternal := config.IncludeInternalDetails
		includeRequestID := config.IncludeRequestIDInErrors
		errorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
			handleErrorCore(w, r, err, loginRedirect, plainBodyWriter(r, includeInternal, includeRequestID))
		}
	}

	return &App{
		commands:        config.Commands,
		queries:         config.Queries,
		enforcer:        config.Enforcer,
		userIDExtractor: config.UserIDExtractor,
		errorHandler:    errorHandler,
		loginRedirect:   loginRedirect,
		timeout:         config.Timeout,
		maxBodySize:     config.MaxBodySize,
		serviceName:     config.ServiceName,
		beforeDispatch:  config.BeforeDispatch,
		afterDispatch:   config.AfterDispatch,
		serverTiming:    config.ServerTiming,
	}, nil
}

// MustNew is like New but panics on error. Useful for init-time setup where
// failure is a programmer error.
func MustNew(config Config) *App {
	app, err := New(config)
	if err != nil {
		panic(err)
	}

	return app
}

// HasCommands returns true if the App has a command dispatcher configured.
func (a *App) HasCommands() bool { return a.commands != nil }

// HasQueries returns true if the App has a query dispatcher configured.
func (a *App) HasQueries() bool { return a.queries != nil }

// ServiceName returns the configured service name (empty if not set).
// Used by EventOptions to set the event source for downstream consumers.
func (a *App) ServiceName() string { return a.serviceName }

// EventOptions returns event.Options built from the request context and
// the App's configured ServiceName. Use this in command/query handlers
// to pass to event dispatchers so emitted events carry user identity,
// correlation IDs, request IDs, deadlines, and the service source.
//
// Returns nil options if none of user ID, correlation ID, request ID,
// deadline, or service source is set.
func (a *App) EventOptions(ctx context.Context) []event.Option {
	opts := EventOptionsFromContext(ctx)
	if a == nil || a.serviceName == "" {
		return opts
	}

	if src, err := event.ParseSource(a.serviceName); err == nil {
		opts = append(opts, event.WithSource(src))
	}

	return opts
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
	config := a.buildHandlerConfigChecked(cmdType.IsZero(), "command", opts)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a.commands == nil {
			a.errorHandler(w, r, errCommandsNil)

			return
		}

		w, r = a.applyServerTiming(w, r)
		r = a.enrichUserID(r)
		//nolint:contextcheck // ctx is extracted from r inside dispatchContext
		a.handleCommandDispatch(w, r, cmdType, config)
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
	config := a.buildHandlerConfigChecked(qryType.IsZero(), "query", opts)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a.queries == nil {
			a.errorHandler(w, r, errQueriesNil)

			return
		}

		w, r = a.applyServerTiming(w, r)
		r = a.enrichUserID(r)
		//nolint:contextcheck // ctx is extracted from r inside dispatchContext
		a.handleQueryDispatch(w, r, qryType, config)
	})
}

// CommandTyped returns a typed http.HandlerFunc that dispatches a command of
// type Q. The decoder must return a value that implements command.Command and is
// assignable to Q; if it returns a different type, the handler responds with
// 400 Bad Request. Use this with DecodeJSONTyped[Q] or DecodeFormTyped[Q].
//
// Example:
//
//	cqrshtmx.CommandTyped[*CreateItemCmd](app, "CreateItem",
//	    cqrshtmx.DecodeJSONTyped[*CreateItemCmd](),
//	)
func CommandTyped[Q command.Command](a *App, cmdType command.Type, opts ...HandlerOption) http.HandlerFunc {
	config := a.buildHandlerConfigChecked(cmdType.IsZero(), "command", opts)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a.commands == nil {
			a.errorHandler(w, r, errCommandsNil)

			return
		}

		w, r = a.applyServerTiming(w, r)
		r = a.enrichUserID(r)
		handleCommandTypedDispatch[Q](a, w, r, cmdType, config)
	})
}

// QueryTyped returns a typed http.HandlerFunc that dispatches a query of type Q
// and expects a result of type R. The decoder must return a value that implements
// query.Query and is assignable to Q. Use this with DecodeJSONQueryTyped[Q] and
// a typed renderer such as RenderJSON[R]().
//
// Example:
//
//	cqrshtmx.QueryTyped[*GetUserQuery, *User](app, "GetUser",
//	    cqrshtmx.DecodeJSONQueryTyped[*GetUserQuery](),
//	    cqrshtmx.RenderJSON[*User](),
//	)
func QueryTyped[Q query.Query, R any](a *App, qryType query.Type, opts ...HandlerOption) http.HandlerFunc {
	config := a.buildHandlerConfigChecked(qryType.IsZero(), "query", opts)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a.queries == nil {
			a.errorHandler(w, r, errQueriesNil)

			return
		}

		w, r = a.applyServerTiming(w, r)
		r = a.enrichUserID(r)
		handleQueryTypedDispatch[Q, R](a, w, r, qryType, config)
	})
}

// buildHandlerConfigChecked is the shared entry-point preamble for App.Command
// and App.Query: it validates that the caller passed a non-empty command/query
// type, then resolves the HandlerConfig (falling back to the App's default
// maxBodySize when the option wasn't set). kind is "command" or "query" and is
// used only in the panic message.
func (a *App) buildHandlerConfigChecked(typeIsZero bool, kind string, opts []HandlerOption) *handlerConfig {
	if typeIsZero {
		//cqrs-lint:ignore(C009) programmer error: empty command/query type at registration
		panic(
			"cqrs-htmx: " + kind + " type must not be empty",
		)
	}

	config := buildHandlerConfig(opts)
	if config.maxBodySize == 0 {
		config.maxBodySize = a.maxBodySize
	}

	return config
}

// Middleware returns an HTTP middleware that enriches the request context
// with user identity. Apply this once to your router/mux.
func (a *App) Middleware() func(http.Handler) http.Handler {
	return ContextEnrichmentMiddleware(a.userIDExtractor)
}

// applyServerTiming wraps the ResponseWriter and injects a *ServerTiming
// collector into the request context when Config.ServerTiming is set and the
// predicate returns true for this request. When disabled (nil predicate or
// predicate returns false), returns the original writer and request unchanged —
// zero overhead.
func (a *App) applyServerTiming(w http.ResponseWriter, r *http.Request) (http.ResponseWriter, *http.Request) {
	if a.serverTiming == nil || !a.serverTiming(r) {
		return w, r
	}

	return servertiming.WrapServerTiming(w, r)
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
		// A failing extractor makes this request anonymous. That is safe ONLY
		// when the handler gates with Authorize/RequireAuth (anonymous then
		// fails the gate normally); handlers without a gate will process the
		// request anonymously by design. Log at Error level — not Warn — so a
		// broken extractor (the common cause: misconfigured session middleware)
		// is loud and debuggable instead of silently degrading every request
		// to anonymous. The library does not force a 401 here because that
		// would break consumers whose extractors legitimately error for
		// not-yet-authenticated requests.
		slog.Error(
			"cqrs-htmx: UserIDExtractor returned error — request proceeds anonymously",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
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
var noopCancel = func() {}

// timeoutCtx returns a context with the handler's timeout applied, if configured.
// Falls back to the App's timeout if the handler has no override (or config is nil).
// The caller must call the returned cancel function when done.
func (a *App) timeoutCtx(
	ctx context.Context,
	config *handlerConfig,
) (context.Context, context.CancelFunc) {
	var t time.Duration
	if config != nil {
		t = config.timeout
	}

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
var (
	healthyBody   = []byte(`{"status":"ok"}`)
	unhealthyBody = []byte(`{"status":"unhealthy","error":"no dispatchers configured"}`)
)

// HealthHandler returns an http.HandlerFunc that reports 200 OK when the App
// has at least one dispatcher configured, or 503 otherwise. Pre-allocates
// response bodies for zero-allocation health checks.
func (a *App) HealthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		hasDispatchers := a.commands != nil || a.queries != nil
		if !hasDispatchers {
			w.Header().Set("Content-Type", ContentTypeJSON)
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write(unhealthyBody)

			return
		}

		w.Header().Set("Content-Type", ContentTypeJSON)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(healthyBody)
	}
}

func buildHandlerConfig(opts []HandlerOption) *handlerConfig {
	config := &handlerConfig{}
	for _, opt := range opts {
		opt(config)
	}

	return config
}
