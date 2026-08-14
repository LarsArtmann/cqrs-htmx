package auditlog

import (
	"net/http"
	"net/http/httptest"
	"testing"

	auditlog "github.com/larsartmann/samber-do-auditlog"
	"github.com/larsartmann/samber-do-auditlog/live"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/require"
)

// TestWithAuditLog_WiresContainerAndViewer is the reference cqrs-htmx
// integration: a do container built with the audit option, a service invoked
// and shut down, and the viewer serving the recorded audit trail.
func TestWithAuditLog_WiresContainerAndViewer(t *testing.T) {
	t.Parallel()

	setup, err := WithAuditLog(
		auditlog.Config{MaxEvents: 100},
		live.Config{Prefix: "/auditlog"},
	)
	require.NoError(t, err)
	require.NotNil(t, setup.Opts)
	require.NotNil(t, setup.Plugin)
	require.NotNil(t, setup.Viewer)

	injector := do.NewWithOpts(setup.Opts)

	type demoService struct{ name string }

	do.Provide(injector, func(_ do.Injector) (*demoService, error) {
		return &demoService{name: "demo"}, nil
	})

	svc, err := do.Invoke[*demoService](injector)
	require.NoError(t, err)
	require.Equal(t, "demo", svc.name)

	report := setup.Plugin.Report()
	require.NotEmpty(t, report.Events, "service invocation should be audited")

	_ = injector.Shutdown()
}

func TestWithAuditLog_ViewerServesDashboard(t *testing.T) {
	t.Parallel()

	setup, err := WithAuditLog(
		auditlog.Config{},
		live.Config{Prefix: "/auditlog"},
	)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/auditlog/", nil)
	setup.Viewer.ServeHTTP(w, r)

	require.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())
	require.Contains(t, w.Body.String(), "audit", "dashboard HTML should render")
}

func TestWithAuditLog_ViewerServesReportJSON(t *testing.T) {
	t.Parallel()

	setup, err := WithAuditLog(
		auditlog.Config{},
		live.Config{Prefix: "/auditlog"},
	)
	require.NoError(t, err)

	injector := do.NewWithOpts(setup.Opts)

	type probeService struct{}

	do.Provide(injector, func(_ do.Injector) (*probeService, error) {
		return &probeService{}, nil
	})

	_, err = do.Invoke[*probeService](injector)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/auditlog/api/report", nil)
	setup.Viewer.ServeHTTP(w, r)

	require.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())
	require.Contains(t, w.Body.String(), "services")

	_ = injector.Shutdown()
}

func TestWithAuditLog_InvalidConfigReturnsOrchestrationError(t *testing.T) {
	t.Parallel()

	setup, err := WithAuditLog(
		auditlog.Config{ContainerID: "bad/id"},
		live.Config{Prefix: "/auditlog"},
	)
	require.Nil(t, setup)
	require.Error(t, err)
	require.Contains(t, err.Error(), "cqrshtmx.auditlog.setup_failed")
}
