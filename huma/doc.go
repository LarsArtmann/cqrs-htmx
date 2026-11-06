// Package huma wires the cqrs-htmx CQRS pipeline into Huma v2 operations.
//
// Huma owns the typed contract: it registers operations, decodes and validates
// the request, picks the status code and generates the OpenAPI document. An
// http.HandlerFunc from cqrs-htmx cannot participate in that contract, which
// left OpenAPI-first apps with two disconnected worlds: Huma handlers
// re-implementing authorization and error mapping per operation while the
// cqrs-htmx pipeline served nothing.
//
// This module is the bridge. It runs the same pipeline stages the endpoint
// builder runs (authorize, validate, guard, dispatch, map error) inside a Huma
// handler, and it renders failures as the same RFC 9457 problem document the
// HTTP endpoints emit. Huma still owns decode, input-schema validation, status
// codes and OpenAPI generation.
//
// # Quick Start
//
// Import the adapter as cqrsadapter, because upstream Huma already owns the
// name huma:
//
//	import (
//	    huma "github.com/danielgtaylor/huma/v2"
//	    "github.com/danielgtaylor/huma/v2/adapters/humago"
//	    "github.com/larsartmann/cqrs-htmx/v4"
//	    cqrsadapter "github.com/larsartmann/cqrs-htmx/huma/v4"
//	)
//
//	pipe, err := cqrsadapter.New(cqrsadapter.Config{
//	    Commands: cmdDisp,
//	    Checker:  svc.Authz(), // any cqrshtmx.PermissionChecker
//	})
//	if err != nil {
//	    panic(err)
//	}
//
//	api := humago.New(mux, huma.DefaultConfig("Visits API", "1.0.0"))
//
//	cqrsadapter.RegisterCommand(api, pipe,
//	    huma.Operation{OperationID: "create-visit", Method: "POST", Path: "/visits", DefaultStatus: 201},
//	    func(ctx context.Context, in *CreateVisitInput) (command.Command, error) {
//	        return newVisitCmd(in.Body.URL), nil
//	    },
//	    cqrsadapter.Authorize("visits", "create"),
//	    cqrsadapter.ValidateCommand(validateVisit),
//	)
//
// An anonymous caller answers 401, a policy denial 403, a validation failure
// 400, and a broken policy engine 503. The mapping is identical to the
// route-based pipeline because both go through httperr.MapError.
//
// # Loading the request identity
//
// The adapter reads the authenticated user from the request context, so mount
// cqrs-htmx's enrichment middleware (or your own) around the Huma handler:
//
//	mux := http.NewServeMux()
//	api := humago.New(mux, huma.DefaultConfig("Visits API", "1.0.0"))
//	// ... register operations ...
//	srv := app.Middleware()(mux) // ContextEnrichmentMiddleware
//
// A Huma handler receives the plain context.Context of that request, so user
// ID, actor, correlation and request ID values reach the pipeline.
//
// # What the adapter does not do
//
// It does not decode bodies (Huma does), it does not render HTML or HTMX
// headers (a JSON API has neither), and it does not run App's
// BeforeDispatch/AfterDispatch hooks, which take an *http.Request that a Huma
// handler never sees. Use Huma middleware for tracing and metrics.
package huma
