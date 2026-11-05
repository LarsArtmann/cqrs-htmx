package setup_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	"github.com/larsartmann/cqrs-htmx/v4/authztest"
)

// The manifest tests in route_postures_test.go pin what the bundle DECLARES;
// this file pins that the declarations are TRUE — the same authztest.Suite
// consumers are told to run, against the fully composed default bundle (the
// one-call surface, session middleware and panel gates included). It is the
// repo's own wiring of the authorization workstream's proof kit (its first
// consumer-grade home outside the CI-blocked integration_test self-test).

// withSessionCookie presents a session token as the cookie the bundle's
// session middleware reads (default cookie name "session").
func withSessionCookie(token string) authztest.Actor {
	return func(r *http.Request) *http.Request {
		r.AddCookie(&http.Cookie{
			Name:     "session",
			Value:    token,
			Secure:   true,
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
		})

		return r
	}
}

// registerForPostureTest registers a fresh user and returns its ID and the
// session token Register issued.
func registerForPostureTest(
	t *testing.T, svc *usermgmt.Service, email string,
) (identitymodel.UserID, string) {
	t.Helper()

	uid := identitymodel.GenerateUserID()
	reg, err := svc.Register(context.Background(), usermgmt.RegisterRequest{
		ID:    uid,
		Email: email,
	})
	if err != nil {
		t.Fatalf("Register %s: %v", email, err)
	}
	if reg.Session == nil {
		t.Fatalf("Register %s: no session issued", email)
	}

	return uid, reg.Session.Token
}

// awaitGlobalAdminGrant polls until the wildcard-domain admin grant is
// visible: GrantRoles dispatches a command and the CasbinProjection applies
// it in its own goroutine — probing before that would measure projection
// lag, not posture.
func awaitGlobalAdminGrant(t *testing.T, svc *usermgmt.Service) {
	t.Helper()

	deadline := time.Now().Add(3 * time.Second)
	domain := identitymodel.NewTenantID(identitymodel.WildcardDomain)

	for {
		has, err := svc.HasAdminGrant(domain)
		if err != nil {
			t.Fatalf("HasAdminGrant: %v", err)
		}

		if has {
			return
		}

		if time.Now().After(deadline) {
			t.Fatal("global admin grant never became visible: projection did not drain")
		}

		time.Sleep(10 * time.Millisecond)
	}
}

// TestRoutePostures_BehaviorMatrix proves the default bundle's declared
// postures behave as declared through the real mounted wiring: the login page
// and readiness probe answer anonymous callers, /auth/me and the dashboard
// refuse them, and the admin panel demands the role (403 for a plain session,
// 200 for the durable wildcard-domain admin grant).
func TestRoutePostures_BehaviorMatrix(t *testing.T) {
	t.Parallel()

	bundle := newDefaultBundle(t)

	mux := http.NewServeMux()
	bundle.Mount(mux)

	svc := bundle.Service

	adminUID, adminToken := registerForPostureTest(t, svc, "posture-admin@test.com")
	_, memberToken := registerForPostureTest(t, svc, "posture-member@test.com")

	wildcard := identitymodel.NewTenantID(identitymodel.WildcardDomain)
	if err := svc.GrantRoles(context.Background(), identitymodel.ActorIDFromUser(adminUID),
		wildcard, []identitymodel.Role{identitymodel.RoleAdmin}); err != nil {
		t.Fatalf("GrantRoles: %v", err)
	}
	awaitGlobalAdminGrant(t, svc)

	authztest.Suite{
		Handler: mux,
		Routes: []authztest.Route{
			{
				Method: http.MethodGet, Path: "/", Posture: authztest.Public,
				Why: "the login page is the entry point; it renders for anonymous callers",
			},
			{
				Method: http.MethodGet, Path: "/health", Posture: authztest.Public,
				Why: "readiness probe; unauthenticated by design (503 while draining is a state, not a denial)",
			},
			{
				Method: http.MethodGet, Path: "/auth/me", Posture: authztest.Authenticated,
				Why: "returns the session user; the ADR-0055 self-wrap refuses anonymous callers",
			},
			{
				Method: http.MethodGet, Path: "/dashboard/", Posture: authztest.Authenticated,
				Why: "the dashboard renders event payloads; the RequireSession gate answers 401 without a session",
			},
			{
				Method: http.MethodGet, Path: "/admin/", Posture: authztest.RoleGated,
				Why: "the admin authorizer demands the role: plain session 403, wildcard grant 200",
			},
		},
		Anonymous:    func(r *http.Request) *http.Request { return r },
		Unprivileged: withSessionCookie(memberToken),
		Privileged:   withSessionCookie(adminToken),
	}.Run(t)
}
