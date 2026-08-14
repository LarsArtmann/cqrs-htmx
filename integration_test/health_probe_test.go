package integration_test

import (
	"testing"

	"github.com/larsartmann/cqrs-htmx/health/v4"
	gohealth "github.com/larsartmann/go-health"
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
