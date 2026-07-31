package cqrshtmx

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestReadinessHandler_AllPass(t *testing.T) {
	t.Parallel()

	handler := ReadinessHandler(
		NewNamedCheck("a", func() error { return nil }),
		NewNamedCheck("b", func() error { return nil }),
	)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/ready", nil))

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestReadinessHandler_OneFails(t *testing.T) {
	t.Parallel()

	handler := ReadinessHandler(
		NewNamedCheck("ok", func() error { return nil }),
		NewNamedCheck("bad", func() error { return errors.New("connection refused") }),
	)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/ready", nil))

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", rr.Code)
	}

	body := rr.Body.String()
	if !contains(body, "bad") {
		t.Errorf("expected body to contain failing check name 'bad', got: %s", body)
	}
	if !contains(body, "connection refused") {
		t.Errorf("expected body to contain error message, got: %s", body)
	}
}

func TestReadinessHandler_NoChecks(t *testing.T) {
	t.Parallel()

	handler := ReadinessHandler()

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/ready", nil))

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 with no checks, got %d", rr.Code)
	}
}

func TestDebugHandler_ReturnsJSON(t *testing.T) {
	t.Parallel()

	handler := DebugHandler(map[string]any{
		"version": "1.0.0",
		"ok":      true,
	})

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/debug", nil))

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	body := rr.Body.String()
	if !contains(body, "1.0.0") {
		t.Errorf("expected body to contain '1.0.0', got: %s", body)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
