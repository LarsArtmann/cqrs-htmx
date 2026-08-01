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
// Run: go run . and open http://localhost:8098
//
//	curl -X POST http://localhost:8098/ping -d '{"msg":"hello"}' -w '\n%{http_code}\n'
//
// You will see the logging middleware emit "dispatching"/"failed" lines for the two
// retried attempts, then the request completes with 204.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync/atomic"
	"time"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/middleware/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// pingCmd implements command.Command. The JSON body decodes directly into Msg.
type pingCmd struct {
	id  id.CommandID
	sid id.StreamID
	Msg string `json:"msg"`
}

func (c *pingCmd) Type() command.Type { return "Ping" }

func (c *pingCmd) StreamID() id.StreamID { return c.sid }

func (c *pingCmd) ID() id.CommandID { return c.id }

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

func newHandler() http.Handler {
	service := &flakyService{}

	cmdDisp := command.NewDispatcher()

	// --- THE KEY IDEA: wire go-cqrs-lite middleware onto the dispatcher. ---
	// Order is outer-to-inner (first registered wraps the rest). Recovery must be
	// outermost so panics never escape; retry sits inside recovery so it can
	// re-dispatch retryable errors; logging is innermost to log each attempt.
	cmdDisp.Use(middleware.CommandRecovery())
	cmdDisp.Use(middleware.CommandRetry(middleware.DefaultRetryConfig(), middleware.WithLogger(slog.Default())))
	cmdDisp.Use(middleware.CommandCircuitBreaker(middleware.DefaultCircuitBreakerConfig()))
	cmdDisp.Use(middleware.CommandLogging(slog.Default()))

	_ = command.RegisterTyped(cmdDisp, "Ping",
		func(_ context.Context, c *pingCmd) error {
			return service.ping(c.Msg)
		})

	app := cqrshtmx.MustNew(cqrshtmx.Config{Commands: cmdDisp})

	mux := http.NewServeMux()
	mux.Handle("GET /htmx.js", cqrshtmx.HTMXScriptHandler())
	mux.Handle("POST /ping", app.Command("Ping",
		cqrshtmx.DecodeJSON(func(r pingRequest) (command.Command, error) {
			return &pingCmd{id: id.NewCommandID(), sid: id.NewStreamID(), Msg: r.Msg}, nil
		}),
		cqrshtmx.WithSuccessStatus(http.StatusNoContent),
	))

	return cqrshtmx.Chain(cqrshtmx.RecoveryMiddleware, cqrshtmx.SecurityHeadersMiddleware)(mux)
}

func main() {
	handler := newHandler()

	addr := ":8098"
	fmt.Printf("middleware-demo on http://localhost%s/ping (POST {\"msg\":...})\n", addr)
	fmt.Println("First request retries twice (transient failures) then succeeds with 204.")

	server := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}
	_ = server.ListenAndServe()
}
