// Command samber-do-demo is a runnable showcase of wiring cqrs-htmx with the
// samber/do v2 dependency-injection container.
//
// It demonstrates every canonical samber/do pattern applied to cqrs-htmx:
//   - Composition root with cleanup function (NewContainer)
//   - Eager foundation values (AppConfig via ProvideValue)
//   - Lazy singletons (*usermgmt.Service, *cqrshtmx.App via Provide)
//   - Named services (TOTP auth provider via ProvideNamed)
//   - Lifecycle adapter (serviceLifecycle implements ShutdownerWithContextAndError)
//   - Typed accessors (Container.Service(), Container.App(), etc.)
//
// Run: `go run .` and open http://localhost:8098/ (override the port with
// `PORT=9090 go run .`).
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	gohealth "github.com/larsartmann/go-health"
	"github.com/larsartmann/httputil"
)

// demoAddr resolves the listen address: the PORT environment variable when
// set (digits only, no colon), else :8098. The override exists so tests and
// containers can run the demo on any free port.
func demoAddr() string {
	if port := os.Getenv("PORT"); port != "" {
		if _, err := strconv.Atoi(port); err == nil {
			return ":" + port
		}
	}

	return ":8098"
}

func main() {
	addr := demoAddr()

	// 1. Create the DI container. The returned cleanup function MUST be
	// deferred — it calls injector.Shutdown(), which cascades to every
	// service implementing do.Shutdowner* (including usermgmt.Service.Close()).
	container, cleanup, err := NewContainer(AppConfig{
		Addr:       addr,
		TOTPIssuer: "cqrs-htmx samber/do Demo",
	})
	if err != nil {
		log.Fatalf("build DI container: %v", err)
	}
	defer cleanup()

	// 2. Resolve the usermgmt.Service — lazy — and seed a demo user so the
	// service has data.
	svc, err := container.Service()
	if err != nil {
		log.Fatalf("resolve Service: %v", err)
	}
	seed(context.Background(), svc)

	// 3. Wire routes (buildRouter owns the full mux — tests exercise the
	// same function, so route conflicts panic in tests, not at boot).
	mux, err := buildRouter(container)
	if err != nil {
		log.Fatalf("build router: %v", err)
	}

	logger, _ := container.Logger()
	logger.Info("samber-do-demo starting", "addr", addr, "hint", "open http://localhost"+addr+"/")

	srv, err := httputil.NewServer(httputil.ServerConfig{Addr: addr}, mux)
	if err != nil {
		log.Fatalf("NewServer: %v", err)
	}

	if err := <-srv.Start(); err != nil {
		log.Fatal(err)
	}
}

// buildRouter mounts every route the demo serves. main and the boot tests
// share this function: a ServeMux pattern conflict (the Go 1.22+ class that
// once panicked this demo at boot) fails HERE, in tests, not in main.
func buildRouter(container *Container) (*http.ServeMux, error) {
	app, err := container.App()
	if err != nil {
		return nil, fmt.Errorf("resolve App: %w", err)
	}

	mux := http.NewServeMux()
	// Method-less root: go-health's RegisterRoutes mounts method-less /healthz,
	// /readyz, /startupz patterns, and a method-scoped "GET /" conflicts with
	// any of them under Go 1.22+ ServeMux rules (incomparable specificity).
	// A method-less "/" is strictly more general than every other pattern —
	// no conflict, and the index answers any method as a plain info page.
	mux.HandleFunc("/", indexHandler)
	mux.Handle("GET /htmx.js", cqrshtmx.HTMXScriptHandler())
	// Real Kubernetes health probes from the go-health surface NewContainer
	// started: /healthz (liveness — dependency-blind, always 200), /readyz
	// (readiness — 503 while projections drain or a critical projection
	// fails), /startupz (startup latch). No hand-rolled static handler: a
	// health endpoint that answers 200 without checking anything is the
	// false-green class this demo exists to avoid.
	probe, err := container.Probe()
	if err != nil {
		return nil, fmt.Errorf("resolve health probe: %w", err)
	}
	probe.RegisterRoutes(mux, gohealth.DefaultRoutes())
	// Live audit-log viewer from the auditlog/v4 bridge (plugin recorded every
	// service invocation in the container; HTML UI + JSON API + SSE stream).
	// GET-scoped and mounted WITHOUT StripPrefix: the viewer serves its own
	// configured prefix (Prefix: "/audit" routes /audit/, /audit/api/...
	// internally), and a method-less "/audit/" pattern would conflict with
	// "GET /" under Go 1.22+ ServeMux rules (the demo used to panic at boot
	// over exactly this).
	mux.Handle("GET /audit/", container.AuditViewer)
	// Projection health dashboard from the health/v4 bridge (one check per
	// projection worker of the usermgmt.Service; live via SSE).
	healthDashboard, err := container.HealthDashboard()
	if err != nil {
		return nil, fmt.Errorf("resolve HealthDashboard: %w", err)
	}
	mux.Handle("GET /health-ui", healthDashboard.Handler())
	mux.Handle("POST /command/hello", app.Command(
		"Hello",
		cqrshtmx.DecodeJSON(func(req helloRequest) (command.Command, error) {
			core, err := command.New("Hello", id.NewStreamID())
			if err != nil {
				return nil, err
			}
			return &helloCmd{BasicCommand: core, Name: req.Name}, nil
		}),
	))

	return mux, nil
}

// indexHandler serves a simple HTML page demonstrating the app is wired.
func indexHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = fmt.Fprint(w, `<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="utf-8">
	<meta name="viewport" content="width=device-width, initial-scale=1">
	<title>cqrs-htmx + samber/do Demo</title>
	<script src="/htmx.js"></script>
</head>
<body style="font-family: system-ui, sans-serif; max-width: 720px; margin: 4rem auto; padding: 0 1rem">
	<h1>cqrs-htmx + samber/do v2</h1>
	<p>This server is wired entirely through a samber/do dependency-injection container.</p>
	<ul>
		<li><code>*usermgmt.Service</code> — lazy singleton (event-sourced identity)</li>
		<li><code>*cqrshtmx.App</code> — lazy singleton (HTMX handler factory)</li>
		<li><code>*cqrshtmx.Broadcaster</code> — lazy singleton (SSE live updates)</li>
		<li><code>identitymodel.TOTPProvider</code> — interface service via <code>do.As</code> + <code>InvokeAs</code></li>
		<li><code>serviceLifecycle</code> — ShutdownerWithContextAndError adapter</li>
		<li><code>auditlog.WithAuditLog</code> — live audit viewer at <code>/audit/</code></li>
		<li><code>health.NewProbe</code> + <code>NewDashboard</code> — real health probes at <code>/healthz</code> <code>/readyz</code> <code>/startupz</code>, UI at <code>/health-ui</code></li>
	</ul>
	<p>Visit <a href="/readyz"><code>/readyz</code></a> for readiness, <a href="/healthz"><code>/healthz</code></a> for liveness, <a href="/audit/"><code>/audit/</code></a> for the live audit log, or <a href="/health-ui"><code>/health-ui</code></a> for the projection health dashboard.</p>
</body>
</html>`)
}
