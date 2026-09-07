// Command setup-demo is a runnable showcase of the setup one-call composition
// root: the whole full-stack app (auth API, login page, admin panel, CQRS
// observability dashboard, health endpoint) from a single setup.New call.
//
// Open http://localhost:8099/ and you are signed in as the demo admin
// automatically. This is a demo only — in-memory storage, dev-only login
// shortcut. Real applications configure a WebAuthn/TOTP/OAuth2 provider so
// users sign in through the login page, and back everything with a persistent
// event store.
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	ds "github.com/larsartmann/cqrs-htmx/datastar/v4"
	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/cqrs-htmx/setup/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	"github.com/larsartmann/go-sse"
)

const (
	addr         = ":8099"
	cookieName   = "session"
	adminEmail   = "admin@demo.dev"
	adminUserID  = "01JXSETUPDEMO01"
	dayInSeconds = 86400
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	// Cancel the context on SIGINT/SIGTERM for a graceful drain.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 1. One call, whole app: event-sourced user management + auth API +
	//    login page + admin panel + CQRS dashboard + /health + shared SSE,
	//    with the documented middleware ordering applied per panel.
	//
	//    ServiceConfig is the escape hatch: service knobs the flattened
	//    fields cannot express (here MaxUsers) flow straight into
	//    usermgmt.NewService. Precedence: Service > ServiceConfig > flattened.
	bundle, err := setup.New(setup.Config{ //nolint:exhaustruct // demo uses in-memory defaults
		Title:     "cqrs-htmx Setup Demo",
		LogoutURL: "/dev-logout",
		ServiceConfig: &usermgmt.ServiceConfig{
			MaxUsers: 50,
		},
		SSEPath: "/sse",
		SSEURL:  "/sse",
		// DataStar feed on the SAME hub as /sse: one broadcast reaches both
		// transports (ADR-0050). The SDK script auto-mounts at /datastar.js.
		DataStarPath: "/ds/events",
	})
	if err != nil {
		return fmt.Errorf("setup.New: %w", err)
	}

	// 2. Seed a super_admin so the admin panel's default role-based
	//    authorizer admits the demo user (exercises the real authz path).
	token := seed(ctx, bundle)

	// 3. Compose a mux with one extra demo route next to the bundle's own
	//    routes, then serve with RunHandler: safe timeouts, graceful
	//    shutdown, and bundle cleanup in one call.
	mux := http.NewServeMux()
	mux.HandleFunc("/dev-login", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{ //nolint:exhaustruct,gosec // dev-only demo cookie
			Name: cookieName, Value: token, Path: "/",
			HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: dayInSeconds,
		})
		http.Redirect(w, r, "/admin/", http.StatusSeeOther)
	})
	mux.HandleFunc("/dev-logout", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{ //nolint:exhaustruct // dev-only
			Name: cookieName, Value: "", Path: "/", MaxAge: -1,
		})
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})
	// Note: "/" is NOT registered here — the bundle's login page owns the
	// site root (registering a second "/" would panic the mux).

	// POST /broadcast pushes ONE action through the bundle's shared fan-out
	// hub twice — once as a raw SSE event (HTMX clients: the admin panel's
	// sync indicator, /sse listeners) and once as a DataStar signal patch
	// (/ds/events clients). Both land on the same hub, which is the whole
	// point of the dual-transport setup.
	broadcastCount := 0
	mux.HandleFunc("POST /broadcast", func(w http.ResponseWriter, _ *http.Request) {
		broadcastCount++

		bundle.Broadcaster.Broadcast(sse.Event{
			Event: "demoBroadcast",
			Data:  `{"message":"hello from setup-demo"}`,
		})

		if bundle.DataStarBroadcaster != nil {
			patch, err := ds.SignalsPatch(map[string]any{"broadcasts": broadcastCount})
			if err == nil {
				bundle.DataStarBroadcaster.Broadcast(patch)
			}
		}

		w.WriteHeader(http.StatusAccepted)
	})

	// GET /ds-demo is a minimal DataStar client page: it loads the SDK from
	// the bundle's script mount and renders the live "broadcasts" signal
	// that POST /broadcast patches.
	mux.HandleFunc("GET /ds-demo", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(dsDemoPage))
	})

	fmt.Printf(
		"setup-demo\nOpen http://localhost%s/dev-login  (signs in as %s)\n",
		addr,
		adminEmail,
	)
	fmt.Println(
		"Routes: /admin/ · /dashboard/ · /health · /auth/* · /sse · /ds/events · /ds-demo · POST /broadcast · / (login page)",
	)

	return bundle.RunHandler(ctx, addr, bundle.Handler(mux))
}

// dsDemoPage is the minimal DataStar client for /ds-demo: the SDK connects
// to /ds/events and the span re-renders whenever the "broadcasts" signal is
// patched by POST /broadcast.
const dsDemoPage = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>setup-demo — DataStar feed</title>
  <script type="module" src="/datastar.js"></script>
</head>
<body data-signals-broadcasts="0">
  <h1>Live broadcasts (DataStar)</h1>
  <p>POST /broadcast to bump the counter — the patch arrives on /ds/events.</p>
  <p>Broadcasts so far: <span data-text="$broadcasts"></span></p>
</body>
</html>
`

// seed registers the demo admin and returns their session token.
func seed(ctx context.Context, bundle *setup.Bundle) string {
	adminID := identitymodel.SyntheticUserID(adminUserID)
	resp, err := bundle.Service.Register(ctx, usermgmt.RegisterRequest{
		ID: adminID, Email: adminEmail, DisplayName: "Setup Demo Admin",
	})
	if err != nil {
		log.Fatalf("register admin: %v", err)
	}

	if err := bundle.Service.Authz().AddGroupPolicy(identitymodel.GroupPolicy{
		Subject: adminID.Get().String(), Role: identitymodel.RoleSuperAdmin, Domain: "*",
	}); err != nil {
		log.Fatalf("grant super_admin: %v", err)
	}

	return resp.Session.Token
}
