// Package main demonstrates wiring go-cqrs-lite's middleware module into a
// cqrs-htmx command dispatcher.
//
// cqrs-htmx's [command.Dispatcher] (passed to [cqrshtmx.Config.Commands]) is the
// exact same type go-cqrs-lite's middleware targets, so any of the 27 middleware
// factories (logging, recovery, retry, circuit breaker, metrics, tracing,
// idempotency, validation) compose with zero glue — just call `dispatcher.Use(...)`
// before building the App.
//
// This example mounts a single command whose handler fails transiently twice and
// then succeeds. The retry middleware makes the HTTP request still return 204.
//
// Tracing + metrics run for real here: cqrsotel.Setup registers the global
// tracer/meter providers (pretty-printing every span to stdout), and the
// CommandTracing + CommandTypedMetrics factories emit per-dispatch spans and
// typed metrics; cqrsprom.Setup serves them at /metrics. This single
// Setup call also activates go-cqrs-lite's built-in decider/store spans
// ("free domain spans") — see guide §2.4 in
// docs/guides/leveraging-go-cqrs-lite.md. The full end-to-end story (HTTP
// root spans via otelhttp) lives in examples/observability-demo.
//
// Run: go run . and open http://localhost:8098
//
//	curl -X POST http://localhost:8098/ping -d '{"msg":"hello"}' -w '\n%{http_code}\n'
//	curl http://localhost:8098/metrics | grep cqrs
//
// You will see the logging middleware emit "dispatching"/"failed" lines for the two
// retried attempts, the tracing middleware print the dispatch spans, then the
// request completes with 204.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sync/atomic"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/middleware/v4"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v4"
	cqrsprom "github.com/larsartmann/go-cqrs-lite/prometheus/v4"
	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/larsartmann/httputil"
	"go.opentelemetry.io/otel"
)

// pingCmd embeds *command.BasicCommand for Type()/StreamID()/ID().
// The JSON body decodes directly into Msg.
type pingCmd struct {
	*command.BasicCommand
	Msg string `json:"msg"`
}

type pingRequest struct {
	Msg string `json:"msg"`
}

// flakyService simulates a downstream that fails transiently the first two calls
// and then recovers. A Transient error is classified as retryable by
// errorfamily.IsRetryable, so the retry middleware will re-attempt it.
type flakyService struct {
	calls atomic.Int32
}

func (s *flakyService) ping(msg string) error {
	n := s.calls.Add(1)
	if n < 3 {
		// Retryable: the middleware's IsRetryable (errorfamily.IsRetryable)
		// recognises the Transient family and re-dispatches.
		return errorfamily.NewTransient(
			"demo.downstream_busy",
			fmt.Sprintf("downstream rejected %q on attempt %d", msg, n),
		)
	}

	return nil
}

// newHandler builds the full HTTP handler with live OTel tracing (stdout
// spans) and Prometheus metrics wired through dispatch middleware. Returns
// the handler and both providers for graceful shutdown.
func newHandler(logger *slog.Logger) (http.Handler, *cqrsprom.Provider, *cqrsotel.Provider, error) {
	// Registers the global tracer + meter providers AND the W3C propagator.
	// Side effect that matters beyond this demo: go-cqrs-lite's decider and
	// storage layers resolve their spans against the global tracer, so this
	// one call turns on the "free domain spans" for any wired CQRS stack
	// (guide §2.4).
	otelProvider, err := cqrsotel.Setup(
		cqrsotel.WithService("middleware-demo", "1.0.0", "local"),
		cqrsotel.WithStdoutExporter(os.Stdout),
	)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("otel setup: %w", err)
	}

	promProvider, err := cqrsprom.Setup(cqrsprom.WithViews(cqrsotel.NewCQRSViews()...))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("prometheus setup: %w", err)
	}

	// Route the CQRS instruments to Prometheus: cqrsotel.Setup registered a
	// meter provider without a reader; the Prometheus provider has one.
	otel.SetMeterProvider(promProvider.AsMeterProvider())

	tracer := cqrsotel.NewTracer("middleware-demo")
	recorder, err := middleware.NewOTelMetricsRecorder(cqrsotel.NewMeter("middleware-demo"))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("metrics recorder: %w", err)
	}

	service := &flakyService{}

	cmdDisp := command.NewDispatcher()

	// --- THE KEY IDEA: wire go-cqrs-lite middleware onto the dispatcher. ---
	// Order is outer-to-inner (first registered wraps the rest). Recovery must be
	// outermost so panics never escape; retry sits inside recovery so it can
	// re-dispatch retryable errors; tracing and metrics wrap the handler so each
	// attempt is observed; logging is innermost to log each attempt.
	cmdDisp.Use(middleware.CommandRecovery())
	cmdDisp.Use(middleware.CommandRetry(middleware.DefaultRetryConfig(), middleware.WithLogger(logger)))
	cmdDisp.Use(middleware.CommandCircuitBreaker(middleware.DefaultCircuitBreakerConfig()))
	cmdDisp.Use(middleware.CommandTracing(tracer))
	cmdDisp.Use(middleware.CommandTypedMetrics(recorder))
	cmdDisp.Use(middleware.CommandLogging(logger))

	//cqrs-lint:ignore(C028) example: error handling omitted for brevity
	_ = command.RegisterTyped(cmdDisp, "Ping",
		func(_ context.Context, c *pingCmd) error {
			return service.ping(c.Msg)
		})

	app := cqrshtmx.MustNew(cqrshtmx.Config{Commands: cmdDisp})

	mux := http.NewServeMux()
	mux.Handle("GET /htmx.js", cqrshtmx.HTMXScriptHandler())
	mux.Handle("GET /metrics", promProvider.Handler())
	mux.Handle("POST /ping", app.Command("Ping",
		cqrshtmx.DecodeJSON(func(r pingRequest) (command.Command, error) {
			core, err := command.New("Ping", id.NewStreamID())
			if err != nil {
				return nil, err
			}
			return &pingCmd{BasicCommand: core, Msg: r.Msg}, nil
		}),
		cqrshtmx.WithSuccessStatus(http.StatusNoContent),
	))

	handler := cqrshtmx.Chain(
		cqrshtmx.RecoveryMiddleware,
		httputil.SecurityHeaders(httputil.DefaultSecurityHeadersConfig()),
	)(
		mux,
	)

	return handler, promProvider, otelProvider, nil
}

func main() {
	handler, promProvider, otelProvider, err := newHandler(slog.Default())
	if err != nil {
		fmt.Fprintf(os.Stderr, "setup: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = otelProvider.Shutdown(context.Background()) }()
	defer func() { _ = promProvider.Shutdown(context.Background()) }()

	addr := ":8098"
	fmt.Printf("middleware-demo on http://localhost%s/ping (POST {\"msg\":...})\n", addr)
	fmt.Println("First request retries twice (transient failures) then succeeds with 204.")
	fmt.Println("  GET /metrics — Prometheus metrics (cqrs.operation.duration, cqrs.operation.count)")

	srv, err := httputil.NewServer(httputil.ServerConfig{Addr: addr}, handler)
	if err != nil {
		fmt.Printf("NewServer: %v\n", err)
		return
	}
	if err := <-srv.Start(); err != nil {
		fmt.Printf("server: %v\n", err)
	}
}
