package integration_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	auditlog "github.com/larsartmann/cqrs-htmx/auditlog/v4"
	"github.com/larsartmann/cqrs-htmx/health/v4"
	gohealth "github.com/larsartmann/go-health"
	doauditlog "github.com/larsartmann/samber-do-auditlog"
	"github.com/larsartmann/samber-do-auditlog/live"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/require"
)

// TestHealthProbe_AgainstRealService proves the health/v4 bridge against a
// REAL *usermgmt.Service (the module's own tests use a fake provider): after
// synchronous startup every projection worker is live, so the probe evaluates
// to pass with one named check per projection.
func TestHealthProbe_AgainstRealService(t *testing.T) {
	_, svc := setupFullstackUI(t)

	probe, err := health.NewProbe(svc)
	require.NoError(t, err)

	resp := probe.Evaluate(t.Context())
	require.Equal(t, gohealth.StatusPass, resp.Status)

	// One check per projection worker, plus the three known read/authz models.
	statuses := svc.ProjectionStatuses()
	require.Len(t, resp.Checks, len(statuses))
	for _, name := range []string{"user-read-model", "casbin-projection", "tenant-read-model"} {
		check, ok := resp.Checks[name]
		require.True(t, ok, "expected a check for projection %q", name)
		require.Equal(t, gohealth.StatusPass, check.Status)
		require.Empty(t, check.Error)
	}
}

// TestHealthProbe_RecorderMergesInjectorChecks runs the samber/do merge path
// against the real Service: the injector's own service checks appear next to
// the projection checks in one recorder batch.
func TestHealthProbe_RecorderMergesInjectorChecks(t *testing.T) {
	_, svc := setupFullstackUI(t)

	injector := do.New()
	t.Cleanup(func() { _ = injector.Shutdown() })
	do.ProvideValue(injector, stubHealthchecker{})

	// The elimination baseline is the Service's own projection set (7 workers:
	// user/bot/membership/tenant read models, casbin, audit-log, ...).
	projections := make(map[string]bool, 8)
	for _, entry := range svc.ProjectionStatuses() {
		projections[entry.Name] = true
	}
	require.NotEmpty(t, projections)

	results := health.Recorder(svc).RecordHealthCheckWithContext(t.Context(), injector)
	for name, err := range results {
		if projections[name] {
			require.NoError(t, err, "projection %q should be live", name)
		}
	}

	// do derives the injector service's check name from its type; count by
	// elimination (same pattern as health/probe_test.go).
	injectorChecks := 0
	for name, err := range results {
		if projections[name] {
			continue
		}
		require.NoError(t, err)
		injectorChecks++
	}
	require.Equal(t, 1, injectorChecks, "injector service check missing: %v", results)
}

type stubHealthchecker struct{}

func (stubHealthchecker) HealthCheck() error { return nil }

// TestHealthProbe_StartAndRoutesE2E proves the full probe lifecycle against a
// real Service: Start runs the background refresh, RegisterRoutes mounts the
// Kubernetes paths, /readyz serves a real 200 verdict (not a zero-value
// cached pass), and /startupz flips the one-way latch. This test asserts
// PUBLISHED health/v4 behavior only (it runs under GOWORK=off in CI): the
// duration_ns upgrades live with the health module's own suite until the
// next health train.
func TestHealthProbe_StartAndRoutesE2E(t *testing.T) {
	t.Parallel()

	_, svc := setupFullstackUI(t)

	probe, err := health.NewProbe(svc,
		gohealth.WithCriticalServices("user-read-model", "casbin-projection"),
		gohealth.WithRefreshInterval(50*time.Millisecond),
	)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)
	require.NoError(t, probe.Start(ctx))
	t.Cleanup(probe.Shutdown)

	mux := http.NewServeMux()
	probe.RegisterRoutes(mux, gohealth.DefaultRoutes())

	// The handlers serve the cached response; wait for the first refresh.
	deadline := time.Now().Add(10 * time.Second)
	for len(probe.CachedResponse().Checks) == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}

	require.NotEmpty(t, probe.CachedResponse().Checks, "first refresh never populated the cache")

	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	require.Equal(t, http.StatusOK, w.Code, "readyz body: %s", w.Body.String())
	require.Contains(t, w.Body.String(), "user-read-model", "real projection verdict missing")

	// /startupz reports complete and flips the one-way latch (only this
	// handler moves it, so the assertion order matters).
	require.False(t, probe.StartupComplete())

	w = httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/startupz", nil))
	require.Equal(t, http.StatusOK, w.Code, "startupz body: %s", w.Body.String())
	require.True(t, probe.StartupComplete())

	// Liveness answers 200 with the probe's roll-up.
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	require.Equal(t, http.StatusOK, w.Code)
}

// TestHealthProbe_AuditlogPluginIsRecorder proves the auditlog README's
// composition claim: the samber-do-auditlog plugin satisfies go-health's
// HealthRecorder implicitly (compile-time assertion below) and drives a real
// probe's batches over a real injector — DI audit and health in one probe.
// The projections-plus-plugin chain itself (health.RecorderChain) is proven
// in the health module's suite; wiring it here waits for the next health
// train, since this module resolves health/v4 from its published tag under
// CI's GOWORK=off.
func TestHealthProbe_AuditlogPluginIsRecorder(t *testing.T) {
	t.Parallel()

	auditSetup, err := auditlog.WithAuditLog(doauditlog.Config{}, live.Config{})
	require.NoError(t, err)

	var _ gohealth.HealthRecorder = auditSetup.Plugin

	injector := do.New()
	t.Cleanup(func() { _ = injector.Shutdown() })
	do.ProvideValue(injector, stubHealthchecker{})

	// The plugin observes the injector's service checks and reports them.
	pluginResults := auditSetup.Plugin.RecordHealthCheckWithContext(t.Context(), injector)
	require.NotEmpty(t, pluginResults, "plugin recorder should merge injector service checks")
	for name, err := range pluginResults {
		require.NoError(t, err, "injector service %q should be healthy", name)
	}

	// The plugin drives a real probe: one batch answers liveness/readiness.
	probe := gohealth.New(injector, gohealth.WithHealthRecorder(auditSetup.Plugin))
	resp := probe.Evaluate(t.Context())
	require.Equal(t, gohealth.StatusPass, resp.Status)
	require.Len(t, resp.Checks, len(pluginResults))
}
