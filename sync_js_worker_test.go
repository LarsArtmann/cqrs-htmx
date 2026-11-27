package cqrshtmx

import "testing"

// TestSyncWorkerJSSuite runs the zero-dependency Node test suite in
// jstests against the real embedded sync/sync-worker.js: it loads the
// shipped file into a sandbox with a minimal deterministic IndexedDB shim
// and drives the SharedWorker message protocol (queue, batch flush, event
// cache, backend reset). This is the fast seam for pure-JS logic defects:
// both v1.5.0 asset bugs (keyless meta put, reset detection) were only
// catchable by full Playwright batteries before this suite existed.
func TestSyncWorkerJSSuite(t *testing.T) {
	runJSSuite(t, "./sync/jstests/sync-worker.test.js")
}
