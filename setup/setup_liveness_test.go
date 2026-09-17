package setup_test

import (
	"encoding/json/v2"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/larsartmann/cqrs-htmx/setup/v4"
)

// M7 of the 2026-09-17 gap-bundle plan: LivePath gives orchestrators a real
// liveness probe (always-200 while serving, opt-in — "" mounts nothing), and
// HealthPath "-" opts the readiness endpoint out for consumers whose own
// health stack owns probing.

func newLivenessBundleServer(t *testing.T, cfg setup.Config) *httptest.Server {
	t.Helper()

	bundle, err := setup.New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = bundle.Close() })

	mux := http.NewServeMux()
	bundle.Mount(mux)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	return server
}

func TestMount_LivePath_Always200JSONWhileServing(t *testing.T) {
	t.Parallel()

	server := newLivenessBundleServer(t, setup.Config{Title: "Live", LivePath: "/health/live-x"})

	resp, err := server.Client().Get(server.URL + "/health/live-x")
	if err != nil {
		t.Fatalf("GET /health/live-x: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("liveness = %d, want 200 (public, no session needed)", resp.StatusCode)
	}

	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Fatalf("liveness content-type = %q, want JSON", ct)
	}

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("liveness must serve JSON, got %.100s: %v", body, err)
	}
}

func TestMount_LivePath_DefaultOff(t *testing.T) {
	t.Parallel()

	server := newLivenessBundleServer(t, setup.Config{Title: "No Liveness"})

	resp, err := server.Client().Get(server.URL + "/health/live-x")
	if err != nil {
		t.Fatalf("GET /health/live-x: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()

	if strings.HasPrefix(strings.TrimSpace(string(body)), "{") {
		t.Error("zero LivePath must not mount a JSON liveness route")
	}
}

func TestMount_HealthPath_DashOptsOut(t *testing.T) {
	t.Parallel()

	server := newLivenessBundleServer(t, setup.Config{Title: "No Health", HealthPath: "-"})

	// The default mount point must NOT exist: "-" is an explicit opt-out, so
	// /health falls through to the login page catch-all (HTML, not the
	// readiness JSON).
	resp, err := server.Client().Get(server.URL + "/health")
	if err != nil {
		t.Fatalf("GET /health: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()

	if strings.HasPrefix(strings.TrimSpace(string(body)), "{") {
		t.Error("HealthPath \"-\" must not mount the readiness endpoint")
	}

	if ct := resp.Header.Get("Content-Type"); strings.Contains(ct, "application/json") {
		t.Errorf("HealthPath \"-\" must not serve readiness JSON, got content-type %q", ct)
	}
}

func TestMount_HealthPath_EmptyStillDefaults(t *testing.T) {
	t.Parallel()

	server := newLivenessBundleServer(t, setup.Config{Title: "Default Health"})

	resp, err := server.Client().Get(server.URL + "/health")
	if err != nil {
		t.Fatalf("GET /health: %v", err)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("default HealthPath must still serve readiness, got %d", resp.StatusCode)
	}
}

func TestNew_LivePath_Validation(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		cfg  setup.Config
		want string
	}{
		{
			name: "missing slash rejected",
			cfg:  setup.Config{Title: "Shape", LivePath: "live"},
			want: "LivePath must start with",
		},
		{
			name: "root rejected",
			cfg:  setup.Config{Title: "Root", LivePath: "/"},
			want: "reserved for the login page",
		},
		{
			name: "health collision rejected",
			cfg:  setup.Config{Title: "Collision", LivePath: "/health"},
			want: "routes would conflict",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := setup.New(tc.cfg)
			if err == nil {
				t.Fatal("expected rejection, got nil")
			}

			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("rejection must mention %q, got: %v", tc.want, err)
			}
		})
	}
}
