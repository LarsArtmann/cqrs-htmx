package dashboardui

import (
	"net/http"
	"testing"

	memorystorage "github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

// mustTestDashboardWithConfig builds a dashboard from config. When the config
// sets neither EventSource nor Journal, one fresh in-memory store serves as
// both — a test's Config then states only what it deviates by. For sites
// exercising non-HTTP surface (middleware, capabilities, broadcaster).
func mustTestDashboardWithConfig(t *testing.T, config Config) *Dashboard {
	t.Helper()

	if config.EventSource == nil && config.Journal == nil {
		store := memorystorage.NewMemoryStore()
		config.EventSource = store
		config.Journal = store
	}

	d, err := New(config)
	if err != nil {
		t.Fatalf("New Dashboard: %v", err)
	}

	return d
}

// mustTestDashboardMuxWithConfig mounts the config-built dashboard at
// /dashboard/ on its own mux — the shape handler/layout tests serve requests
// through. Tests needing extra routes (root-index coexistence) add them to
// the returned mux.
func mustTestDashboardMuxWithConfig(t *testing.T, config Config) *http.ServeMux {
	t.Helper()

	d := mustTestDashboardWithConfig(t, config)

	mux := http.NewServeMux()
	d.Mount(mux, "/dashboard/")

	return mux
}

// newTestDashboardMux builds the vanilla test dashboard — a fresh in-memory
// store wired as both EventSource and Journal, mounted at /dashboard/ on its
// own mux — the prologue of any test with no Config deviations. Deviating
// fixtures build their own via mustTestDashboardMuxWithConfig, so the
// store+New+Mount sequence cannot drift between tests.
func newTestDashboardMux(t *testing.T) *http.ServeMux {
	t.Helper()

	return mustTestDashboardMuxWithConfig(t, Config{})
}
