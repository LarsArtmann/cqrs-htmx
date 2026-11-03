package integration_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/cqrs-htmx/v4/authztest"
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// This file is the repo's own dogfood of the authztest kit (plan items m04.7
// and m04.9): the same suites consumers are told to import, run against a real
// usermgmt.Service and a really mounted HTTP surface — not the kit's internal
// fixtures. If the kit regresses against its own library, this file goes red.

const (
	grantTenant        = "tenant-alpha"
	grantForeignTenant = "tenant-beta"
)

// serviceGovernor adapts *usermgmt.Service to authztest.Governor. It also does
// the one thing the kit's README insists on: after each write it waits, with a
// bounded poll, until the CasbinProjection has applied the grant or revoke.
// Without this the suite would measure projection lag, not policy truth.
type serviceGovernor struct {
	svc *usermgmt.Service
}

func (g serviceGovernor) GrantRoles(
	ctx context.Context, subject, tenant string, roles []string,
) error {
	converted := make([]identitymodel.Role, len(roles))
	for i, r := range roles {
		converted[i] = identitymodel.Role(r)
	}

	if err := g.svc.GrantRoles(ctx, identitymodel.ActorIDFromUser(cqrshtmx.MustParseUserID(subject)),
		identitymodel.NewTenantID(tenant), converted); err != nil {
		return errorfamily.WrapTransient(err, "authztest_selftest.grant", "grant roles")
	}

	return g.awaitPolicy(ctx, subject, tenant, true)
}

func (g serviceGovernor) RevokeRoles(ctx context.Context, subject, tenant string) error {
	if err := g.svc.RevokeRoles(ctx,
		identitymodel.ActorIDFromUser(cqrshtmx.MustParseUserID(subject)),
		identitymodel.NewTenantID(tenant)); err != nil {
		return errorfamily.WrapTransient(err, "authztest_selftest.revoke", "revoke roles")
	}

	return g.awaitPolicy(ctx, subject, tenant, false)
}

// awaitPolicy polls until Decide reports the subject allowed (or not) in the
// tenant: the observable effect of the projection having applied the write.
// The 3s bound mirrors usermgmt's own waitForAdminGrant; a projection that
// never applies the write surfaces as a deadline error naming the tenant.
func (g serviceGovernor) awaitPolicy(
	ctx context.Context, subject, tenant string, wantAllowed bool,
) error {
	deadline := time.Now().Add(3 * time.Second)

	for {
		decision, err := cqrshtmx.Decide(g.svc.Authz(), cqrshtmx.Request{
			Subject:  subject,
			Domain:   tenant,
			Resource: cqrshtmx.Resource{Type: "authztest-drain", ID: "probe"},
			Action:   "probe",
		})
		if err != nil {
			return errorfamily.WrapTransient(err, "authztest_selftest.decide", "await policy drain")
		}

		if decision.Allowed == wantAllowed {
			return nil
		}

		if time.Now().After(deadline) {
			return errorfamily.NewTransient(
				"authztest_selftest.drain_timeout",
				"projection never applied the grant/revoke in tenant "+tenant,
			)
		}

		select {
		case <-ctx.Done():
			return errorfamily.WrapTransient(ctx.Err(), "authztest_selftest.ctx", "await policy drain")
		case <-time.After(10 * time.Millisecond):
		}
	}
}

// TestAuthzGrantSuite_RealService runs the durable-grant suite against a real
// event-sourced Service: tenant isolation, revocation, and a Fresh checker
// built from the journal alone (a second service lifetime over the same store
// — a restart in effect).
func TestAuthzGrantSuite_RealService(t *testing.T) {
	shared := memory.NewMemoryStore()

	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{EventStore: shared})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	t.Cleanup(func() { _ = svc.Close() })

	uid := cqrshtmx.NewUserID()
	registerTestUser(t, svc, uid, "authz-grants@test.com")

	suite := authztest.GrantSuite{
		Checker:       svc.Authz(),
		Governor:      serviceGovernor{svc: svc},
		Subject:       uid.String(),
		Tenant:        grantTenant,
		ForeignTenant: grantForeignTenant,
		Roles:         []string{string(identitymodel.RoleAdmin)},
		ResourceType:  "orders",
		Action:        "read",
		Fresh: func(t *testing.T) cqrshtmx.PermissionChecker {
			t.Helper()

			second, err := usermgmt.NewService(usermgmt.ServiceConfig{EventStore: shared})
			if err != nil {
				t.Fatalf("second service lifetime: %v", err)
			}
			t.Cleanup(func() { _ = second.Close() })

			return second.Authz()
		},
	}

	suite.ProveTenantIsolation(t)
	suite.ProveRevocation(t)
	suite.ProveDurability(t)
}

// variadicAuthz adapts *identitymodel.Authz to cqrshtmx.Enforcer: the typed
// 4-arg Enforce does not satisfy the variadic interface, EnforceAny does the
// same casbin call in the shape the App asks it.
type variadicAuthz struct{ az *identitymodel.Authz }

func (v variadicAuthz) Enforce(rvals ...any) (bool, error) {
	ok, err := v.az.EnforceAny(rvals...)
	if err != nil {
		return false, errorfamily.WrapTransient(err, "authztest_selftest.enforce", "enforce any")
	}

	return ok, nil
}

// decodeOrder builds the command behind a mounted CQRS endpoint. The posture
// probes never send a body, so the command carries no payload.
func decodeOrder(kind command.Type) func(struct{ Kind string }) (command.Command, error) {
	return func(struct{ Kind string }) (command.Command, error) {
		return command.New(kind, id.NewStreamID())
	}
}

// waitForGlobalAdmin polls until the wildcard-domain admin grant is visible:
// GrantRoles dispatches a command, the CasbinProjection applies it in its own
// goroutine, and probing before that would prove lag rather than posture.
func waitForGlobalAdmin(t *testing.T, svc *usermgmt.Service) {
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

// withSession returns an Actor that presents the given session token as the
// session cookie the middleware chain reads.
func withSession(token string) authztest.Actor {
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

// mountPostureSurface builds the canonical stack the posture matrix probes:
// session middleware -> context enrichment -> mounted routes, including one of
// usermgmt's own /auth routes and both gate strengths of the App.
func mountPostureSurface(t *testing.T, svc *usermgmt.Service) http.Handler {
	t.Helper()

	dispatcher := command.NewDispatcher()
	for _, kind := range []command.Type{"SubmitOrder", "CreateOrder"} {
		if err := dispatcher.Register(kind, func(context.Context, command.Command) error {
			return nil
		}); err != nil {
			t.Fatalf("Register %s: %v", kind, err)
		}
	}

	app := cqrshtmx.MustNew(cqrshtmx.Config{Commands: dispatcher, Enforcer: variadicAuthz{az: svc.Authz()}})

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", app.HealthHandler())
	usermgmt.NewAuthHandler(svc).RegisterRoutes(mux)
	mux.Handle("POST /orders", app.Command("SubmitOrder",
		cqrshtmx.DecodeForm(decodeOrder("SubmitOrder")),
		cqrshtmx.RequireAuth(),
		cqrshtmx.WithSuccessStatus(201)))
	mux.Handle("POST /admin/orders", app.Command("CreateOrder",
		cqrshtmx.DecodeForm(decodeOrder("CreateOrder")),
		cqrshtmx.Authorize("orders", "create"),
		cqrshtmx.WithSuccessStatus(201)))

	return cqrshtmx.Chain(
		usermgmt.NewSessionMiddleware(svc, "session"),
		cqrshtmx.ContextEnrichmentMiddleware(func(r *http.Request) (cqrshtmx.UserID, error) {
			raw := usermgmt.UserIDFromRequest(r)
			if raw == "" {
				return cqrshtmx.UserID{}, nil
			}

			return cqrshtmx.MustParseUserID(raw), nil
		}),
	)(mux)
}

// TestAuthzSuite_MountedSurface proves the posture matrix against the canonical
// stack, with the admin grant made through the durable path.
func TestAuthzSuite_MountedSurface(t *testing.T) {
	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	t.Cleanup(func() { _ = svc.Close() })

	adminUID := cqrshtmx.NewUserID()
	admin := registerTestUser(t, svc, adminUID, "authz-admin@test.com")
	if err := svc.GrantRoles(context.Background(), identitymodel.ActorIDFromUser(adminUID),
		identitymodel.NewTenantID(identitymodel.WildcardDomain),
		[]identitymodel.Role{identitymodel.RoleAdmin}); err != nil {
		t.Fatalf("GrantRoles: %v", err)
	}
	waitForGlobalAdmin(t, svc)

	memberUID := cqrshtmx.NewUserID()
	member := registerTestUser(t, svc, memberUID, "authz-member@test.com")

	authztest.Suite{
		Handler: mountPostureSurface(t, svc),
		Routes: []authztest.Route{
			{
				Method: http.MethodGet, Path: "/health", Posture: authztest.Public,
				Why: "liveness answers before authentication",
			},
			{
				Method: http.MethodGet, Path: "/auth/me", Posture: authztest.Authenticated,
				Why: "returns the session user; the ADR-0055 self-wrap refuses anonymous callers",
			},
			{
				Method: http.MethodPost, Path: "/orders", Posture: authztest.Authenticated,
				Why: "any signed-in user may submit an order",
			},
			{
				Method: http.MethodPost, Path: "/admin/orders", Posture: authztest.RoleGated,
				Why: "order creation asks the wildcard domain for admin (the admin-panel grant)",
			},
		},
		Anonymous:    func(r *http.Request) *http.Request { return r },
		Unprivileged: withSession(member.Session.Token),
		Privileged:   withSession(admin.Session.Token),
	}.Run(t)
}
