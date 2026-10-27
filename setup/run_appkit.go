// RunWithAppkit: the appkit-composed alternative server layer for Bundle (ADR-001 uplift).
package setup

import (
	"context"
	"net/http"
	"time"

	appkit "github.com/larsartmann/go-appkit"
	errorfamily "github.com/larsartmann/go-error-family"
)

// RunWithAppkit composes the bundle's full stack (domain middleware, mounting,
// health) onto a go-appkit Service instead of httputil.Server. Adopted from
// the ADR-001 uplift: the go-appkit require is a published tag, no replace
// directives.
//
// Verified equivalences with RunHandler (see run_appkit_test.go):
//
//   - timeout policy: Read/WriteTimeout disabled for SSE (appkit.NoTimeout),
//     ReadHeaderTimeout 5s / IdleTimeout 60s reaping pair kept, shutdown
//     budget 30s.
//   - SSE header flush survives appkit's middleware chain (outer) wrapping
//     the bundle's chain (inner).
//   - /health/ready keeps projection-aware semantics AND the full bundle
//     readiness: appkit's ReadyCheck runs every bundleReadinessCheck
//     (projection drain gate, shared-hub check, Config.HealthChecks) —
//     503 while any of them fails, 200 once all pass. appkit owns the
//     /health* routes (the bundle's HealthPath mount sits under the "/"
//     catch-all and is shadowed when the paths collide — set
//     Config.HealthPath to a distinct path to serve the rich JSON body
//     alongside).
//
// Deliberate differences (the appkit uplift ADR-001 wants):
//
//   - graceful drain: appkit flips readiness to 503, waits DrainDelay, then
//     stops accepting; RunHandler goes straight to Shutdown.
//   - appkit's generic middleware (Recovery, RequestID, Logging,
//     SecurityHeaders) wraps the bundle's domain chain.
//   - Addr() net.Addr is available while serving (listener captured).
func (b *Bundle) RunWithAppkit(ctx context.Context, addr string, handler http.Handler) error {
	return b.runWithAppkit(ctx, addr, handler, appkitDefaultDrainDelay, "")
}

// runWithAppkit is the shared worker for [Bundle.RunWithAppkit] and the
// integration tests + adoption benchmark. DrainDelay and LogLevel are parameters so
// internal callers can pick near-zero values:
//
//   - DrainDelay: production = appkitDefaultDrainDelay (2s); tests use
//     50ms so a 2s drain never bloats the suite runtime per case; the
//     adoption benchmark uses 10ms so b.N scales by request work alone, not
//     drain phase.
//   - LogLevel: production = "" (appkit default INFO + per-request access
//     logs); the adoption benchmark uses LogLevelError to isolate the
//     request-path work each stack performs from appkit's per-request log
//     line (comparison-report finding 7).
func (b *Bundle) runWithAppkit(
	ctx context.Context,
	addr string,
	handler http.Handler,
	drainDelay time.Duration,
	logLevel appkit.LogLevel,
) error {
	if handler == nil {
		mux := http.NewServeMux()
		b.Mount(mux)
		handler = b.Middleware()(mux)
	}

	//nolint:exhaustruct // optional appkit fields default sensibly inside appkit
	svc, err := appkit.NewService(appkit.ServiceConfig{
		Addr:              addr,
		LogLevel:          logLevel,
		ReadTimeout:       appkit.NoTimeout,
		ReadHeaderTimeout: runReadHeaderTimeout,
		WriteTimeout:      appkit.NoTimeout,
		IdleTimeout:       runIdleTimeout,
		ShutdownTimeout:   runShutdownTimeout,
		DrainDelay:        drainDelay,
		ReadyCheck:        b.bundleReadyCheck(),
		RegisterHealth:    nil, // default: appkit serves /health, /health/live, /health/ready (see doc above)
		Metrics:           b.config.Metrics,
		Version:           b.config.Version,
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
// the first behavioral uplift of RunHandler.
const appkitDefaultDrainDelay = 2 * time.Second

// bundleReadyCheck adapts the FULL bundle readiness composition onto appkit's
// boolean probe: ready only when every check passes (projections drained,
// hub healthy, consumer checks green). This is the same check set the
// mounted readiness handler serves — one composition, two surfaces.
func (b *Bundle) bundleReadyCheck() func() bool {
	return func() bool {
		for _, check := range b.bundleReadinessChecks() {
			if err := check.Check(); err != nil {
				return false
			}
		}

		return true
	}
}
