package integration_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	auditlog "github.com/larsartmann/cqrs-htmx/auditlog/v4"
	"github.com/larsartmann/samber-do-auditlog/live"
	"github.com/stretchr/testify/require"
)

// TestAuditLogViewer_AlongsidFullstackUI proves the auditlog/v4 bridge's live
// viewer serves next to the fullstack UI: the viewer mounts as a plain
// http.Handler on the same mux as admin/dashboard/login, renders its HTML UI,
// and exposes the JSON events API — the cross-module proof that
// auditlog.WithAuditLog composes with a real usermgmt.Service stack.
func TestAuditLogViewer_AlongsidFullstackUI(t *testing.T) {
	t.Parallel()

	handler, _ := setupFullstackUI(t)

	auditSetup, err := auditlog.WithAuditLog(
		live.Config{Prefix: "/audit"}, //nolint:exhaustruct // test defaults; the plugin enables itself
	)
	require.NoError(t, err)

	mux := http.NewServeMux()
	mux.Handle("/audit/", http.StripPrefix("/audit", auditSetup.Viewer))
	mux.Handle("/", handler)

	// The dashboard UI renders.
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/audit/", nil))
	require.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())

	// The JSON report API responds (the endpoint the SSE dashboard polls).
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/audit/api/report", nil))
	require.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())
}
