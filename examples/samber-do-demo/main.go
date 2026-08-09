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
// Run: go run . and open http://localhost:8098/
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/httputil"
)

func main() {
	// 1. Create the DI container. The returned cleanup function MUST be
	// deferred — it calls injector.Shutdown(), which cascades to every
	// service implementing do.Shutdowner* (including usermgmt.Service.Close()).
	container, cleanup := NewContainer(AppConfig{
		Addr:       ":8098",
		TOTPIssuer: "cqrs-htmx samber/do Demo",
	})
	defer cleanup()

	// 2. Resolve the cqrshtmx.App — lazy singleton, constructed on first
	// invocation. No do.MustInvoke here: we handle the error gracefully.
	app, err := container.App()
	if err != nil {
		log.Fatalf("resolve App: %v", err)
	}

	// 3. Resolve the usermgmt.Service — also lazy.
	svc, err := container.Service()
	if err != nil {
		log.Fatalf("resolve Service: %v", err)
	}

	// Seed a demo user so the service has data.
	seed(context.Background(), svc)

	// 4. Wire routes. The app produces HTTP handlers via app.Command() /
	// app.Query(). This demo mounts a simple page + health check.
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", indexHandler)
	mux.Handle("GET /htmx.js", cqrshtmx.HTMXScriptHandler())
	mux.HandleFunc("GET /health", healthHandler)
	_ = app // app.Command(...) / app.Query(...) would go here in a real app

	logger, _ := container.Logger()
	logger.Info("samber-do-demo starting", "addr", ":8098", "hint", "open http://localhost:8098/")

	srv, err := httputil.NewServer(httputil.ServerConfig{Addr: ":8098"}, mux)
	if err != nil {
		log.Fatalf("NewServer: %v", err)
	}

	if err := <-srv.Start(); err != nil {
		log.Fatal(err)
	}
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
		<li><code>usermgmt.TOTPProvider</code> — named service <code>"auth.totp"</code></li>
		<li><code>serviceLifecycle</code> — ShutdownerWithContextAndError adapter</li>
	</ul>
	<p>Visit <a href="/health"><code>/health</code></a> for a health check.</p>
</body>
</html>`)
}

// healthHandler is a simple readiness probe.
func healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = fmt.Fprintf(w, `{"status":"ok"}`)
}
