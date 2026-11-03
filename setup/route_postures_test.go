package setup_test

import (
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/larsartmann/cqrs-htmx/setup/v4"
)

func newDefaultBundle(t *testing.T) *setup.Bundle {
	t.Helper()

	bundle, err := setup.New(setup.Config{Logger: slog.New(slog.DiscardHandler)})
	if err != nil {
		t.Fatalf("setup.New: %v", err)
	}
	t.Cleanup(func() { _ = bundle.Close() })

	return bundle
}

// TestRoutePostures_DefaultManifest pins the default aggregate: every layer,
// its posture, and its justification. A new mounted layer changes this golden
// — update it in the same commit as the Mount change.
func TestRoutePostures_DefaultManifest(t *testing.T) {
	t.Parallel()

	bundle := newDefaultBundle(t)

	rows := bundle.RoutePostures()
	lines := make([]string, 0, len(rows))
	for _, row := range rows {
		if row.Why == "" {
			t.Errorf("%s %s: manifest row without a Why", row.Method, row.Pattern)
		}
		lines = append(lines, row.Method+" "+row.Pattern+" "+string(row.Posture))
	}

	got := strings.Join(lines, "\n")
	want := strings.Join([]string{
		"* /auth/* ungated",
		"GET / public",
		"* /admin/ role",
		"* /dashboard/ authenticated",
		"* /health public",
	}, "\n")

	if got != want {
		t.Fatalf("default aggregate manifest drifted:\n--- want ---\n%s\n--- got ---\n%s", want, got)
	}
}

// TestRoutePostures_DashboardAuthorizerRefinesToRoleGated proves the
// dashboard row reports role-gated exactly when DashboardAuthorizer is set.
func TestRoutePostures_DashboardAuthorizerRefinesToRoleGated(t *testing.T) {
	t.Parallel()

	bundle := newDefaultBundle(t)
	for _, row := range bundle.RoutePostures() {
		if row.Pattern == "/dashboard/" && row.Posture != "authenticated" {
			t.Fatalf("dashboard row without authorizer: got %s, want authenticated", row.Posture)
		}
	}

	refined, err := setup.New(setup.Config{
		Logger:              slog.New(slog.DiscardHandler),
		DashboardAuthorizer: func(*http.Request) error { return nil },
	})
	if err != nil {
		t.Fatalf("setup.New with DashboardAuthorizer: %v", err)
	}
	defer func() { _ = refined.Close() }()

	for _, row := range refined.RoutePostures() {
		if row.Pattern == "/dashboard/" && row.Posture != "role" {
			t.Fatalf("dashboard row with authorizer: got %s, want role", row.Posture)
		}
	}
}

// TestRoutePostures_DisabledLayersAreAbsent proves the manifest honors the
// Disable* flags: a disabled layer mounts nothing and declares nothing.
func TestRoutePostures_DisabledLayersAreAbsent(t *testing.T) {
	t.Parallel()

	bundle, err := setup.New(setup.Config{
		Logger:           slog.New(slog.DiscardHandler),
		DisableAuth:      true,
		DisableLogin:     true,
		DisableAdmin:     true,
		DisableDashboard: true,
		HealthPath:       "-",
	})
	if err != nil {
		t.Fatalf("setup.New fully disabled: %v", err)
	}
	defer func() { _ = bundle.Close() }()

	if rows := bundle.RoutePostures(); len(rows) != 0 {
		t.Fatalf("fully-disabled bundle must declare nothing, got %d rows", len(rows))
	}
}

// TestRoutePostures_OptInEndpointsAppear proves each opt-in endpoint adds its
// session-gated row (and the public script rows for the feeds).
func TestRoutePostures_OptInEndpointsAppear(t *testing.T) {
	t.Parallel()

	bundle, err := setup.New(setup.Config{
		Logger:               slog.New(slog.DiscardHandler),
		SSEPath:              "/sse",
		SSEScriptPath:        "/sse.js",
		DataStarPath:         "/ds",
		DataStarScriptPath:   "/ds.js",
		LivePath:             "/live",
		EventCatalogPath:     "/events/catalog.json",
		ProjectionStatusPath: "/projections/status.json",
		DebugPath:            "/debug.json",
		DocsPath:             "/docs",
		DocsHandler:          http.NotFoundHandler(),
	})
	if err != nil {
		t.Fatalf("setup.New with opt-ins: %v", err)
	}
	defer func() { _ = bundle.Close() }()

	want := map[string]string{
		"/sse":                     "authenticated",
		"/sse.js":                  "public",
		"/ds":                      "authenticated",
		"/ds.js":                   "public",
		"/live":                    "public",
		"/events/catalog.json":     "authenticated",
		"/projections/status.json": "authenticated",
		"/debug.json":              "authenticated",
		"/docs":                    "authenticated",
	}

	got := map[string]string{}
	for _, row := range bundle.RoutePostures() {
		got[row.Pattern] = string(row.Posture)
	}

	for pattern, posture := range want {
		if got[pattern] != posture {
			t.Errorf("%s: got posture %q, want %q", pattern, got[pattern], posture)
		}
	}
}
