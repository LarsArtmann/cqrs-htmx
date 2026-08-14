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

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/cqrs-htmx/setup/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
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
	//    login page + admin panel + CQRS dashboard + /health, with the
	//    documented middleware ordering applied per panel.
	bundle, err := setup.New(setup.Config{ //nolint:exhaustruct // demo uses in-memory defaults
		Title:     "cqrs-htmx Setup Demo",
		LogoutURL: "/dev-logout",
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

	fmt.Printf("setup-demo\nOpen http://localhost%s/dev-login  (signs in as %s)\n", addr, adminEmail)
	fmt.Println("Routes: /admin/ · /dashboard/ · /health · /auth/* · / (login page)")

	return bundle.RunHandler(ctx, addr, bundle.Handler(mux))
}

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
