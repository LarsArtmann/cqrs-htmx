package dashboardui

import (
	"net/http"
	"testing"

	memorystorage "github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

// newTestDashboardMux builds the vanilla test dashboard — a fresh in-memory
// store wired as both EventSource and Journal, mounted at /dashboard/ on its
// own mux — which is the prologue nearly every handler/layout test starts
// from. Configs that deviate (stub journal, event bus, read-only, custom
// authz) build their own via mustTestDashboardWithConfig; the vanilla shape
// lives here so the store+New+Mount sequence cannot drift between tests.
func newTestDashboardMux(t *testing.T) *http.ServeMux {
	t.Helper()

	store := memorystorage.NewMemoryStore()

	d, err := New(Config{EventSource: store, Journal: store})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	mux := http.NewServeMux()
	d.Mount(mux, "/dashboard/")

	return mux
}
