package setup

import (
	"context"
	"net/http"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/larsartmann/httputil"
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
	mux := http.NewServeMux()
	b.Mount(mux)

	server, err := httputil.NewServer(httputil.ServerConfig{ //nolint:exhaustruct // SSE-safe subset, see doc comment
		Addr:              addr,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
		ShutdownTimeout:   30 * time.Second,
	}, b.Middleware()(mux))
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

	if err := server.Shutdown(context.Background()); err != nil {
		_ = b.Close()

		return errorfamily.WrapInfrastructure(err, "setup.run", "graceful shutdown failed")
	}

	return b.Close()
}
