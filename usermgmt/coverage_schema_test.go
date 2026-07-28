package usermgmt

import (
	"context"
	"encoding/json/v2"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
	"github.com/larsartmann/go-cqrs-lite/watermill/v4"
)

// --- writeJSON error path ---

type failingResponseWriter struct {
	header http.Header
}

func (f *failingResponseWriter) Header() http.Header {
	if f.header == nil {
		f.header = http.Header{}
	}
	return f.header
}

func (*failingResponseWriter) WriteHeader(int) {}

func (*failingResponseWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}

func TestWriteJSON_EncodeError(t *testing.T) {
	w := &failingResponseWriter{}
	writeJSON(w, http.StatusOK, make(chan int))
	// http.Error sets Content-Type to text/plain on encode failure
	if ct := w.Header().Get("Content-Type"); ct != "text/plain; charset=utf-8" {
		t.Errorf("expected text/plain Content-Type on encode failure, got %q", ct)
	}
}

func TestWriteJSON_WriteError(t *testing.T) {
	w := &failingResponseWriter{}
	writeJSON(w, http.StatusOK, map[string]string{"ok": "true"})
}

// --- RegisterCommands with nil dispatcher panics ---

func TestRegisterCommands_NilDispatcherPanics(t *testing.T) {
	store := memory.NewMemoryStore()
	bus := watermill.NewEventBus()
	defer func() { _ = bus.Close() }()
	repo, err := decider.NewRepository(store, bus, UserDecider())
	if err != nil {
		t.Fatalf("create repo: %v", err)
	}
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for nil dispatcher")
		}
	}()
	_ = RegisterCommands(nil, repo)
}

// --- DefaultEventSourcedSetup success ---

func TestDefaultEventSourcedSetup_Success(t *testing.T) {
	setup, err := DefaultEventSourcedSetup()
	if err != nil {
		t.Fatalf("DefaultEventSourcedSetup: %v", err)
	}
	if setup.Store == nil || setup.Bus == nil || setup.Repository == nil || setup.ReadModel == nil {
		t.Error("expected all fields non-nil")
	}
	closeBus(setup.Bus)
}

// --- emailFromEvent not found ---

func TestEmailFromEvent_UserNotFound(t *testing.T) {
	svc := newTestService(t)
	aggID := id.NewStreamID()
	evt, err := event.NewEvent(eventUserRegistered, aggID, aggregateTypeUser, 1,
		mustMarshal(t, UserRegisteredPayload{
			Email: "ghost@test.com",
		}))
	if err != nil {
		t.Fatalf("create event: %v", err)
	}
	email := svc.emailFromEvent(evt)
	if email != "" {
		t.Errorf("emailFromEvent for unknown user = %q, want empty", email)
	}
}

// --- Auth handler handleMe unauthorized ---

func TestHandleMe_Unauthorized(t *testing.T) {
	svc := newTestService(t)
	h := NewAuthHandler(svc)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	mux.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

// --- handleDeleteCredential bad encoding ---

func TestHandleDeleteCredential_BadEncoding(t *testing.T) {
	svc := newTestService(t)
	h := NewAuthHandler(svc)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	r := httptest.NewRequest(http.MethodDelete, "/auth/credentials/!!!notbase64!!!", nil)
	r = r.WithContext(WithUser(r.Context(), &User{ID: NewUserID("01HK1549P84T9XF8R94E960633")}))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// --- handler decode error paths (table-driven) ---

func TestHandler_DecodeBadBody(t *testing.T) {
	cases := []struct {
		name string
		path string
		body string
	}{
		{"register", "/auth/register", "{invalid json"},
		{"webauthn register begin", "/auth/webauthn/register/begin", "not json"},
		{"webauthn login begin", "/auth/webauthn/login/begin", "not json"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := newTestService(t)
			h := NewAuthHandler(svc)
			mux := http.NewServeMux()
			h.RegisterRoutes(mux)

			w := postJSON(t, mux, tc.path, tc.body)
			if w.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
			}
		})
	}
}

func mustMarshal(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return b
}

// --- DeleteUser session revocation failure ---

func TestDeleteUser_SessionDeleteFailure(t *testing.T) {
	svc := newTestServiceWithConfig(t, ServiceConfig{
		SessionStore: failingDeleteByUserIDSessionStore("redis down"),
	})
	reg := registerTestUser(t, svc, "ds1", "ds1@test.com")
	if err := svc.DeleteUser(context.Background(), reg.User.ID, "test"); err != nil {
		t.Fatalf("DeleteUser: %v", err)
	}
}

// --- FindByUserID invalid ID ---

func TestFindByUserID_InvalidID(t *testing.T) {
	m := NewUserReadModel()
	_, ok := m.FindByUserID(NewUserID("not-a-valid-id"))
	if ok {
		t.Error("expected false for invalid UserID")
	}
}

// --- writeJSON success path via handler ---

func TestWriteJSON_SuccessViaHandler(t *testing.T) {
	svc := newTestService(t)
	reg := registerTestUser(t, svc, "wj1", "wj1@test.com")
	h := NewAuthHandler(svc)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	r := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	r = r.WithContext(WithUser(r.Context(), reg.User))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if !strings.Contains(w.Body.String(), "wj1@test.com") {
		t.Errorf("response body missing email: %s", w.Body.String())
	}
}

// --- handleListCredentials unauthorized ---

func TestHandleListCredentials_Unauthorized(t *testing.T) {
	svc := newTestService(t)
	h := NewAuthHandler(svc)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	r := httptest.NewRequest(http.MethodGet, "/auth/credentials", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

// --- handleDeleteCredential unauthorized ---

func TestHandleDeleteCredential_Unauthorized(t *testing.T) {
	svc := newTestService(t)
	h := NewAuthHandler(svc)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	r := httptest.NewRequest(http.MethodDelete, "/auth/credentials/abc", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

// --- RegisterCommands all commands wired correctly ---

func TestRegisterCommands_AllWired(t *testing.T) {
	store := memory.NewMemoryStore()
	bus := watermill.NewEventBus()
	defer func() { _ = bus.Close() }()
	repo, err := decider.NewRepository(store, bus, UserDecider())
	if err != nil {
		t.Fatalf("create repo: %v", err)
	}
	disp := command.NewDispatcher()
	if err := RegisterCommands(disp, repo); err != nil {
		t.Fatalf("RegisterCommands: %v", err)
	}

	// Dispatch a RegisterUser command to verify wiring works end-to-end
	aggID := id.NewStreamID()
	err = disp.Dispatch(context.Background(), NewRegisterUserCmd(aggID, "wire@test.com", "Wire", []Role{RoleUser}))
	if err != nil {
		t.Fatalf("dispatch RegisterUser: %v", err)
	}
}
