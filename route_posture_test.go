package cqrshtmx

import "testing"

func TestPostureValid(t *testing.T) {
	t.Parallel()

	for _, posture := range []Posture{PosturePublic, PostureAuthenticated, PostureRoleGated, PostureUngated} {
		if !posture.Valid() {
			t.Errorf("%q should be valid", posture)
		}
	}

	for _, posture := range []Posture{"", "ADMIN", "private", "ungated "} {
		if posture.Valid() {
			t.Errorf("%q should not be valid", posture)
		}
	}
}

func TestPostureProvable(t *testing.T) {
	t.Parallel()

	for _, posture := range []Posture{PosturePublic, PostureAuthenticated, PostureRoleGated} {
		if !posture.Provable() {
			t.Errorf("%q should be provable", posture)
		}
	}

	if PostureUngated.Provable() {
		t.Error("ungated is a manifest truth, not a probe contract — it must not be provable")
	}
}

func TestRoutePostureString(t *testing.T) {
	t.Parallel()

	row := RoutePosture{Method: "GET", Pattern: "/users/{id}", Posture: PostureRoleGated, Why: "identity data"}
	if got, want := row.String(), "GET /users/{id}"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
