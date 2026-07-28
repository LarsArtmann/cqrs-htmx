// Command admin-demo is a runnable showcase of the adminui panel.
//
// It boots an in-memory usermgmt service, seeds a handful of demo users and
// tenants, and mounts the admin panel at /admin behind cookie session auth.
// Open http://localhost:8097/ to be signed in as the demo admin automatically.
//
// This is a demo only — it uses in-memory storage and a dev-only login
// shortcut. Real applications back the panel with a persistent event store and
// authenticate via WebAuthn or OAuth2.
//
// v4 auth strategy injection: The TOTP provider is injected below, demonstrating
// the v4 sub-module pattern. Consumers import only the auth strategies they need:
//
//	import totp "github.com/larsartmann/cqrs-htmx/usermgmt/totp/v4"
//	import webauthn "github.com/larsartmann/cqrs-htmx/usermgmt/webauthn/v4"
//	import oauth2 "github.com/larsartmann/cqrs-htmx/usermgmt/oauth2/v4"
//
//	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{
//	    TOTP:     totp.New(totp.DefaultConfig()),
//	    WebAuthn: webauthn.New(webauthn.Config{ RPID: "myapp.com", ... }),
//	    OAuth2:   oauth2.New(oauth2.Config{ Providers: map[string]oauth2.ProviderConfig{...} }),
//	})
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/larsartmann/cqrs-htmx/adminui/v4"
	totp "github.com/larsartmann/cqrs-htmx/usermgmt/totp/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

const (
	addr        = ":8097"
	cookieName  = "session"
	adminEmail  = "admin@demo.dev"
	adminUserID = "01JXSUPERADMIN001"
)

func main() {
	ctx := context.Background()

	// 1. Event-sourced user management, in-memory, with an audit log projection.
	//    v4: TOTP strategy injected as an optional sub-module — demonstrates the
	//    import + injection pattern. In a real app you'd also mount the auth
	//    routes (NewAuthHandler(svc).RegisterRoutes(mux)) so the TOTP endpoints
	//    (/auth/totp/setup, /auth/totp/verify, etc.) are reachable. This demo
	//    uses a dev-login shortcut for simplicity.
	totpProvider := totp.New(totp.Config{Issuer: "cqrs-htmx Demo"})
	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{ //nolint:exhaustruct // demo uses in-memory defaults
		AuditLog: usermgmt.NewAuditLog(),
		TOTP:     totpProvider,
	})
	if err != nil {
		log.Fatalf("NewService: %v", err)
	}

	// 2. Seed an admin (with a session) plus demo users and tenants.
	token := seed(ctx, svc)

	// 3. Build the admin panel. The default (role-based) authorizer is used, so
	//    the demo assigns the admin the super_admin role in seed() — proving the
	//    default works rather than bypassing it with RequireAuthenticated.
	//    SSEURL enables the honest UI sync indicator + SSE connection.
	broadcaster := cqrshtmx.NewBroadcaster()

	// Idempotency store: prevents duplicate command execution when a client
	// retries after losing the ACK. The X-Command-Id header ties each mutation
	// to a unique ID; a retry with the same ID gets 409 instead of re-executing.
	idemStore := cqrshtmx.NewMemoryIdempotencyStore(5 * time.Minute)
	defer idemStore.Close()
	panel, err := adminui.New(adminui.Config{ //nolint:exhaustruct // demo uses sensible defaults
		Service:     svc,
		Title:       "cqrs-htmx Admin",
		AccentColor: "#0ea5e9",
		LogoutURL:   "/dev-logout",
		SSEURL:      "/admin/-/events",
	})
	if err != nil {
		log.Fatalf("adminui.New: %v", err)
	}

	// 4. Wire routes. The panel sits behind session middleware + CSRF + the
	//    panel's recommended middleware (recovery + security headers). This is
	//    the production-ready wiring pattern.
	mux := http.NewServeMux()
	sessionMW := usermgmt.NewSessionMiddleware(svc, cookieName)
	csrfMW := cqrshtmx.CSRFMiddleware(cqrshtmx.CSRFConfig{})
	panelMW := panel.Middleware()

	// ACK + idempotency middleware: rejects duplicate commands (409) before
	// they reach the panel, then broadcasts the command ACK over SSE after each
	// mutation. This closes the honest UI loop: pending → confirmed/rejected,
	// and prevents double-execution on retry.
	ackMW := ackMiddleware(broadcaster, idemStore)

	mux.Handle("/admin/", ackMW(sessionMW(csrfMW(panelMW(http.StripPrefix("/admin", panel.Handler()))))))

	// SSE endpoint — streams live events from the Broadcaster to the browser.
	// Sits behind session middleware (auth required) but NOT CSRF (GET-only, safe).
	mux.Handle("/admin/-/events", sessionMW(sseHandler(broadcaster)))

	mux.HandleFunc("/dev-login", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{ //nolint:exhaustruct,gosec // dev-only demo cookie
			Name: cookieName, Value: token, Path: "/",
			HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: 86400,
		})
		http.Redirect(w, r, "/admin/", http.StatusSeeOther)
	})
	mux.HandleFunc("/dev-logout", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{ //nolint:exhaustruct,gosec // dev-only
			Name: cookieName, Value: "", Path: "/", MaxAge: -1,
		})
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/dev-login", http.StatusSeeOther)
	})

	fmt.Printf("admin-demo\nOpen http://localhost%s/  (auto-signs in as %s)\n", addr, adminEmail)
	srv := &http.Server{ //nolint:exhaustruct // demo server
		Addr: addr,
		Handler: cqrshtmx.ServerTimingMiddlewareWhen(func(r *http.Request) bool {
			return r.URL.Query().Has("debug")
		})(logging(mux)),
		ReadHeaderTimeout: 5 * time.Second,
	}
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

// seed registers the admin (returning a session token) plus demo users and
// tenants so the panel has something to show. Storage is in-memory, so every
// boot starts fresh and the registrations always succeed.
func seed(ctx context.Context, svc *usermgmt.Service) string {
	adminID := usermgmt.SyntheticUserID(adminUserID)
	resp, err := svc.Register(ctx, usermgmt.RegisterRequest{
		ID: adminID, Email: adminEmail, DisplayName: "Demo Admin",
	})
	if err != nil {
		log.Fatalf("register admin: %v", err)
	}
	// Grant super_admin so the panel's default role-based authorizer admits the
	// demo admin. This exercises the real authorization path (not a bypass).
	if err := svc.Authz().AddGroupPolicy(usermgmt.GroupPolicy{
		Subject: adminID.Get().String(), Role: usermgmt.RoleSuperAdmin, Domain: "*",
	}); err != nil {
		log.Fatalf("grant super_admin: %v", err)
	}

	for _, email := range []string{
		"alice@acme.dev", "bob@acme.dev", "carol@other.dev", "dave@acme.dev",
	} {
		uid := usermgmt.SyntheticUserID("seed-" + email)
		if _, err := svc.Register(ctx, usermgmt.RegisterRequest{
			ID: uid, Email: email, DisplayName: nameOf(email),
		}); err != nil {
			log.Printf("seed register %s: %v", email, err)
		}
	}

	for _, t := range []struct{ id, name, display string }{
		{"acme", "acme", "Acme Corporation"},
		{"globex", "globex", "Globex International"},
		{"initech", "initech", "Initech"},
	} {
		if _, err := svc.CreateTenant(ctx, usermgmt.CreateTenantRequest{
			ID: usermgmt.NewTenantID(t.id), Name: t.name, DisplayName: t.display,
		}); err != nil {
			log.Printf("seed tenant %s: %v", t.id, err)
		}
	}

	return resp.Session.Token
}

// nameOf turns an email into a capitalized display name.
func nameOf(email string) string {
	for i, r := range email {
		if r == '@' {
			n := email[:i]
			if len(n) > 0 {
				n = string(toUpper(n[0])) + n[1:]
			}
			return n
		}
	}
	return email
}

func toUpper(b byte) byte {
	if b >= 'a' && b <= 'z' {
		return b - 32
	}
	return b
}

// logging is a tiny request logger for the demo.
func logging(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.Path) //nolint:gosec // demo logging
		h.ServeHTTP(w, r)
	})
}

// statusRecorder wraps http.ResponseWriter to capture the status code.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (sr *statusRecorder) WriteHeader(code int) {
	sr.status = code
	sr.ResponseWriter.WriteHeader(code)
}

// ackMiddleware rejects duplicate commands via the idempotency store, then
// broadcasts a command ACK over SSE after each mutation request.
// The X-Command-Id header (auto-generated by admin.js) ties the ACK back to the
// pending UI item. This closes the honest UI loop: pending → confirmed/rejected
// and prevents double-execution when a client retries after losing the ACK.
func ackMiddleware(bc *cqrshtmx.Broadcaster, idem cqrshtmx.IdempotencyStore) func(http.Handler) http.Handler {
	hook := bc.BroadcastOnAck()
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Idempotency check: reject duplicate command IDs before processing.
			if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodDelete {
				if cmdID := cqrshtmx.CommandIDFromRequest(r); cmdID != "" {
					if err := idem.CheckAndRecord(r.Context(), cmdID, 10*time.Minute); err != nil {
						w.Header().Set("Content-Type", "text/plain; charset=utf-8")
						w.WriteHeader(http.StatusConflict)
						_, _ = w.Write([]byte(err.Error()))
						return
					}
				}
			}

			sr := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(sr, r)

			// Only broadcast ACKs for mutation methods with a command ID
			if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodDelete {
				if cqrshtmx.CommandIDFromRequest(r) != "" {
					if sr.status >= 400 {
						hook(r.Context(), r, errorfamily.NewRejection(
							"admin-demo.http_error", fmt.Sprintf("HTTP %d", sr.status),
						))
					} else {
						hook(r.Context(), r, nil)
					}
				}
			}
		})
	}
}

// sseHandler streams live SSE events from the Broadcaster to the browser.
func sseHandler(bc *cqrshtmx.Broadcaster) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		stream := cqrshtmx.NewSSEStream(w, r)
		defer stream.Close()

		// Flush headers immediately
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}

		ch := bc.Subscribe()
		defer bc.Unsubscribe(ch)

		for {
			select {
			case <-stream.Context().Done():
				return
			case evt, ok := <-ch:
				if !ok {
					return
				}
				if err := stream.Send(evt); err != nil {
					return
				}
			}
		}
	})
}
