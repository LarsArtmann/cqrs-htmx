package adminui

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
)

// benchUsers is the full page: the bench renders one 200-row directory page,
// the largest the panel can legally serve (PageSize clamps at 200).
const benchUsers = 200

// newBenchPanel boots a real in-memory service seeded with 200 registered
// users and two tenants behind a panel mounted like a consumer mounts it.
// Seeding cost sits outside the measured loop.
func newBenchPanel(b *testing.B) http.Handler {
	b.Helper()
	ctx := context.Background()
	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{AuditLog: usermgmt.NewAuditLog()})
	if err != nil {
		b.Fatalf("NewService: %v", err)
	}
	handler, err := New(Config{
		Service: svc, Authorizer: RequireAuthenticated(), PageSize: benchUsers,
	})
	if err != nil {
		b.Fatalf("New: %v", err)
	}
	if _, err := svc.CreateTenant(ctx, usermgmt.CreateTenantRequest{
		ID: identitymodel.NewTenantID("acme"), Name: "acme", DisplayName: "Acme Corp",
	}); err != nil {
		b.Fatalf("create tenant: %v", err)
	}
	for i := range benchUsers {
		email := fmt.Sprintf("bench-user-%04d@example.test", i)
		if _, err := svc.Register(ctx, usermgmt.RegisterRequest{
			ID:          identitymodel.SyntheticUserID(fmt.Sprintf("u-bench-%04d", i)),
			Email:       email,
			DisplayName: fmt.Sprintf("Bench User %04d", i),
		}); err != nil {
			b.Fatalf("register %s: %v", email, err)
		}
	}
	admin := mustUserBench("bench-admin@example.test")
	// Warm the last-seen TTL cache so the loop measures steady-state renders.
	_ = handler.lastSeenIndex()
	return serveAsBench(handler, admin)
}

func mustUserBench(email string) *identitymodel.User {
	return identitymodel.NewUser(identitymodel.SyntheticUserID("u-"+email), email, "")
}

// serveAsBench is the bench-local mount (serveAs lives in the test files and
// the bench keeps its own copy so each file stands alone).
func serveAsBench(h *Handler, user *identitymodel.User) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/admin/", http.StripPrefix("/admin", h.Handler()))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mux.ServeHTTP(w, r.WithContext(usermgmt.WithUser(r.Context(), user)))
	})
}

// BenchmarkUsersPageRender measures one full-page GET /users render: directory
// scan, per-render authz snapshot (one GroupPolicies + one AllTenants pass),
// TTL-cached last-seen map, and templ page render, against a real in-memory
// service. Baseline: docs/benchmarks/adminui-baseline.raw.txt.
func BenchmarkUsersPageRender(b *testing.B) {
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.DiscardHandler))
	b.Cleanup(func() { slog.SetDefault(prev) })

	panel := newBenchPanel(b)

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/admin/users", nil)
		panel.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			b.Fatalf("render status = %d, want 200", rec.Code)
		}
	}
	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "req/s")
}
