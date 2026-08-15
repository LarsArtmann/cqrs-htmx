// cqrs-lint:ignore(E009) spike: RunWithAppkit is an alternative composition root for the M18 spike (ADR-001 P3)
package setup

import (
	"context"
	"net/http"
	"time"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	appkit "github.com/larsartmann/go-appkit"
	errorfamily "github.com/larsartmann/go-error-family"
)

// SPIKE (ADR-001 P3 / plan task M18) — do not merge as-is.
//
// RunWithAppkit is RunHandler with the server layer swapped: instead of
// httputil.Server it drives an appkit.Service. Everything else — the bundle's
// own middleware chain, mounting, Close-on-every-exit — is identical. The
// require/replace for go-appkit in go.mod is spike-only and must be dropped
// (or pointed at a published tag) before this can merge.
//
// Verified equivalences with RunHandler (see run_appkit_test.go):
//
//   - timeout policy: Read/WriteTimeout disabled for SSE (appkit.NoTimeout),
//     ReadHeaderTimeout 5s / IdleTimeout 60s reaping pair kept, shutdown
//     budget 30s.
//   - SSE header flush survives appkit's middleware chain (outer) wrapping
//     the bundle's chain (inner).
//   - /health/ready keeps projection-aware semantics: appkit's ReadyCheck is
//     wired to cqrshtmx.ProjectionReadinessCheck (503 while any projection
//     drains or has failed, 200 once live/stopped).
//
// Deliberate differences (the appkit uplift ADR-001 wants):
//
//   - graceful drain: appkit flips readiness to 503, waits DrainDelay, then
//     stops accepting; RunHandler goes straight to Shutdown.
//   - appkit's generic middleware (Recovery, RequestID, Logging,
//     SecurityHeaders) wraps the bundle's domain chain.
//   - Addr() net.Addr is available while serving (listener captured).
func (b *Bundle) RunWithAppkit(ctx context.Context, addr string, handler http.Handler) error {
	if handler == nil {
		mux := http.NewServeMux()
		b.Mount(mux)
		handler = b.Middleware()(mux)
	}

	svc, err := appkit.NewService(appkit.ServiceConfig{
		Addr:              addr,
		ReadTimeout:       appkit.NoTimeout,
		ReadHeaderTimeout: runReadHeaderTimeout,
		WriteTimeout:      appkit.NoTimeout,
		IdleTimeout:       runIdleTimeout,
		ShutdownTimeout:   runShutdownTimeout,
		DrainDelay:        appkitDefaultDrainDelay,
		ReadyCheck:        b.projectionReadyCheck(),
		RegisterHealth:    nil, // default: appkit serves /health, /health/live, /health/ready
	})
	if err != nil {
		//cqrs-lint:ignore(C023) error path: the config rejection is primary; Close is secondary
		_ = b.Close()

		return errorfamily.WrapRejection(err, "setup.run_appkit", "invalid appkit configuration")
	}

	// The bundle's whole handler (its own middleware included) mounts under
	// appkit's catch-all: appkit's generic stack runs outside, the bundle's
	// domain chain inside. Health endpoints stay appkit's own so the drain
	// probe and ReadyCheck govern them.
	svc.Mux.Handle("/", handler)

	errChan, err := svc.Start()
	if err != nil {
		//cqrs-lint:ignore(C023,C015) error path: the start failure is primary; Close is secondary
		_ = b.Close()

		return errorfamily.WrapInfrastructure(err, "setup.run_appkit", "failed to start")
	}

	select {
	case err := <-errChan:
		//cqrs-lint:ignore(C023,C015) error path: the serve result is primary; Close is secondary
		_ = b.Close()

		if err != nil {
			return errorfamily.WrapInfrastructure(err, "setup.run_appkit", "server failed")
		}

		return nil
	case <-ctx.Done():
	}

	// Mirror RunHandler: Shutdown must run detached from the cancelled
	// context but keep its values and its own timeout budget. appkit's drain
	// sequence (readiness 503 → DrainDelay → stop) happens inside.
	if err := svc.Shutdown(context.WithoutCancel(ctx)); err != nil {
		//cqrs-lint:ignore(C023) error path: the shutdown failure is primary; Close is secondary
		_ = b.Close()

		return errorfamily.WrapInfrastructure(err, "setup.run_appkit", "graceful shutdown failed")
	}

	return b.Close()
}

// appkitDefaultDrainDelay gives load balancers a beat to observe the readiness
// flip before connections close. RunHandler has no drain phase at all; this is
// the first behavioral uplift of the spike.
const appkitDefaultDrainDelay = 2 * time.Second

// projectionReadyCheck adapts the bundle's projection-aware readiness onto
// appkit's boolean probe: ready when the check passes (nil error).
func (b *Bundle) projectionReadyCheck() func() bool {
	return func() bool {
		if b.Service == nil {
			return true
		}

		return cqrshtmx.ProjectionReadinessCheck(b.Service).Check() == nil
	}
}
