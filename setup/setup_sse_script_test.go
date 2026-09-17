package setup_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/larsartmann/cqrs-htmx/setup/v4"
)

// M8 of the 2026-09-17 gap-bundle plan: SSEScriptPath serves the HTMX SSE
// extension next to SSEPath (default "/sse.js", "-" opts out) — the same
// treatment DataStarScriptPath gives the DataStar SDK, so HTMX SSE consumers
// self-host nothing.

func newSSEScriptServer(t *testing.T, cfg setup.Config) *httptest.Server {
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

func TestMount_SSEScriptPath_DefaultMountedWithSSEPath(t *testing.T) {
	t.Parallel()

	server := newSSEScriptServer(t, setup.Config{Title: "sse.js default", SSEPath: "/sse"})

	resp, err := server.Client().Get(server.URL + "/sse.js")
	if err != nil {
		t.Fatalf("GET /sse.js: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("sse.js = %d, want 200 (public static script)", resp.StatusCode)
	}

	if resp.Header.Get("Etag") == "" {
		t.Error("sse.js must carry an ETag (immutable versioned script)")
	}

	if !strings.Contains(resp.Header.Get("Content-Type"), "javascript") {
		t.Errorf("sse.js content-type = %q, want JavaScript", resp.Header.Get("Content-Type"))
	}

	if len(body) == 0 {
		t.Error("sse.js must serve the extension payload")
	}
}

func TestMount_SSEScriptPath_DashOptsOut(t *testing.T) {
	t.Parallel()

	server := newSSEScriptServer(t, setup.Config{
		Title:         "sse.js off",
		SSEPath:       "/sse",
		SSEScriptPath: "-",
	})

	resp, err := server.Client().Get(server.URL + "/sse.js")
	if err != nil {
		t.Fatalf("GET /sse.js: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()

	if strings.Contains(resp.Header.Get("Content-Type"), "javascript") {
		t.Error("SSEScriptPath \"-\" must not serve the script")
	}

	if len(body) > 0 && strings.HasPrefix(strings.TrimSpace(string(body)), "//") {
		t.Error("SSEScriptPath \"-\" must not serve the extension payload")
	}
}

func TestMount_SSEScriptPath_AbsentWithoutSSEPath(t *testing.T) {
	t.Parallel()

	server := newSSEScriptServer(t, setup.Config{Title: "no sse"})

	resp, err := server.Client().Get(server.URL + "/sse.js")
	if err != nil {
		t.Fatalf("GET /sse.js: %v", err)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()

	if strings.Contains(resp.Header.Get("Content-Type"), "javascript") {
		t.Error("no SSEPath must not serve the script (field is ignored)")
	}
}

func TestNew_SSEScriptPath_Validation(t *testing.T) {
	t.Parallel()

	_, err := setup.New(setup.Config{Title: "Shape", SSEScriptPath: "sse.js"})
	if err == nil {
		t.Fatal("SSEScriptPath without a leading slash must be rejected")
	}

	if !strings.Contains(err.Error(), "SSEScriptPath") {
		t.Errorf("rejection must name SSEScriptPath, got: %v", err)
	}
}
