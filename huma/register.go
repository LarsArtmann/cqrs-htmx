package huma

import (
	"context"
	"log/slog"

	humav2 "github.com/danielgtaylor/huma/v2"
	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/cqrs-htmx/v4/httperr"
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// RegisterCommand registers op on api and dispatches the command build returns.
//
// Huma decodes and validates the typed input; the adapter then runs the stages
// the endpoint builder runs, in the same order: authorize → build → validate →
// guard → dispatch. Every error is rendered as the RFC 9457 problem document
// http.NewErrorHandler would emit, so a Huma client and a route-based client see
// one error contract.
//
// The success response carries no body: Huma answers op.DefaultStatus (204
// unless set). Set DefaultStatus 201/202 for a create or an accepted command.
//
// A command handler that fails answers the status its error family maps to
// (rejection 400, conflict 409, transient 503, corruption 500). An operation
// that declares an Authorize option answers 401 for an anonymous caller, 403
// for a policy denial, and 503 when no Checker is configured at all.
func RegisterCommand[I any](
	api humav2.API,
	a *Adapter,
	op humav2.Operation,
	build func(ctx context.Context, in *I) (command.Command, error),
	opts ...Option,
) {
	requireAPI(api)
	requireAdapter(a)
	requireBuild("RegisterCommand", build)

	cfg := newOperationConfig(opts)

	humav2.Register(api, op, func(ctx context.Context, in *I) (*struct{}, error) {
		if err := a.authorize(ctx, cfg, in); err != nil {
			return nil, a.fail(ctx, op, err)
		}

		cmd, err := build(ctx, in)
		if err != nil {
			return nil, a.fail(ctx, op, err)
		}

		if cmd == nil {
			return nil, a.fail(ctx, op, httperr.ErrDecoderReturnedNil)
		}

		if err := validateCommandValue(cmd, cfg.validateCommand); err != nil {
			return nil, a.fail(ctx, op, err)
		}

		if err := runGuard(ctx, cfg, cmd); err != nil {
			return nil, a.fail(ctx, op, err)
		}

		if err := a.dispatchCommand(ctx, cfg.timeout, cmd); err != nil {
			return nil, a.fail(ctx, op, err)
		}

		return nil, nil
	})
}

// RegisterQuery registers op on api and answers it with the result build's
// query returns.
//
// The stage order matches RegisterCommand: authorize → build → validate → guard
// → dispatch, then respond turns the dispatch result into the operation's typed
// output. respond receives the result as any because a query dispatcher answers
// any; assert it to the concrete result type the query handler registered.
//
// Errors follow the same contract as RegisterCommand.
func RegisterQuery[I, O any](
	api humav2.API,
	a *Adapter,
	op humav2.Operation,
	build func(ctx context.Context, in *I) (query.Query, error),
	respond func(ctx context.Context, in *I, result any) (*O, error),
	opts ...Option,
) {
	requireAPI(api)
	requireAdapter(a)
	requireBuild("RegisterQuery", build)

	if respond == nil {
		panic("[cqrs-htmx/huma] RegisterQuery requires a respond function")
	}

	cfg := newOperationConfig(opts)

	humav2.Register(api, op, func(ctx context.Context, in *I) (*O, error) {
		if err := a.authorize(ctx, cfg, in); err != nil {
			return nil, a.fail(ctx, op, err)
		}

		qry, err := build(ctx, in)
		if err != nil {
			return nil, a.fail(ctx, op, err)
		}

		if qry == nil {
			return nil, a.fail(ctx, op, httperr.ErrDecoderReturnedNil)
		}

		if err := validateQueryValue(qry, cfg.validateQuery); err != nil {
			return nil, a.fail(ctx, op, err)
		}

		if err := runGuard(ctx, cfg, qry); err != nil {
			return nil, a.fail(ctx, op, err)
		}

		result, err := a.dispatchQuery(ctx, cfg.timeout, qry)
		if err != nil {
			return nil, a.fail(ctx, op, err)
		}

		output, err := respond(ctx, in, result)
		if err != nil {
			return nil, a.fail(ctx, op, err)
		}

		return output, nil
	})
}

// authorize runs the operation's authorization posture.
//
// An empty resource or action means the operation only requires authentication
// (RequireAuth) or is public, so no policy question is asked.
func (a *Adapter) authorize(ctx context.Context, cfg *operationConfig, in any) error {
	switch cfg.mode {
	case authNone:
		return nil
	case authRequired:
		return requireAuthenticated(ctx, cfg.resource, cfg.action)
	case authAuthorized:
		req := cqrshtmx.Request{ //nolint:exhaustruct_v5 // Subject defaults from ctx below
			Domain:   cfg.domain,
			Resource: cqrshtmx.Resource{Type: cfg.resource, ID: ""},
			Action:   cfg.action,
		}

		if cfg.resolve != nil {
			resolved, err := cfg.resolve(ctx, in)
			if err != nil {
				return err
			}

			req = resolved
		}

		return a.authorizeRequest(ctx, req)
	default:
		return nil
	}
}

// runGuard runs the operation's guard, when it has one.
func runGuard(ctx context.Context, cfg *operationConfig, value any) error {
	if cfg.guard == nil {
		return nil
	}

	return cfg.guard(ctx, value)
}

// fail logs a classified pipeline error and converts it into the value Huma
// writes. Errors the consumer already shaped as Huma errors (huma.Error422...,
// a custom StatusError) pass through untouched: the consumer said what the
// response should be, and re-wrapping it would relabel a 422 as a 503.
func (a *Adapter) fail(ctx context.Context, op humav2.Operation, err error) error {
	if !isStatusError(err) {
		// One record per failure, level follows the family: Transient logs WARN
		// (expected to self-heal), everything else ERROR. The operation identity
		// rides along so a 503 in the logs names its route.
		errorfamily.LogErrorContext(ctx, err, slog.Default().With(
			slog.String("huma.operation", op.OperationID),
			slog.String("huma.method", op.Method),
			slog.String("huma.path", op.Path),
		))
	}

	return MapError(ctx, err)
}

// requireAPI rejects a nil Huma API at registration — a programmer error, like
// an empty command type on App.Command.
func requireAPI(api humav2.API) {
	if api == nil {
		panic("[cqrs-htmx/huma] a non-nil huma.API is required")
	}
}

// requireAdapter rejects a nil adapter at registration.
func requireAdapter(a *Adapter) {
	if a == nil {
		panic("[cqrs-htmx/huma] a non-nil *Adapter is required — build one with New")
	}
}

// requireBuild rejects a nil build function at registration.
func requireBuild[T any](helper string, build T) {
	if any(build) == nil {
		panic("[cqrs-htmx/huma] " + helper + " requires a build function")
	}
}
