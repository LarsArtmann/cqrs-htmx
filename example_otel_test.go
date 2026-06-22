package cqrshtmx_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v3"
	command "github.com/larsartmann/go-cqrs-lite/command/v3"
)

// This file demonstrates the recommended pattern for integrating OpenTelemetry
// (or any tracing/metrics system) into cqrs-htmx applications via the
// BeforeDispatchHook / AfterDispatchHook lifecycle hooks.
//
// Why hooks and not a middleware?
//   - Hooks run INSIDE the dispatch path, so the span can include the actual
//     command/query type and dispatch result. Middleware outside the dispatch
//     can only see the HTTP method/path — it cannot distinguish a CreateUser
//     command from an UpdateUser command.
//   - The hook context round-trip lets you carry a span across the dispatch
//     boundary and into the command/query handler via the standard
//     `trace.SpanFromContext(ctx)` accessor.
//
// cqrs-htmx intentionally does NOT import go.opentelemetry.io/otel directly.
// The library principle is: never enforce a tracing/metrics dependency.
// Consumers bring their own OTel SDK and pass hooks into Config.

// --- The OTel types we use are local stubs for this example. ---
// In a real project, replace these with the SDK types:
//
//	import "go.opentelemetry.io/otel"
//	import "go.opentelemetry.io/otel/attribute"
//	import "go.opentelemetry.io/otel/codes"
//	import "go.opentelemetry.io/otel/trace"
//
// The shape of the API is identical (Tracer.Start, Span.End, Span.SetStatus,
// Span.RecordError, Span.SetAttributes).

// otelSpan is the subset of otel.Span used by the example.
type otelSpan interface{ End(err error) }

// otelTracer is a stand-in for otel.Tracer.
type otelTracer interface {
	Start(ctx context.Context, name string) (context.Context, otelSpan)
}

type otelCtxKey struct{}

type otelSpanCtx struct{ span otelSpan }

// OtelBeforeDispatch returns a BeforeDispatchHook that starts a new span for
// each dispatch. The span is stored in the context, so downstream handlers
// and repositories can attach attributes via otel's normal context propagation.
//
// Real OTel version:
//
//	import (
//	    "go.opentelemetry.io/otel"
//	    "go.opentelemetry.io/otel/attribute"
//	    "go.opentelemetry.io/otel/trace"
//	)
//
//	func OtelBeforeDispatch() cqrshtmx.BeforeDispatchHook {
//	    tr := otel.Tracer("cqrs-htmx")
//	    return func(ctx context.Context, r *http.Request) context.Context {
//	        name := dispatchTypeFromRequest(r) // your helper
//	        ctx, _ = tr.Start(ctx, "cqrs.dispatch "+name)
//	        return ctx
//	    }
//	}
func OtelBeforeDispatch(tr otelTracer) cqrshtmx.BeforeDispatchHook {
	return func(ctx context.Context, _ *http.Request) context.Context {
		ctx, span := tr.Start(ctx, "cqrs.dispatch")
		_ = span
		return ctx
	}
}

// OtelAfterDispatch returns an AfterDispatchHook that ends the span started
// by OtelBeforeDispatch. It records the dispatch error (if any) and finishes
// the span, exporting it via the configured OTel exporter.
//
// Real OTel version:
//
//	import "go.opentelemetry.io/otel/codes"
//	import "go.opentelemetry.io/otel/trace"
//
//	func OtelAfterDispatch() cqrshtmx.AfterDispatchHook {
//	    return func(ctx context.Context, _ *http.Request, err error) {
//	        span := trace.SpanFromContext(ctx)
//	        if err != nil {
//	            span.RecordError(err)
//	            span.SetStatus(codes.Error, err.Error())
//	        }
//	        span.End()
//	    }
//	}
func OtelAfterDispatch() cqrshtmx.AfterDispatchHook {
	return func(ctx context.Context, _ *http.Request, err error) {
		if sc, ok := ctx.Value(otelCtxKey{}).(otelSpanCtx); ok {
			sc.span.End(err)
		}
	}
}

// fakeTracer is a minimal test double to keep this example self-contained.
type fakeTracer struct{ spans []string }

func (f *fakeTracer) Start(_ context.Context, name string) (context.Context, otelSpan) {
	f.spans = append(f.spans, name)
	span := &fakeSpan{name: name, tr: f}
	return context.WithValue(context.Background(), otelCtxKey{}, otelSpanCtx{span: span}), span
}

type fakeSpan struct {
	name string
	tr   *fakeTracer
	err  error
}

func (s *fakeSpan) End(err error) {
	s.err = err
	s.tr.spans = append(s.tr.spans, "end:"+s.name)
}

// ExampleOtelBeforeDispatch shows wiring the OTel-pattern hooks into a
// cqrs-htmx App. In a real project, replace `fakeTracer` with the OTel SDK
// tracer obtained from otel.Tracer("cqrs-htmx") and the example will
// start/stop real spans for every dispatch.
func ExampleOtelBeforeDispatch() {
	tr := &fakeTracer{}

	disp := command.NewDispatcher()
	// Real apps register commands via disp.Register("create_item", handler).
	// This example exercises the hook wiring; no command registration is
	// required for the hooks to run.

	app, _ := cqrshtmx.New(cqrshtmx.Config{
		Commands:       disp,
		BeforeDispatch: OtelBeforeDispatch(tr),
		AfterDispatch:  OtelAfterDispatch(),
	})

	mux := http.NewServeMux()
	mux.Handle("/items", app.Command("create_item"))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/items", nil)
	mux.ServeHTTP(w, r)

	// The fake tracer recorded at least one span for inspection.
	fmt.Println(len(tr.spans) > 0)
	// Output: true
}
