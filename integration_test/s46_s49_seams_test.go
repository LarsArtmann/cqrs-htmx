package integration_test

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	auditlog "github.com/larsartmann/cqrs-htmx/auditlog/v4"
	"github.com/larsartmann/cqrs-htmx/health/v4"
	setup "github.com/larsartmann/cqrs-htmx/setup/v4"
	usermgmt "github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
	snapshot "github.com/larsartmann/go-cqrs-lite/snapshot/v4"
	eventstore "github.com/larsartmann/go-cqrs-lite/storage/v4"
	doauditlog "github.com/larsartmann/samber-do-auditlog"
	"github.com/larsartmann/samber-do-auditlog/live"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

// S46/S49 consumer seam proofs: the setup DocsHandler mount (shape-B seam),
// the health RecorderChain "one probe" composition, and the StateShape
// persistence round-trip verdicts.

// TestSetup_DocsHandler_SessionGatedMount is the S46a end-to-end proof: a
// consumer-built docs handler mounted through setup.Config.DocsPath/DocsHandler
// serves its content to an authenticated session and 401s everyone else —
// machine-endpoint gating, exactly as documented.
func TestSetup_DocsHandler_SessionGatedMount(t *testing.T) {
	t.Parallel()

	const marker = "DOCS-CONTENT-MARKER"

	docs := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(marker))
	})

	bundle, err := setup.New(setup.Config{DocsPath: "/docs", DocsHandler: docs})
	require.NoError(t, err)
	t.Cleanup(func() { _ = bundle.Close() })

	mux := http.NewServeMux()
	handler := bundle.Handler(mux)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/docs", nil))
	require.Equal(
		t, http.StatusUnauthorized, w.Code,
		"unauthenticated docs request must 401, body: %s", w.Body.String(),
	)

	resp, err := bundle.Service.Register(t.Context(), usermgmt.RegisterRequest{
		Email:       "docs-seam@example.com",
		DisplayName: "Docs Seam",
	})
	require.NoError(t, err)
	require.NotNil(t, resp.Session, "registration must issue a session (ADR-0055 register-issued cookie)")

	authed := httptest.NewRequest(http.MethodGet, "/docs", nil)
	authed.AddCookie(&http.Cookie{Name: "session", Value: resp.Session.Token}) //nolint:gosec // test cookie

	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, authed)
	require.Equal(t, http.StatusOK, w2.Code, "authenticated docs request must serve, body: %s", w2.Body.String())
	require.Contains(t, w2.Body.String(), marker)
}

// TestSetup_DocsConfigMismatchRejected pins the config contract: DocsPath
// without DocsHandler (or vice versa) is rejected at New, not silently ignored.
func TestSetup_DocsConfigMismatchRejected(t *testing.T) {
	t.Parallel()

	_, err := setup.New(setup.Config{DocsPath: "/docs"})
	require.Error(t, err, "DocsPath without DocsHandler must be rejected at New")
}

// TestHealth_RecorderChain_MergesProjectionsAndAuditPlugin is the S46b chain
// proof: RecorderChain(Recorder(svc), auditPlugin) reports the projection
// checks AND composes with the audit plugin's own HealthRecorder — the
// "one probe" composition from the auditlog row, executed for real.
func TestHealth_RecorderChain_MergesProjectionsAndAuditPlugin(t *testing.T) {
	t.Parallel()

	_, svc := setupFullstackUI(t)

	auditSetup, err := auditlog.WithAuditLog(
		doauditlog.Config{},
		live.Config{Prefix: "/audit-chain"},
	)
	require.NoError(t, err)

	injector := do.New()

	chain := health.RecorderChain(health.Recorder(svc), auditSetup.Plugin)
	results := chain.RecordHealthCheckWithContext(t.Context(), injector)
	require.NotEmpty(t, results, "the chain must report the service's projection checks")

	for name, err := range results {
		require.NoErrorf(t, err, "health check %s", name)
	}

	// Nil recorders are skipped by contract; the chain still reports.
	withNil := health.RecorderChain(health.Recorder(svc), nil, auditSetup.Plugin)
	require.NotEmpty(t, withNil.RecordHealthCheckWithContext(t.Context(), injector))
}

// stampedSnapshot builds a fully-stamped snapshot for the S49 round-trips.
func stampedSnapshot() snapshot.Snapshot {
	return snapshot.Snapshot{
		StreamID:   id.NewStreamID(),
		StreamType: "User",
		Version:    event.Version(3),
		State:      []byte(`{"email":"s49@example.com"}`),
		Encoding:   record.EncodingJSON,
		CreatedAt:  time.Now().UTC(),
		StateShape: "user@v2",
	}
}

// TestS49_MemorySnapshotStore_RoundTripsStateShape pins the in-tree contract:
// the memory store (and its clone path) preserves the StateShape stamp, so
// shape-aware readers can reject stale-shaped snapshots after restarts.
func TestS49_MemorySnapshotStore_RoundTripsStateShape(t *testing.T) {
	t.Parallel()

	store := usermgmt.NewMemorySnapshotStore()
	snap := stampedSnapshot()

	require.NoError(t, store.Save(t.Context(), snap))

	loaded, err := store.Load(t.Context(), id.StreamRef{Type: snap.StreamType, ID: snap.StreamID})
	require.NoError(t, err)
	require.NotNil(t, loaded)
	require.Equal(t, "user@v2", loaded.StateShape, "memory store must round-trip the StateShape stamp")
}

// TestS49_SQLSnapshotStore_DropsStateShape_UpstreamGap pins the verified
// verdict (S49, audited 2026-10-25 against go-cqrs-lite/storage v4.11.1):
// the SQL snapshot store persists only (stream_type, stream_id, version,
// state, created_at) — the StateShape stamp is silently dropped, which
// degrades stamped snapshots to unstamped across restarts for SQL consumers.
// The fix belongs upstream (schema + INSERT/SELECT column); this assertion
// flips to Equal("user@v2") in the same family-bump change that adopts it.
// Upstream ask routed to the W13 owner packet.
func TestS49_SQLSnapshotStore_DropsStateShape_UpstreamGap(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("sqlite", "file:s49_snapshots?mode=memory&cache=shared")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec(eventstore.SQLiteSnapshotSchema())
	require.NoError(t, err)

	store, err := eventstore.NewSQLiteSnapshotStore(db)
	require.NoError(t, err)

	snap := stampedSnapshot()
	require.NoError(t, store.Save(t.Context(), snap))

	loaded, err := store.Load(t.Context(), id.StreamRef{Type: snap.StreamType, ID: snap.StreamID})
	require.NoError(t, err)
	require.NotNil(t, loaded, "the SQL store must load the snapshot it saved")
	require.Equal(t, snap.Version, loaded.Version)
	require.Equal(t, snap.StreamID, loaded.StreamID)
	require.Empty(t, loaded.StateShape,
		"VERDICT CHANGE: upstream storage now round-trips StateShape — flip this assertion")
}
