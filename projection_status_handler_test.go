package cqrshtmx_test

import (
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
)

type mockStatusProvider struct {
	statuses []cqrshtmx.ProjectionStatusEntry
}

func (m *mockStatusProvider) ProjectionStatuses() []cqrshtmx.ProjectionStatusEntry {
	return m.statuses
}

func TestProjectionStatusHandler_Empty(t *testing.T) {
	provider := &mockStatusProvider{statuses: nil}
	handler := cqrshtmx.ProjectionStatusHandler(provider)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/health/projections", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	if ct := w.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/json; charset=utf-8")
	}

	if cc := w.Header().Get("Cache-Control"); cc != "no-cache" {
		t.Errorf("Cache-Control = %q, want %q", cc, "no-cache")
	}

	body := strings.TrimSpace(w.Body.String())
	if body != "null" && body != "[]" {
		t.Errorf("empty status body = %q, want %q or %q", body, "null", "[]")
	}
}

func TestProjectionStatusHandler_RunningState(t *testing.T) {
	provider := &mockStatusProvider{
		statuses: []cqrshtmx.ProjectionStatusEntry{
			{
				Name:       "user-read-model",
				Status:     "live",
				Checkpoint: "evt-123",
				Processed:  5000,
				Errors:     0,
				Restarts:   0,
				LagMillis:  1500,
			},
			{
				Name:       "casbin-projection",
				Status:     "live",
				Checkpoint: "evt-123",
				Processed:  5000,
				Errors:     2,
				Restarts:   1,
				LagMillis:  32,
				LastError:  "connection reset",
			},
		},
	}
	handler := cqrshtmx.ProjectionStatusHandler(provider)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/health/projections", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	body := w.Body.String()
	for _, want := range []string{
		`"user-read-model"`,
		`"casbin-projection"`,
		`"live"`,
		`"processed":5000`,
		`"lastError":"connection reset"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing %q\nbody: %s", want, body)
		}
	}

	var entries []cqrshtmx.ProjectionStatusEntry
	if err := json.Unmarshal(w.Body.Bytes(), &entries); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(entries) != 2 {
		t.Errorf("unmarshaled %d entries, want 2", len(entries))
	}
}

func TestProjectionStatusHandler_FailedState(t *testing.T) {
	provider := &mockStatusProvider{
		statuses: []cqrshtmx.ProjectionStatusEntry{
			{
				Name:      "membership-read-model",
				Status:    "failed",
				Errors:    50,
				Restarts:  5,
				LastError: "schema mismatch on MemberAdded",
			},
		},
	}
	handler := cqrshtmx.ProjectionStatusHandler(provider)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/health/projections", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (handler reports data, not 5xx)", w.Code, http.StatusOK)
	}

	body := w.Body.String()
	if !strings.Contains(body, `"failed"`) {
		t.Errorf("body should contain failed status\nbody: %s", body)
	}

	if !strings.Contains(body, `"schema mismatch on MemberAdded"`) {
		t.Errorf("body should contain last error\nbody: %s", body)
	}
}

func TestProjectionStatusHandler_ETagChangesWithData(t *testing.T) {
	provider := &mockStatusProvider{
		statuses: []cqrshtmx.ProjectionStatusEntry{
			{Name: "test", Status: "live", Processed: 100},
		},
	}
	handler := cqrshtmx.ProjectionStatusHandler(provider)

	first := httptest.NewRecorder()
	handler.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/health/projections", nil))
	firstETag := first.Header().Get("ETag")

	provider.statuses[0].Processed = 200

	second := httptest.NewRecorder()
	handler.ServeHTTP(second, httptest.NewRequest(http.MethodGet, "/health/projections", nil))
	secondETag := second.Header().Get("ETag")

	if firstETag == "" {
		t.Fatal("first response has no ETag")
	}

	if firstETag == secondETag {
		t.Error("ETag should change when projection status data changes")
	}
}

func TestProjectionStatusHandler_304OnMatchingETag(t *testing.T) {
	provider := &mockStatusProvider{
		statuses: []cqrshtmx.ProjectionStatusEntry{
			{Name: "test", Status: "live", Processed: 100},
		},
	}
	handler := cqrshtmx.ProjectionStatusHandler(provider)

	first := httptest.NewRecorder()
	handler.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/health/projections", nil))
	etag := first.Header().Get("ETag")

	second := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health/projections", nil)
	req.Header.Set("If-None-Match", etag)
	handler.ServeHTTP(second, req)

	if second.Code != http.StatusNotModified {
		t.Errorf("status = %d, want %d", second.Code, http.StatusNotModified)
	}

	if second.Body.Len() != 0 {
		t.Errorf("304 body should be empty, got %d bytes", second.Body.Len())
	}
}

func TestProjectionStatusHandler_NilProvider(t *testing.T) {
	handler := cqrshtmx.ProjectionStatusHandler(nil)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/health/projections", nil))

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}
