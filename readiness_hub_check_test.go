package cqrshtmx

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHubReadinessCheck_HealthyWhileAccepting(t *testing.T) {
	t.Parallel()

	b := NewBroadcaster()
	defer b.Close()

	check := HubReadinessCheck(b)
	if err := check.Check(); err != nil {
		t.Fatalf("expected nil error while hub accepts subscribers: %v", err)
	}

	if check.Name != "sse-hub" {
		t.Errorf("check name: got %q, want %q", check.Name, "sse-hub")
	}
}

func TestHubReadinessCheck_FailsOnceClosed(t *testing.T) {
	t.Parallel()

	b := NewBroadcaster()

	ch := b.Subscribe()
	defer func() { _ = ch }()

	b.Close()

	check := HubReadinessCheck(b)

	err := check.Check()
	if err == nil {
		t.Fatal("expected error after hub Close")
	}

	if !strings.Contains(err.Error(), "closed") || !strings.Contains(err.Error(), "subscribers=") {
		t.Errorf("error should name the state and carry subscriber count: %v", err)
	}
}

func TestReadinessHandler_IncludesHubCheck(t *testing.T) {
	t.Parallel()

	b := NewBroadcaster()
	defer b.Close()

	h := ReadinessHandler(HubReadinessCheck(b))

	rec := httptest.NewRecorder()
	h(rec, httptest.NewRequest(http.MethodGet, "/health", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200\nbody: %s", rec.Code, rec.Body.String())
	}

	if !strings.Contains(rec.Body.String(), `"sse-hub"`) {
		t.Errorf("health body missing sse-hub check\nbody: %s", rec.Body.String())
	}
}
