package cqrshtmx_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
)

// TestProjectionReadinessCheck verifies the readiness gate logic: it must pass
// (return nil) only when every projection has reached a terminal drain state
// ("live" or "stopped"), and fail while any is still draining or has failed.
func TestProjectionReadinessCheck(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		nilProvider bool
		statuses    []cqrshtmx.ProjectionStatusEntry
		wantErr     bool
	}{
		{
			name:        "nil_provider_passes",
			nilProvider: true,
			wantErr:     false,
		},
		{
			name:     "empty_statuses_passes",
			statuses: []cqrshtmx.ProjectionStatusEntry{},
			wantErr:  false,
		},
		{
			name: "all_live_passes",
			statuses: []cqrshtmx.ProjectionStatusEntry{
				{Name: "user-read-model", Status: "live"},
				{Name: "casbin-projection", Status: "live"},
			},
			wantErr: false,
		},
		{
			name: "live_and_stopped_passes",
			statuses: []cqrshtmx.ProjectionStatusEntry{
				{Name: "user-read-model", Status: "live"},
				{Name: "casbin-projection", Status: "stopped"},
			},
			wantErr: false,
		},
		{
			name: "running_fails",
			statuses: []cqrshtmx.ProjectionStatusEntry{
				{Name: "user-read-model", Status: "running"},
			},
			wantErr: true,
		},
		{
			name: "idle_fails",
			statuses: []cqrshtmx.ProjectionStatusEntry{
				{Name: "user-read-model", Status: "idle"},
			},
			wantErr: true,
		},
		{
			name: "backoff_fails",
			statuses: []cqrshtmx.ProjectionStatusEntry{
				{Name: "user-read-model", Status: "backoff"},
			},
			wantErr: true,
		},
		{
			name: "draining_fails",
			statuses: []cqrshtmx.ProjectionStatusEntry{
				{Name: "user-read-model", Status: "draining"},
			},
			wantErr: true,
		},
		{
			name: "mixed_live_and_running_fails",
			statuses: []cqrshtmx.ProjectionStatusEntry{
				{Name: "user-read-model", Status: "live"},
				{Name: "casbin-projection", Status: "running"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var provider cqrshtmx.ProjectionStatusProvider
			if !tt.nilProvider {
				provider = &mockStatusProvider{statuses: tt.statuses}
			}

			check := cqrshtmx.ProjectionReadinessCheck(provider)
			err := check.Check()

			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}

			if !tt.wantErr && err != nil {
				t.Fatalf("expected nil, got error: %v", err)
			}
		})
	}
}

// TestProjectionReadinessCheck_FailedPriority verifies that a failed projection
// is reported as a failure (not just draining), so the error message helps
// operators distinguish "still catching up" from "broken".
func TestProjectionReadinessCheck_FailedPriority(t *testing.T) {
	t.Parallel()

	provider := &mockStatusProvider{
		statuses: []cqrshtmx.ProjectionStatusEntry{
			{Name: "user-read-model", Status: "failed", LastError: "disk full"},
		},
	}

	check := cqrshtmx.ProjectionReadinessCheck(provider)

	err := check.Check()
	if err == nil {
		t.Fatal("expected error for failed projection, got nil")
	}

	if !strings.Contains(err.Error(), "failed") {
		t.Errorf("expected 'failed' in error message, got: %s", err.Error())
	}

	if !strings.Contains(err.Error(), "disk full") {
		t.Errorf("expected last error 'disk full' in message, got: %s", err.Error())
	}
}

// TestProjectionDrainReady verifies the exported drain predicate: exactly the
// terminal drain states report ready (the same semantics ProjectionReadinessCheck
// applies per projection), and unknown states fail closed.
func TestProjectionDrainReady(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status string
		want   bool
	}{
		{cqrshtmx.ProjectionStatusLive, true},
		{cqrshtmx.ProjectionStatusStopped, true},
		{cqrshtmx.ProjectionStatusIdle, false},
		{cqrshtmx.ProjectionStatusRunning, false},
		{cqrshtmx.ProjectionStatusBackoff, false},
		{cqrshtmx.ProjectionStatusDraining, false},
		{cqrshtmx.ProjectionStatusFailed, false},
		{"", false},
		{"unknown-state", false},
	}

	for _, tt := range tests {
		if got := cqrshtmx.ProjectionDrainReady(tt.status); got != tt.want {
			t.Errorf("ProjectionDrainReady(%q) = %v, want %v", tt.status, got, tt.want)
		}
	}
}

// TestProjectionStatusConstants pins the wire vocabulary: the exported status
// constants must equal the strings ProjectionStatusHandler serves and
// projectionhost workers report, so health bridges and consumer alerting can
// reference the constants instead of hardcoding strings.
func TestProjectionStatusConstants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		got  string
		want string
	}{
		{cqrshtmx.ProjectionStatusIdle, "idle"},
		{cqrshtmx.ProjectionStatusRunning, "running"},
		{cqrshtmx.ProjectionStatusBackoff, "backoff"},
		{cqrshtmx.ProjectionStatusDraining, "draining"},
		{cqrshtmx.ProjectionStatusLive, "live"},
		{cqrshtmx.ProjectionStatusStopped, "stopped"},
		{cqrshtmx.ProjectionStatusFailed, "failed"},
	}

	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("status constant = %q, want %q", tt.got, tt.want)
		}
	}
}

// TestProjectionReadinessCheck_HandlerHTTP verifies the check wired through
// ReadinessHandler returns 200 when ready and 503 when draining.
func TestProjectionReadinessCheck_HandlerHTTP(t *testing.T) {
	t.Parallel()

	ready := &mockStatusProvider{
		statuses: []cqrshtmx.ProjectionStatusEntry{
			{Name: "user-read-model", Status: "live"},
		},
	}
	handler := cqrshtmx.ReadinessHandler(cqrshtmx.ProjectionReadinessCheck(ready))

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/health", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("ready: status = %d, want %d", w.Code, http.StatusOK)
	}

	draining := &mockStatusProvider{
		statuses: []cqrshtmx.ProjectionStatusEntry{
			{Name: "user-read-model", Status: "running"},
		},
	}
	handler = cqrshtmx.ReadinessHandler(cqrshtmx.ProjectionReadinessCheck(draining))

	w = httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/health", nil))

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("draining: status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}
