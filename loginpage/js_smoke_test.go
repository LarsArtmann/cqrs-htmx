package loginpage

import (
	"os/exec"
	"strings"
	"testing"
)

// TestLoginJSSuite runs the zero-dependency Node test suite in
// internal/jstests against the exported helpers of the embedded
// assets/login.js (Base64URL round-trip property test + serializer
// shape goldens). Skips when node is unavailable so runners without a
// JS runtime stay green; the Playwright e2e lane covers the DOM paths.
func TestLoginJSSuite(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not found in PATH; JS suite skipped")
	}

	cmd := exec.Command( //nolint:gosec // G204: node binary from LookPath above; test-only
		node,
		"--test",
		"--test-reporter=tap",
		"./internal/jstests/login.test.js",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("node --test ./internal/jstests/login.test.js failed: %v\n%s", err, out)
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
