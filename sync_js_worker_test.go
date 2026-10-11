package cqrshtmx

import (
	"os/exec"
	"strings"
	"testing"
)

// TestSyncWorkerJSSuite runs the zero-dependency Node test suite in
// jstests against the real embedded sync/sync-worker.js: it loads the
// shipped file into a sandbox with a minimal deterministic IndexedDB shim
// and drives the SharedWorker message protocol (queue, batch flush, event
// cache, backend reset). This is the fast seam for pure-JS logic defects —
// both v1.5.0 asset bugs (keyless meta put, reset detection) were only
// catchable by full Playwright batteries before this suite existed.
// Skips when node is unavailable so runners without a JS runtime stay
// green; the Playwright e2e lane covers the browser paths.
func TestSyncWorkerJSSuite(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not found in PATH; JS suite skipped")
	}

	cmd := exec.Command( //nolint:gosec // G204: node binary from LookPath above; test-only
		node,
		"--test",
		"--test-reporter=tap",
		"./sync/jstests/sync-worker.test.js",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("node --test ./sync/jstests/sync-worker.test.js failed: %v\n%s", err, out)
	}

	var passed int
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "ok ") {
			passed++
		}
		if strings.HasPrefix(line, "not ok ") {
			t.Errorf("JS subtest failed: %s", line)
		}
	}
	t.Logf("node --test: %d subtests ok", passed)
}
