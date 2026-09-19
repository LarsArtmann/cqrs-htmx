package dashboardui

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v4"
	memorystorage "github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

type fmtStubCommandJournal struct{}

func (fmtStubCommandJournal) ReadAll(context.Context) ([]*command.PersistedCommand, error) {
	return nil, nil
}

type fmtStubQueryJournal struct{}

func (fmtStubQueryJournal) ReadAllQueries(context.Context) ([]*query.PersistedQuery, error) {
	return nil, nil
}

type fmtStubSnapshotStore struct{}

func (fmtStubSnapshotStore) Save(context.Context, snapshot.Snapshot) error { return nil }

func (fmtStubSnapshotStore) Delete(context.Context, id.StreamRef) error { return nil }

func (fmtStubSnapshotStore) Load(context.Context, id.StreamRef) (*snapshot.Snapshot, error) {
	return nil, nil
}

func (fmtStubSnapshotStore) LoadAtVersion(context.Context, id.StreamRef, event.Version) (*snapshot.Snapshot, error) {
	return nil, nil
}

type fmtStubProjection struct{}

func (fmtStubProjection) Name() string                                  { return "fmt-marker-projection" }
func (fmtStubProjection) Handle(_ context.Context, _ event.Event) error { return nil }
func (fmtStubProjection) EventTypes() []event.Type                      { return nil }

// fmtDashboard builds a dashboard with every capability panel enabled and a
// seeded journal, like a real consumer would run.
func fmtDashboard(t *testing.T) http.Handler {
	t.Helper()

	store := memorystorage.NewMemoryStore()

	for _, streamType := range []string{"User", "Tenant"} {
		aggID := id.NewStreamID()
		ref := id.StreamRef{Type: id.StreamType(streamType), ID: aggID}

		for v := uint64(1); v <= 2; v++ {
			evt, eErr := event.New(
				event.Type(streamType+".registered"),
				aggID,
				id.StreamType(streamType),
				event.Version(v),
				map[string]string{"seq": "x"},
			)
			if eErr != nil {
				t.Fatalf("event.New: %v", eErr)
			}

			if aErr := store.AppendBatch(context.Background(), ref, []event.Event{evt}); aErr != nil {
				t.Fatalf("AppendBatch: %v", aErr)
			}
		}
	}

	host, hErr := projectionhost.New(store, memorystorage.NewMemoryCheckpointStore())
	if hErr != nil {
		t.Fatalf("projectionhost.New: %v", hErr)
	}

	if rErr := host.Register(fmtStubProjection{}); rErr != nil {
		t.Fatalf("Register: %v", rErr)
	}

	d, err := New(Config{
		EventSource:     store,
		Journal:         store,
		CommandJournal:  fmtStubCommandJournal{},
		QueryJournal:    fmtStubQueryJournal{},
		SnapshotStore:   fmtStubSnapshotStore{},
		DeadLetterStore: projectionhost.NewMemoryDeadLetterStore(),
		ProjectionHost:  host,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	mux := http.NewServeMux()
	d.Mount(mux, "/dashboard/")

	return mux
}

// TestRenderedPages_FreeOfFmtErrorMarkers renders every dashboard route and
// asserts no Go fmt error markers ("%!(EXTRA ...)") leak into the HTML.
// Sprintf argument-count bugs pass every string-contains assertion and only
// surface visually - exactly the class the screenshot pass caught on the
// events filter bar (a leftover fifth basePath argument). Every route must
// also serve 200: a capability panel that silently 404s would hide its
// surface from this and every other page-level assertion.
func TestRenderedPages_FreeOfFmtErrorMarkers(t *testing.T) {
	h := fmtDashboard(t)

	for _, route := range []string{
		"/dashboard/",
		"/dashboard/events",
		"/dashboard/aggregates",
		"/dashboard/commands",
		"/dashboard/queries",
		"/dashboard/projections",
		"/dashboard/dead-letters",
		"/dashboard/time-travel",
		"/dashboard/snapshots",
	} {
		req := httptest.NewRequest(http.MethodGet, route, nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("%s: status %d, want 200 (page must render for this guard to mean anything)", route, rec.Code)

			continue
		}

		body := rec.Body.String()
		if idx := strings.Index(body, "%!("); idx >= 0 {
			t.Errorf(
				"%s: fmt error marker in output at %d: %q",
				route,
				idx,
				body[max(0, idx-40):min(len(body), idx+60)],
			)
		}
	}
}
