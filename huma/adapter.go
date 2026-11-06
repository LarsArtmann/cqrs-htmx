package huma

import (
	"context"
	"fmt"
	"time"

	"github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/cqrs-htmx/v4/httperr"
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// Config configures an Adapter.
type Config struct {
	// Commands dispatches commands. Required for RegisterCommand.
	Commands *command.Dispatcher
	// Queries dispatches queries. Required for RegisterQuery.
	Queries *query.Dispatcher
	// Checker answers the authorization questions the Authorize options ask.
	// Pass the same engine the HTTP endpoints use (*identitymodel.Authz
	// satisfies this structurally) so a grant means one thing everywhere.
	// A nil Checker is not a silent allow: an operation that declares an
	// Authorize option answers 503 while the engine is missing, because a
	// broken authorization setup must not look like a permissions problem.
	Checker cqrshtmx.PermissionChecker
	// Timeout bounds every dispatch. Zero means no timeout. An operation can
	// override it with WithTimeout.
	Timeout time.Duration
}

// Adapter runs the cqrs-htmx pipeline stages for Huma operations.
//
// Build one with New and hand it to RegisterCommand/RegisterQuery. The same
// value is safe for concurrent use: it holds configuration, never per-request
// state.
type Adapter struct {
	commands *command.Dispatcher
	queries  *query.Dispatcher
	checker  cqrshtmx.PermissionChecker
	timeout  time.Duration
}

// New builds an Adapter from cfg.
//
// It fails when neither Commands nor Queries is set: an adapter that cannot
// dispatch anything would answer every operation with an empty success.
func New(cfg Config) (*Adapter, error) {
	if cfg.Commands == nil && cfg.Queries == nil {
		return nil, errorfamily.NewInfrastructure("config_invalid",
			"[cqrs-htmx/huma] at least one of Commands or Queries must be non-nil")
	}

	return &Adapter{
		commands: cfg.Commands,
		queries:  cfg.Queries,
		checker:  cfg.Checker,
		timeout:  cfg.Timeout,
	}, nil
}

// HasCommands reports whether a command dispatcher is configured.
func (a *Adapter) HasCommands() bool { return a.commands != nil }

// HasQueries reports whether a query dispatcher is configured.
func (a *Adapter) HasQueries() bool { return a.queries != nil }

// DispatchCommand dispatches an already-built command with the request metadata
// from ctx (actor, user, correlation, request ID) applied first.
//
// It is the dispatch stage RegisterCommand runs, exported for consumers who
// need a Huma handler shape the registration helpers do not cover (custom
// output types, response headers, a second dispatch). Errors come back
// unwrapped by classification, so MapError turns them into the problem
// document a client expects.
func (a *Adapter) DispatchCommand(ctx context.Context, cmd command.Command) error {
	return a.dispatchCommand(ctx, 0, cmd)
}

// DispatchQuery dispatches an already-built query and returns its result.
//
// It is the dispatch stage RegisterQuery runs. Use MapError on the returned
// error before handing it back to Huma.
func (a *Adapter) DispatchQuery(ctx context.Context, qry query.Query) (any, error) {
	return a.dispatchQuery(ctx, 0, qry)
}

// Authorize runs the authorization stage for one question.
//
// The subject defaults to the authenticated user in ctx; an empty Domain means
// the wildcard domain, matching cqrshtmx.Authorize. A denial comes back as the
// classified error httperr.MapError turns into 401 (nobody is authenticated) or
// 403 (policy refused), and an engine failure is returned as an error rather
// than a denial, so a broken policy engine answers 503 instead of pretending
// the caller lacked permission.
func (a *Adapter) Authorize(ctx context.Context, req cqrshtmx.Request) error {
	return a.authorizeRequest(ctx, req)
}

// dispatchCommand is the shared command dispatch: metadata enrichment, the
// timeout, then the dispatcher.
func (a *Adapter) dispatchCommand(ctx context.Context, override time.Duration, cmd command.Command) error {
	if a.commands == nil {
		return errorfamily.NewInfrastructure("commands_nil",
			"[cqrs-htmx/huma] command dispatcher is required").WithCause(httperr.ErrCommandsNil)
	}

	enrichCommand(ctx, cmd)

	ctx, cancel := a.dispatchContext(ctx, override)
	defer cancel()

	return a.commands.Dispatch(ctx, cmd) //nolint:wrapcheck // callers map the error; a wrap would shadow its context
}

// dispatchQuery is the query-side mirror of dispatchCommand.
func (a *Adapter) dispatchQuery(ctx context.Context, override time.Duration, qry query.Query) (any, error) {
	if a.queries == nil {
		return nil, errorfamily.NewInfrastructure("queries_nil",
			"[cqrs-htmx/huma] query dispatcher is required").WithCause(httperr.ErrQueriesNil)
	}

	enrichQuery(ctx, qry)

	ctx, cancel := a.dispatchContext(ctx, override)
	defer cancel()

	result, err := a.queries.Dispatch(ctx, qry) //nolint:wrapcheck // callers map the error; a wrap would shadow its context
	if err != nil {
		return nil, err
	}

	return result, nil
}

// dispatchContext applies the per-operation timeout when it is set and the
// adapter timeout otherwise.
func (a *Adapter) dispatchContext(ctx context.Context, override time.Duration) (context.Context, context.CancelFunc) {
	timeout := override
	if timeout <= 0 {
		timeout = a.timeout
	}

	if timeout <= 0 {
		return ctx, noopCancel
	}

	return context.WithTimeout(ctx, timeout)
}

// noopCancel is the cancel function of a dispatch with no timeout.
func noopCancel() {}

// authorizeRequest answers one authorization question, defaulting the subject
// to the authenticated user and an empty domain to the wildcard domain.
func (a *Adapter) authorizeRequest(ctx context.Context, req cqrshtmx.Request) error {
	if req.Subject == "" {
		req.Subject = cqrshtmx.UserIDFromContext(ctx).String()
	}

	decision, err := cqrshtmx.Decide(a.checker, req)
	if err != nil {
		return err
	}

	if decision.Allowed {
		return nil
	}

	return denialError(req, decision)
}

// denialError mirrors the denial shape the root package produces for its
// Authorize options: a rejection whose cause is the shared unauthorized or
// forbidden sentinel, so httperr.MapError answers 401 or 403 and the problem
// document carries the same code on every transport.
func denialError(req cqrshtmx.Request, decision cqrshtmx.Decision) error {
	code := httperr.CodeForbidden
	sentinel := httperr.ErrForbidden

	if decision.Reason == cqrshtmx.ReasonNotAuthenticated {
		code = httperr.CodeUnauthorized
		sentinel = httperr.ErrUnauthorized
	}

	return errorfamily.NewRejection(code,
		fmt.Sprintf("subject=%q domain=%q resource=%q action=%q: %s",
			req.Subject, req.Domain, req.Resource.String(), req.Action, string(decision.Reason)),
	).WithCause(sentinel)
}

// requireAuthenticated reports the 401 a RequireAuth operation answers when the
// request carries no authenticated user.
func requireAuthenticated(ctx context.Context, resource, action string) error {
	if !cqrshtmx.UserIDFromContext(ctx).IsZero() {
		return nil
	}

	return errorfamily.NewRejection(httperr.CodeUnauthorized,
		"authentication required for "+resource+"/"+action).
		WithCause(httperr.ErrUnauthorized)
}
