package cqrshtmx

import "testing"

// TestTrackerJSSuite runs the zero-dependency Node test suite in
// tracking/jstests against the real embedded tracking/tracker.js: it loads
// the shipped file into a sandbox with a minimal window, document and
// network stub, and drives the signal lifecycle (page views, foreground
// time, clicks, impressions, batching, opt-outs). This is the fast seam for
// pure-JS logic defects; the Playwright lane proves the same tracker in a
// real browser against a real endpoint.
func TestTrackerJSSuite(t *testing.T) {
	runJSSuite(t, "./tracking/jstests/tracker.test.js")
}
