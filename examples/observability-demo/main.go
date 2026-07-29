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
	"time"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/middleware/v4"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v4"
	cqrsprom "github.com/larsartmann/go-cqrs-lite/prometheus/v4"
)

type pingCmd struct {
	id  id.CommandID
	sid id.StreamID
	Msg string `json:"msg"`
}

func (c *pingCmd) Type() command.Type     { return "Ping" }
func (c *pingCmd) StreamID() id.StreamID  { return c.sid }
func (c *pingCmd) ID() id.CommandID       { return c.id }

type pingRequest struct {
	Msg string `json:"msg"`
}

func main() {
	logger := slog.Default()
	ctx := context.Background()

	// --- 1. OTel tracing with stdout exporter ---
	otelProvider, err := cqrsotel.Setup(
		cqrsotel.WithService("observability-demo", "1.0.0", "local"),
		cqrsotel.WithStdoutExporter(os.Stdout),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "otel setup: %v\n", err)
		os.Exit(1)
	}
	defer otelProvider.Shutdown(ctx)

	tracer := cqrsotel.NewTracer("observability-demo")

	// --- 2. Prometheus metrics ---
	promProvider, err := cqrsprom.Setup(cqrsprom.WithViews(cqrsotel.NewCQRSViews()...))
	if err != nil {
		fmt.Fprintf(os.Stderr, "prometheus setup: %v\n", err)
		os.Exit(1)
	}
	defer promProvider.Shutdown(ctx)

	meter := cqrsotel.NewMeter("observability-demo")
	metricsRecorder, err := middleware.NewOTelMetricsRecorder(meter)
	if err != nil {
		fmt.Fprintf(os.Stderr, "metrics recorder: %v\n", err)
		os.Exit(1)
	}

	// --- 3. Command dispatcher with full middleware stack ---
	cmdDisp := command.NewDispatcher()
	cmdDisp.Use(middleware.CommandRecovery())
	cmdDisp.Use(middleware.CommandRetry(middleware.DefaultRetryConfig(), middleware.WithLogger(logger)))
	cmdDisp.Use(middleware.CommandTracing(tracer))
	cmdDisp.Use(middleware.CommandTypedMetrics(metricsRecorder))
	cmdDisp.Use(middleware.CommandLogging(logger))

	_ = command.RegisterTyped(cmdDisp, "Ping",
		func(_ context.Context, c *pingCmd) error {
			logger.Info("handling ping", "msg", c.Msg)
			return nil
		})

	app := cqrshtmx.MustNew(cqrshtmx.Config{Commands: cmdDisp})

	// --- 4. HTTP server ---
	mux := http.NewServeMux()
	mux.Handle("GET /htmx.js", cqrshtmx.HTMXScriptHandler())
	mux.Handle("GET /metrics", promProvider.Handler())
	mux.Handle("POST /ping", app.Command("Ping",
		cqrshtmx.DecodeJSON(func(r pingRequest) (command.Command, error) {
			return &pingCmd{id: id.NewCommandID(), sid: id.NewStreamID(), Msg: r.Msg}, nil
		}),
		cqrshtmx.WithSuccessStatus(http.StatusNoContent),
	))

	addr := ":8099"
	fmt.Printf("observability-demo on http://localhost%s\n", addr)
	fmt.Println("  POST /ping     — dispatch a command (traces + metrics emitted)")
	fmt.Println("  GET  /metrics  — Prometheus metrics endpoint")

	server := &http.Server{
		Addr:              addr,
		Handler:           cqrshtmx.Chain(cqrshtmx.RecoveryMiddleware, cqrshtmx.SecurityHeadersMiddleware)(mux),
		ReadHeaderTimeout: 5 * time.Second,
	}
	_ = server.ListenAndServe()
}
