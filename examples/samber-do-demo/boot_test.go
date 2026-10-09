package main

import (
	"bytes"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestBoot_RouterBuilds exercises main's EXACT mux construction (buildRouter):
// any ServeMux pattern conflict — the Go 1.22+ class that once panicked this
// demo at boot while `go test` stayed green — fails here instead.
func TestBoot_RouterBuilds(t *testing.T) {
	t.Parallel()

	container, cleanup, err := NewContainer(AppConfig{
		Addr:       ":0",
		TOTPIssuer: "boot-test",
	})
	require.NoError(t, err)
	t.Cleanup(cleanup)

	mux, err := buildRouter(container)
	require.NoError(t, err)

	// The index page answers (method-less "/" catch-all).
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())

	// Liveness is dependency-blind and answers immediately.
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())

	// The audit viewer and htmx.js are mounted.
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/htmx.js", nil))
	require.Equal(t, http.StatusOK, rec.Code)

	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/audit/", nil))
	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())

	// Readiness serves the cached batch; wait for the first refresh, then it
	// must report a REAL pass (sync startup drained every projection).
	deadline := time.Now().Add(10 * time.Second)
	for {
		rec = httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))
		if rec.Code == http.StatusOK || time.Now().After(deadline) {
			break
		}

		time.Sleep(50 * time.Millisecond)
	}
	require.Equal(t, http.StatusOK, rec.Code, "readyz never passed: %s", rec.Body.String())
}

// TestBoot_ProcessSmoke runs the compiled binary end-to-end on an ephemeral
// port (PORT env override) and polls readiness over real HTTP: container
// construction, probe start, router build, and the listener all have to work
// for /readyz to answer 200. This is the boot proof `go test` cannot give —
// it never builds main.
func TestBoot_ProcessSmoke(t *testing.T) {
	if testing.Short() {
		t.Skip("process smoke test skipped in -short mode")
	}

	dir, err := os.Getwd()
	require.NoError(t, err)

	bin := filepath.Join(t.TempDir(), "samber-do-demo")
	build := exec.Command("go", "build", "-o", bin, ".")
	build.Dir = dir
	build.Env = append(os.Environ(), "GOEXPERIMENT=jsonv2")
	out, err := build.CombinedOutput()
	require.NoError(t, err, "go build: %s", out)

	// Pick a free port, then hand it to the demo via PORT.
	probeListener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := probeListener.Addr().(*net.TCPAddr).Port
	require.NoError(t, probeListener.Close())

	var serverLog safeBuffer
	run := exec.Command(bin)
	run.Dir = dir
	run.Env = append(os.Environ(), "PORT="+fmt.Sprint(port))
	run.Stdout = &serverLog
	run.Stderr = &serverLog
	require.NoError(t, run.Start())
	t.Cleanup(func() {
		_ = run.Process.Kill()
		_, _ = run.Process.Wait()
	})

	base := fmt.Sprintf("http://127.0.0.1:%d", port)
	client := &http.Client{Timeout: 2 * time.Second}

	var last string
	deadline := time.Now().Add(30 * time.Second)
	for {
		resp, err := client.Get(base + "/readyz")
		if err == nil {
			body := make([]byte, 512)
			n, _ := resp.Body.Read(body)
			_ = resp.Body.Close()
			last = fmt.Sprintf("readyz: %d %s", resp.StatusCode, body[:n])

			if resp.StatusCode == http.StatusOK {
				break
			}
		} else {
			last = err.Error()
		}

		if time.Now().After(deadline) {
			t.Fatalf("demo never became ready on %s (last: %s)\nserver log:\n%s",
				base, last, serverLog.String())
		}

		time.Sleep(100 * time.Millisecond)
	}

	// The index and the audit viewer answer over the real listener too.
	resp, err := client.Get(base + "/")
	require.NoError(t, err)
	_ = resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	resp, err = client.Get(base + "/audit/")
	require.NoError(t, err)
	_ = resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
}

// safeBuffer is a concurrency-safe io.Writer for process output (exec writes
// from a separate goroutine while the test goroutine reads diagnostics).
type safeBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (s *safeBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.buf.Write(p)
}

func (s *safeBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.buf.String()
}
