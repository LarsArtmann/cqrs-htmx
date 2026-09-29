package setup_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/larsartmann/cqrs-htmx/setup/v4"
	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// PapDashboard feedback item 3 (2026-09-29): health as injectable checks.
// Config.HealthChecks appends consumer-owned checks to the mounted readiness
// endpoint so an app keeps ONE health surface instead of running setup's
// /health and its own probe on the same port.

// TestHealthChecks_ConsumerCheckReportsInMountedHealth asserts a passing
// consumer check appears in the /health body alongside the built-ins and
// keeps the endpoint 200.
func TestHealthChecks_ConsumerCheckReportsInMountedHealth(t *testing.T) {
	t.Parallel()

	bundle := setup.MustNew(setup.Config{
		Title: "Injectable Health",
		HealthChecks: []cqrshtmx.NamedCheck{
			cqrshtmx.NewNamedCheck("event-store", func() error { return nil }),
		},
	})
	defer func() { _ = bundle.Close() }()

	mux := http.NewServeMux()
	server := httptest.NewServer(bundle.Handler(mux))
	defer server.Close()

	resp, err := server.Client().Get(server.URL + "/health")
	if err != nil {
		t.Fatalf("GET /health: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("/health must stay 200 when every check passes, got %d (body: %s)", resp.StatusCode, body)
	}

	for _, name := range []string{"projections", "event-store"} {
		if !strings.Contains(string(body), "\""+name+"\"") {
			t.Errorf("health body must report the %q check, got: %s", name, body)
		}
	}
}

// TestHealthChecks_FailingCheckFlipsReadinessTo503 asserts a failing consumer
// check degrades the SAME endpoint the built-ins use — one probe surface, one
// failure semantic.
func TestHealthChecks_FailingCheckFlipsReadinessTo503(t *testing.T) {
	t.Parallel()

	bundle := setup.MustNew(setup.Config{
		Title: "Degrade",
		HealthChecks: []cqrshtmx.NamedCheck{
			cqrshtmx.NewNamedCheck("upstream-db", func() error {
				return errorfamily.NewRejection("test.upstream_down", "connection refused")
			}),
		},
	})
	defer func() { _ = bundle.Close() }()

	mux := http.NewServeMux()
	server := httptest.NewServer(bundle.Handler(mux))
	defer server.Close()

	resp, err := server.Client().Get(server.URL + "/health")
	if err != nil {
		t.Fatalf("GET /health: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("failing consumer check must flip /health to 503, got %d (body: %s)", resp.StatusCode, body)
	}

	if !strings.Contains(string(body), "connection refused") {
		t.Errorf("health body must carry the failing check's error, got: %s", body)
	}
}

// TestHealthChecks_ValidationRejectsBadEntries pins the fail-fast contract:
// duplicate names (also against the built-ins), empty names, and nil check
// functions abort New — the readiness body keys results by name, and a nil
// Check would panic at probe time.
func TestHealthChecks_ValidationRejectsBadEntries(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		config setup.Config
		wantIn string
	}{
		{
			name: "duplicate consumer name",
			config: setup.Config{HealthChecks: []cqrshtmx.NamedCheck{
				cqrshtmx.NewNamedCheck("dup", func() error { return nil }),
				cqrshtmx.NewNamedCheck("dup", func() error { return nil }),
			}},
			wantIn: `"dup"`,
		},
		{
			name: "built-in name collision",
			config: setup.Config{HealthChecks: []cqrshtmx.NamedCheck{
				cqrshtmx.NewNamedCheck("projections", func() error { return nil }),
			}},
			wantIn: `"projections"`,
		},
		{
			name: "empty name",
			config: setup.Config{HealthChecks: []cqrshtmx.NamedCheck{
				{Name: "", Check: func() error { return nil }},
			}},
			wantIn: "non-empty Name",
		},
		{
			name: "nil check",
			config: setup.Config{HealthChecks: []cqrshtmx.NamedCheck{
				{Name: "hollow"},
			}},
			wantIn: "non-nil Check",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tt.config.Title = "Validation"

			bundle, err := setup.New(tt.config)
			if err == nil {
				_ = bundle.Close()

				t.Fatal("expected New to reject the config, got nil error")
			}

			if !strings.Contains(err.Error(), tt.wantIn) {
				t.Errorf("error must mention %q, got: %v", tt.wantIn, err)
			}
		})
	}
}

// TestHealthChecks_NoChecksKeepsBuiltInsOnly pins the zero value: without
// HealthChecks, /health reports exactly the built-ins — byte-identical
// behavior to before the seam existed.
func TestHealthChecks_NoChecksKeepsBuiltInsOnly(t *testing.T) {
	t.Parallel()

	bundle := setup.MustNew(setup.Config{Title: "Zero Value"})
	defer func() { _ = bundle.Close() }()

	mux := http.NewServeMux()
	server := httptest.NewServer(bundle.Handler(mux))
	defer server.Close()

	resp, err := server.Client().Get(server.URL + "/health")
	if err != nil {
		t.Fatalf("GET /health: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)

	if !strings.Contains(string(body), "\"projections\"") {
		t.Errorf("built-in projection check must still report, got: %s", body)
	}

	if strings.Contains(string(body), "event-store") {
		t.Errorf("no consumer checks must be present, got: %s", body)
	}
}
