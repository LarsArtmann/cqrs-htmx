package cqrshtmx

// Posture is what a route requires of its caller: the one vocabulary every
// route manifest in the family speaks (adminui, dashboardui, loginpage,
// setup) and every authztest declaration is written in.
//
// It lives here — not in authztest — because production modules expose
// manifests ([adminui.Handler.Routes], [Dashboard.Routes], setup's
// aggregate), and importing a test kit from production code would pull
// httptest into every consumer binary. authztest aliases these constants;
// its probe matrix keeps the fail-closed semantics.
type Posture string

const (
	// PosturePublic routes answer an anonymous caller. Declaring a route
	// public is a claim the matrix proves with an anonymous probe.
	PosturePublic Posture = "public"

	// PostureAuthenticated routes refuse an anonymous caller (401/403) and
	// admit an authenticated one regardless of role.
	PostureAuthenticated Posture = "authenticated"

	// PostureRoleGated routes refuse an anonymous caller, refuse an
	// authenticated caller without the role (exactly 403), and admit one
	// who holds it.
	PostureRoleGated Posture = "role"

	// PostureUngated means the declaration's layer adds no gate: the routes
	// beneath it own their postures, or the consumer's middleware does. A
	// dashboard mounted without an authorizer, or a mount that delegates a
	// whole sub-surface (setup's /auth/* layer), reports this instead of
	// guessing. The authztest matrix refuses it — probe the real routes
	// through your actual wiring instead.
	PostureUngated Posture = "ungated"
)

// Valid reports whether p is a declared posture.
func (p Posture) Valid() bool {
	switch p {
	case PosturePublic, PostureAuthenticated, PostureRoleGated, PostureUngated:
		return true
	}

	return false
}

// Provable reports whether p is a posture the authztest matrix can verify on
// its own. [PostureUngated] is a manifest truth, not a probe contract.
func (p Posture) Provable() bool {
	return p == PosturePublic || p == PostureAuthenticated || p == PostureRoleGated
}

// RoutePosture is one declared route row of a manifest: the method, the
// pattern (wildcards in braces, "*" for any method), the posture, and the
// reason the posture is correct.
//
// Why is required by convention: a posture without a reason is a guess, and a
// guess reviewed later is indistinguishable from a decision.
type RoutePosture struct {
	// Method is the HTTP method the route answers, or "*" for any.
	Method string

	// Pattern is the route pattern, wildcards in braces ("/users/{id}").
	Pattern string

	// Posture is what the route requires.
	Posture Posture

	// Why is the reason this posture is correct.
	Why string
}

// String renders the row as "METHOD /pattern" — the identity used in
// manifests, failure messages, and coverage comparisons.
func (r RoutePosture) String() string { return r.Method + " " + r.Pattern }
