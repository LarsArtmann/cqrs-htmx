// Command admin-demo is a runnable showcase of the adminui panel.
//
// It boots an in-memory usermgmt service, seeds a handful of demo users and
// tenants, and mounts the admin panel at /admin behind cookie session auth.
// Open http://localhost:8097/ to be signed in as the demo admin automatically.
//
// This is a demo only — it uses in-memory storage and a dev-only login
// shortcut. Real applications back the panel with a persistent event store and
// authenticate via WebAuthn or OAuth2.
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/larsartmann/cqrs-htmx/adminui/v3"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v3"
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
	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{
		AuditLog: usermgmt.NewAuditLog(),
	})
	if err != nil {
		log.Fatalf("NewService: %v", err)
	}

	// 2. Seed an admin (with a session) plus demo users and tenants.
	token := seed(ctx, svc)

	// 3. Build the admin panel. RequireAuthenticated keeps the demo simple; in
	//    production use adminui.RequireAnyRole(svc, "*", usermgmt.RoleSuperAdmin)
	//    or your own Config.Authorizer.
	// 3. Build the admin panel. The default (role-based) authorizer is used, so
	//    the demo assigns the admin the super_admin role in seed() — proving the
	//    default works rather than bypassing it with RequireAuthenticated.
	panel, err := adminui.New(adminui.Config{
		Service:     svc,
		Title:       "cqrs-htmx Admin",
		AccentColor: "#0ea5e9",
		LogoutURL:   "/dev-logout",
	})
	if err != nil {
		log.Fatalf("adminui.New: %v", err)
	}

	// 4. Wire routes. The panel sits behind the session middleware so that
	//    *usermgmt.User is in the request context.
	mux := http.NewServeMux()
	sessionMW := usermgmt.NewSessionMiddleware(svc, cookieName)
	mux.Handle("/admin/", sessionMW(http.StripPrefix("/admin", panel.Handler())))

	mux.HandleFunc("/dev-login", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{
			Name: cookieName, Value: token, Path: "/",
			HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: 86400,
		})
		http.Redirect(w, r, "/admin/", http.StatusSeeOther)
	})
	mux.HandleFunc("/dev-logout", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: cookieName, Value: "", Path: "/", MaxAge: -1})
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/dev-login", http.StatusSeeOther)
	})

	fmt.Printf("admin-demo\nOpen http://localhost%s/  (auto-signs in as %s)\n", addr, adminEmail)
	srv := &http.Server{Addr: addr, Handler: logging(mux), ReadHeaderTimeout: 5 * time.Second}
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

// seed registers the admin (returning a session token) plus demo users and
// tenants so the panel has something to show. Storage is in-memory, so every
// boot starts fresh and the registrations always succeed.
func seed(ctx context.Context, svc *usermgmt.Service) string {
	adminID := usermgmt.NewUserID(adminUserID)
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
		uid := usermgmt.NewUserID("seed-" + email)
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
				n = string(toUpper(byte(n[0]))) + n[1:]
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
		log.Printf("%s %s", r.Method, r.URL.Path)
		h.ServeHTTP(w, r)
	})
}
