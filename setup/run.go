package setup

import (
	"context"
	"net/http"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/larsartmann/httputil"
)

// Server timeouts for Run/RunHandler. ReadHeaderTimeout and IdleTimeout form
// the slowloris/keep-alive reaping pair; Read/Write timeouts are deliberately
// absent because SSE streams outlive any fixed deadline (see Run).
const (
	runReadHeaderTimeout = 5 * time.Second
	runIdleTimeout       = 60 * time.Second
	runShutdownTimeout   = 30 * time.Second
)

// Run is the shortest path from [New] to a serving production-shaped HTTP
// server. It mounts all routes, applies [Bundle.Middleware], serves on addr,
// and blocks until ctx is cancelled (or the server fails), then shuts down
// gracefully and calls [Bundle.Close]:
//
//	bundle, err := setup.New(setup.Config{Title: "My App"})
//	if err != nil {
//		log.Fatal(err)
//	}
//	if err := bundle.Run(ctx, ":8080"); err != nil {
//		log.Fatal(err)
//	}
//
// To add your own routes next to the bundle's, use [Bundle.RunHandler] with
// [Bundle.Handler].
//
// The server sets ReadHeaderTimeout (slowloris mitigation) and IdleTimeout
// (idle keep-alive reaping). ReadTimeout and WriteTimeout stay disabled on
// purpose: the bundle serves SSE streams (dashboard live updates) that outlive
// any fixed deadline — a WriteTimeout would kill them mid-stream. Consumers
// who need stricter limits should call [Bundle.Mount] and [Bundle.Middleware]
// with their own [httputil.Server] instead.
//
// Run returns nil after a clean shutdown, and a wrapped error if the server
// fails to start or the graceful shutdown times out. [Bundle.Close] is called
// on every exit path, exactly once.
func (b *Bundle) Run(ctx context.Context, addr string) error {
	return b.RunHandler(ctx, addr, nil)
}

// RunHandler is [Bundle.Run] for a handler you compose yourself. Pass
// [Bundle.Handler] (mux) to serve the bundle's routes next to your own:
//
//	mux := http.NewServeMux()
//	mux.HandleFunc("POST /orders", ordersHandler)
//	err := bundle.RunHandler(ctx, ":8080", bundle.Handler(mux))
//
// The handler is served as-is (no extra middleware is applied); wrap it with
// [Bundle.Middleware] yourself if you build it manually.
func (b *Bundle) RunHandler(ctx context.Context, addr string, handler http.Handler) error {
	if handler == nil {
		mux := http.NewServeMux()
		b.Mount(mux)
		handler = b.Middleware()(mux)
	}

	server, err := httputil.NewServer(httputil.ServerConfig{ //nolint:exhaustruct // SSE-safe subset, see Run doc
		Addr:              addr,
		ReadHeaderTimeout: runReadHeaderTimeout,
		IdleTimeout:       runIdleTimeout,
		ShutdownTimeout:   runShutdownTimeout,
	}, handler)
	if err != nil {
		_ = b.Close()

		return errorfamily.WrapRejection(err, "setup.run", "invalid server configuration")
	}

	errChan := server.Start()

	select {
	case err := <-errChan:
		// Start() only forwards real errors (http.ErrServerClosed is filtered).
		_ = b.Close()

		if err != nil {
			return errorfamily.WrapInfrastructure(err, "setup.run", "server failed")
		}

		return nil
	case <-ctx.Done():
	}

	// ctx is already cancelled at this point; detach the cancellation but keep
	// the values so Shutdown runs with its configured timeout budget.
	if err := server.Shutdown(context.WithoutCancel(ctx)); err != nil {
		_ = b.Close()

		return errorfamily.WrapInfrastructure(err, "setup.run", "graceful shutdown failed")
	}

	return b.Close()
}
