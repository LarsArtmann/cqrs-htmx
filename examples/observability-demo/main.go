// Package main demonstrates OpenTelemetry tracing and Prometheus metrics
// wiring using go-cqrs-lite's otel and prometheus modules with cqrs-htmx.
//
// The demo sets up:
//   - OTel tracing with a stdout exporter (pretty-printed spans to console)
//   - Prometheus metrics endpoint at /metrics
//   - Dispatch middleware: recovery, retry, tracing, metrics, logging
//
// Run: go run . and open http://localhost:8099
//
//	POST http://localhost:8099/ping -d '{"msg":"hello"}'   # dispatch a command
//	GET  http://localhost:8099/metrics                     # Prometheus metrics
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/middleware/v4"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v4"
	cqrsprom "github.com/larsartmann/go-cqrs-lite/prometheus/v4"
	"github.com/larsartmann/httputil"
	"go.opentelemetry.io/otel"
)

type pingCmd struct {
	id  id.CommandID
	sid id.StreamID
	Msg string `json:"msg"`
}

func (c *pingCmd) Type() command.Type    { return "Ping" }
func (c *pingCmd) StreamID() id.StreamID { return c.sid }
func (c *pingCmd) ID() id.CommandID      { return c.id }

type pingRequest struct {
	Msg string `json:"msg"`
}

func main() {
	logger := slog.Default()

	handler, promProvider, otelProvider, err := newHandler(logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "setup: %v\n", err)
		os.Exit(1)
	}
	defer otelProvider.Shutdown(context.Background())
	defer promProvider.Shutdown(context.Background())

	addr := ":8099"
	fmt.Printf("observability-demo on http://localhost%s\n", addr)
	fmt.Println("  POST /ping     — dispatch a command (traces + metrics emitted)")
	fmt.Println("  GET  /metrics  — Prometheus metrics endpoint")

	srv, err := httputil.NewServer(httputil.ServerConfig{Addr: addr}, handler)
	if err != nil {
		fmt.Fprintf(os.Stderr, "NewServer: %v\n", err)
		os.Exit(1)
	}
	if err := <-srv.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "server: %v\n", err)
		os.Exit(1)
	}
}

// newHandler builds the full HTTP handler with OTel tracing + Prometheus metrics
// wired through dispatch middleware. Returns the handler and both providers for
// graceful shutdown.
func newHandler(logger *slog.Logger) (http.Handler, *cqrsprom.Provider, *cqrsotel.Provider, error) {
	otelProvider, err := cqrsotel.Setup(
		cqrsotel.WithService("observability-demo", "1.0.0", "local"),
		cqrsotel.WithStdoutExporter(os.Stdout),
	)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("otel setup: %w", err)
	}

	tracer := cqrsotel.NewTracer("observability-demo")

	promProvider, err := cqrsprom.Setup(cqrsprom.WithViews(cqrsotel.NewCQRSViews()...))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("prometheus setup: %w", err)
	}

	// Override the global meter provider so cqrsotel.NewMeter instruments
	// export to Prometheus. cqrsotel.Setup registered its own meter provider
	// globally, but it has no reader — the Prometheus provider has one.
	otel.SetMeterProvider(promProvider.AsMeterProvider())

	meter := cqrsotel.NewMeter("observability-demo")
	metricsRecorder, err := middleware.NewOTelMetricsRecorder(meter)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("metrics recorder: %w", err)
	}

	cmdDisp := command.NewDispatcher()
	cmdDisp.Use(middleware.CommandRecovery())
	cmdDisp.Use(middleware.CommandRetry(middleware.DefaultRetryConfig(), middleware.WithLogger(logger)))
	cmdDisp.Use(middleware.CommandTracing(tracer))
	cmdDisp.Use(middleware.CommandTypedMetrics(metricsRecorder))
	cmdDisp.Use(middleware.CommandLogging(logger))

	//cqrs-lint:ignore(C028) example: error handling omitted for brevity
	_ = command.RegisterTyped(cmdDisp, "Ping",
		func(_ context.Context, c *pingCmd) error {
			logger.Info("handling ping", "msg", c.Msg)
			return nil
		})

	app := cqrshtmx.MustNew(cqrshtmx.Config{Commands: cmdDisp})

	mux := http.NewServeMux()
	mux.Handle("GET /htmx.js", cqrshtmx.HTMXScriptHandler())
	mux.Handle("GET /metrics", promProvider.Handler())
	mux.Handle("POST /ping", app.Command("Ping",
		cqrshtmx.DecodeJSON(func(r pingRequest) (command.Command, error) {
			return &pingCmd{id: id.NewCommandID(), sid: id.NewStreamID(), Msg: r.Msg}, nil
		}),
		cqrshtmx.WithSuccessStatus(http.StatusNoContent),
	))

	handler := cqrshtmx.Chain(cqrshtmx.RecoveryMiddleware, cqrshtmx.SecurityHeadersMiddleware)(mux)
	return handler, promProvider, otelProvider, nil
}
