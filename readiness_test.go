package cqrshtmx

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
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

	if !strings.Contains(body, "bad") {
		t.Errorf("expected body to contain failing check name 'bad', got: %s", body)
	}

	if !strings.Contains(body, "connection refused") {
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

func TestReadinessHandler_HungCheckTimesOut(t *testing.T) {
	t.Parallel()

	release := make(chan struct{})
	defer close(release) // unblock the abandoned check goroutine

	handler := ReadinessHandler(
		NamedCheck{Name: "ok", Check: func() error { return nil }, Timeout: time.Second},
		NamedCheck{Name: "hung", Check: func() error {
			<-release

			return nil
		}, Timeout: 10 * time.Millisecond},
	)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/ready", nil))

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503 when a check hangs past its timeout, got %d", rr.Code)
	}

	body := rr.Body.String()

	if !strings.Contains(body, "hung") {
		t.Errorf("expected body to name the timed-out check 'hung', got: %s", body)
	}

	if !strings.Contains(body, "hung: timed out after 10ms") {
		t.Errorf("expected body to name check and timeout, got: %s", body)
	}

	if !strings.Contains(body, `"ok":{"status":"ok"}`) {
		t.Errorf("expected the healthy check to still be reported ok, got: %s", body)
	}
}

func TestReadinessHandler_TimedCheckReportsOwnError(t *testing.T) {
	t.Parallel()

	handler := ReadinessHandler(
		NamedCheck{Name: "bad", Check: func() error { return errors.New("boom") }, Timeout: 5 * time.Second},
	)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/ready", nil))

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", rr.Code)
	}

	body := rr.Body.String()

	if !strings.Contains(body, "boom") {
		t.Errorf("expected the check's own error, got: %s", body)
	}

	if strings.Contains(body, "timed out") {
		t.Errorf("a check that fails in time must not be reported as timed out, got: %s", body)
	}
}

func TestReadinessHandler_ZeroTimeoutStaysUnbounded(t *testing.T) {
	t.Parallel()

	handler := ReadinessHandler(
		NewNamedCheck("slow", func() error {
			time.Sleep(50 * time.Millisecond)

			return nil
		}),
	)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/ready", nil))

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	if body := rr.Body.String(); strings.Contains(body, "timed out") {
		t.Errorf("zero Timeout must not impose an implicit deadline, got: %s", body)
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

	if !strings.Contains(body, "1.0.0") {
		t.Errorf("expected body to contain '1.0.0', got: %s", body)
	}
}
