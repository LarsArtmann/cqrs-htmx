package adminui

import (
	"net/http"
	"strings"
	"testing"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	"github.com/larsartmann/cqrs-htmx/v4/authztest"
)

// This file wires [authztest.Suite] over the panel's own route table: every
// [Handler.Routes] row must refuse the callers its posture says it refuses.
// The Suite refuses ungated declarations itself (fail-closed), so a posture
// that drifts to something unprovable fails loudly instead of skipping.

// postureTestActor wraps a handler chain that injects user into the request
// context — the same shape a consumer's session middleware provides. A nil
// user is the anonymous actor: nothing injected.
func postureTestActor(user *identitymodel.User) authztest.Actor {
	return func(req *http.Request) *http.Request {
		if user == nil {
			return req
		}
		return req.WithContext(usermgmt.WithUser(req.Context(), user))
	}
}

// postureTestPanel builds a super-admin panel backed by an in-memory service
// with the DEFAULT role-gated authorizer, plus the two authenticated actors
// the posture matrix needs: a plain user (unprivileged) and a user holding
// the super_admin platform role (privileged), both registered on the service
// so the role check sees real grants.
func postureTestPanel(t *testing.T) (authztest.Suite, *usermgmt.Service) {
	t.Helper()

	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{AuditLog: usermgmt.NewAuditLog()})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	plain := identitymodel.NewUser(identitymodel.SyntheticUserID("01HPLAINUSER"), "plain@example.com", "")
	admin := identitymodel.NewUser(identitymodel.SyntheticUserID("01HADMINUSER"), "admin@example.com", "")
	if err := svc.GrantRoles(t.Context(),
		identitymodel.ActorIDFromUser(admin.ID),
		identitymodel.NewTenantID(identitymodel.WildcardDomain),
		[]identitymodel.Role{identitymodel.RoleSuperAdmin},
	); err != nil {
		t.Fatalf("GrantRoles: %v", err)
	}

	panel, err := New(Config{Service: svc})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	suite := authztest.Suite{
		Handler:      panel.Handler(),
		Routes:       declaredRoutes(panel.Routes()),
		Anonymous:    postureTestActor(nil),
		Unprivileged: postureTestActor(plain),
		Privileged:   postureTestActor(admin),
	}

	return suite, svc
}

// declaredRoutes converts the panel manifest into authztest declarations.
// The manifest's patterns become concrete probe paths (this is what gets
// requested); the methodless "*" fallback row is probed with PUT on an
// unknown path — a method+path no declared GET route matches, so the probe
// lands on the 405 catch-all row itself, whose posture is public (it must
// answer anyone with 405, never a 401/403).
func declaredRoutes(routes []Route) []authztest.Route {
	declared := make([]authztest.Route, 0, len(routes))
	for _, route := range routes {
		method, path := route.Method, route.Pattern
		if method == "*" {
			method, path = http.MethodPut, "/no-such-page"
		}
		path = concreteProbePath(path)
		declared = append(declared, authztest.Route{
			Method:  method,
			Path:    path,
			Posture: route.Posture,
			Why:     route.Why,
		})
	}

	return declared
}

// concreteProbePath replaces each wildcard segment with a concrete id, so
// the probe requests a real URL under the pattern.
func concreteProbePath(pattern string) string {
	replacer := strings.NewReplacer(
		"{id}", "u-1",
		"{tenant}", "t-1",
		"{actor}", "a-1",
		"{credential}", "c-1",
		"{provider}", "github",
	)

	return replacer.Replace(pattern)
}

// TestSecurityPosture_Matrix runs the authztest posture matrix over the whole
// panel route table: every declared row refuses anonymous callers (401),
// role-gated rows refuse authenticated non-admins (403), and the privileged
// admin is admitted. Assets are proven public against the anonymous caller.
func TestSecurityPosture_Matrix(t *testing.T) {
	t.Parallel()

	suite, _ := postureTestPanel(t)
	suite.Run(t)
}

// TestSecurityPosture_DeclarationsMatchManifest pins that the declarations
// the matrix ran are EXACTLY the panel's manifest rows — the matrix cannot
// excuse a route that quietly dropped out of its declared set.
func TestSecurityPosture_DeclarationsMatchManifest(t *testing.T) {
	t.Parallel()

	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{AuditLog: usermgmt.NewAuditLog()})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	panel, err := New(Config{Service: svc})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	suite, _ := postureTestPanel(t)
	if len(suite.Routes) != len(panel.Routes()) {
		t.Fatalf("declared %d routes, manifest has %d", len(suite.Routes), len(panel.Routes()))
	}
}

// TestSecurityPosture_FailClosed proves the Suite refuses to run an
// unprovable declaration: an ungated posture is rejected before any request
// is made, so a lying manifest fails loudly instead of skipping cases.
func TestSecurityPosture_FailClosed(t *testing.T) {
	t.Parallel()

	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{AuditLog: usermgmt.NewAuditLog()})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	panel, err := New(Config{Service: svc})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	lying := authztest.Suite{
		Handler:      panel.Handler(),
		Routes:       []authztest.Route{{Method: http.MethodGet, Path: "/", Posture: authztest.Ungated, Why: "lie"}},
		Anonymous:    postureTestActor(nil),
		Unprivileged: postureTestActor(nil),
		Privileged:   postureTestActor(nil),
	}
	if err := lying.Verify(); err == nil {
		t.Fatal("a suite declaring an ungated route must be refused before any probe runs")
	}
}
