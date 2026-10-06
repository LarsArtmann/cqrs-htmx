package dashboardui

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/larsartmann/httputil"
)

func TestWithActorAndActorFromContext(t *testing.T) {
	ctx := context.Background()

	if _, ok := ActorFromContext(ctx); ok {
		t.Fatal("empty context should report no actor")
	}

	want := Actor{ID: "user-1", Name: "ops@example.com"}
	ctx = WithActor(ctx, want)

	got, ok := ActorFromContext(ctx)
	if !ok {
		t.Fatal("context with actor should report ok")
	}

	if got != want {
		t.Errorf("ActorFromContext() = %+v; want %+v", got, want)
	}
}

func TestGuard_ActorAuthorizerDenied(t *testing.T) {
	d := mustTestDashboardWithConfig(t, Config{
		Journal: &stubJournal{},
		ActorAuthorizer: func(*http.Request) (Actor, error) {
			return Actor{}, errors.New("denied")
		},
	})

	called := false
	h := d.guard(func(http.ResponseWriter, *http.Request) { called = true })

	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

	if called {
		t.Fatal("handler must not run when the actor authorizer denies")
	}

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

func TestGuard_ActorAuthorizerInjectsActor(t *testing.T) {
	want := Actor{ID: "user-7", Name: "admin@example.com"}
	d := mustTestDashboardWithConfig(t, Config{
		Journal: &stubJournal{},
		ActorAuthorizer: func(*http.Request) (Actor, error) {
			return want, nil
		},
	})

	var got Actor

	var ok bool

	h := d.guard(func(_ http.ResponseWriter, r *http.Request) {
		got, ok = ActorFromContext(r.Context())
	})

	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	if !ok {
		t.Fatal("handler should see the actor in its request context")
	}

	if got != want {
		t.Errorf("actor in handler context = %+v; want %+v", got, want)
	}
}

func TestGuard_ActorAuthorizerTakesPrecedence(t *testing.T) {
	legacyCalled := false
	d := mustTestDashboardWithConfig(t, Config{
		Journal: &stubJournal{},
		Authorizer: func(*http.Request) error {
			legacyCalled = true

			return nil
		},
		ActorAuthorizer: func(*http.Request) (Actor, error) {
			return Actor{ID: "actor"}, nil
		},
	})

	h := d.guard(func(http.ResponseWriter, *http.Request) {})
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	if legacyCalled {
		t.Fatal("legacy Authorizer must not run when ActorAuthorizer is set")
	}
}

// recordingHandler captures slog records so tests can assert audit attrs.
type recordingHandler struct {
	mu      sync.Mutex
	records []slog.Record
}

func (h *recordingHandler) Enabled(context.Context, slog.Level) bool { return true }

func (h *recordingHandler) Handle(_ context.Context, record slog.Record) error {
	h.mu.Lock()

	defer h.mu.Unlock()

	h.records = append(h.records, record.Clone())

	return nil
}

func (h *recordingHandler) WithAttrs([]slog.Attr) slog.Handler { return h }

func (h *recordingHandler) WithGroup(string) slog.Handler { return h }

func (h *recordingHandler) auditRecords() []slog.Record {
	h.mu.Lock()
	defer h.mu.Unlock()

	return append([]slog.Record(nil), h.records...)
}

func TestAudit_RecordsActorAndRequestID(t *testing.T) {
	recorder := &recordingHandler{}

	previous := slog.Default()

	slog.SetDefault(slog.New(recorder))

	defer slog.SetDefault(previous)

	store := &fakeSnapshotStore{}
	d := mustTestDashboardWithConfig(t, Config{
		Journal:       &stubJournal{},
		SnapshotStore: store,
		ActorAuthorizer: func(*http.Request) (Actor, error) {
			return Actor{ID: "user-42", Name: "ops@example.com"}, nil
		},
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/snapshots/user/abc/delete", nil)
	r.SetPathValue("type", "user")
	r.SetPathValue("id", "abc")

	httputil.RequestID(httputil.DefaultRequestIDConfig())(
		http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			d.guard(d.snapshotDeleteHandler)(response, request)
		}),
	).ServeHTTP(w, r)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("snapshot delete status = %d; want 303; body=%s", w.Code, w.Body.String())
	}

	audits := auditRecords(recorder)
	if len(audits) != 1 {
		t.Fatalf("expected exactly one audit record, got %d", len(audits))
	}

	attrs := attrMap(audits[0])

	if attrs["op"] != "snapshot.delete" {
		t.Errorf("audit op = %v; want snapshot.delete", attrs["op"])
	}

	if attrs["actor_id"] != "user-42" {
		t.Errorf("audit actor_id = %v; want user-42", attrs["actor_id"])
	}

	if attrs["actor_name"] != "ops@example.com" {
		t.Errorf("audit actor_name = %v; want ops@example.com", attrs["actor_name"])
	}

	if attrs["result"] != "ok" {
		t.Errorf("audit result = %v; want ok", attrs["result"])
	}

	if requestID, ok := attrs["request_id"].(string); !ok || requestID == "" {
		t.Errorf("audit request_id = %v; want the middleware-generated non-empty ID", attrs["request_id"])
	}
}

func TestAudit_AnonymousWithoutActor(t *testing.T) {
	recorder := &recordingHandler{}

	previous := slog.Default()

	slog.SetDefault(slog.New(recorder))

	defer slog.SetDefault(previous)

	store := &fakeSnapshotStore{}
	d := mustTestDashboardWithConfig(t, Config{
		Journal:       &stubJournal{},
		SnapshotStore: store,
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/snapshots/user/abc/delete", nil)
	r.SetPathValue("type", "user")
	r.SetPathValue("id", "abc")
	d.snapshotDeleteHandler(w, r)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("snapshot delete status = %d; want 303", w.Code)
	}

	audits := auditRecords(recorder)
	if len(audits) != 1 {
		t.Fatalf("expected exactly one audit record, got %d", len(audits))
	}

	attrs := attrMap(audits[0])

	if attrs["actor"] != "anonymous" {
		t.Errorf("audit actor = %v; want anonymous", attrs["actor"])
	}

	if _, hasRequestID := attrs["request_id"]; hasRequestID {
		t.Errorf("audit should carry no request_id without the middleware, got %v", attrs["request_id"])
	}
}

func auditRecords(recorder *recordingHandler) []slog.Record {
	var audits []slog.Record

	for _, record := range recorder.auditRecords() {
		if record.Message == "dashboardui.audit" {
			audits = append(audits, record)
		}
	}

	return audits
}

func attrMap(record slog.Record) map[string]any {
	attrs := make(map[string]any)

	record.Attrs(func(a slog.Attr) bool {
		attrs[a.Key] = a.Value.Any()

		return true
	})

	return attrs
}
