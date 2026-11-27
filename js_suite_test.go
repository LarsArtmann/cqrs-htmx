package cqrshtmx

import (
	"os/exec"
	"strings"
	"testing"
)

// runJSSuite runs one zero-dependency Node test suite (node:test, TAP
// reporter) against a shipped script and fails the Go test when the run
// fails or any subtest fails. It skips when node is unavailable so runners
// without a JS runtime stay green; the Playwright e2e lane covers the
// browser paths.
func runJSSuite(t *testing.T, path string) {
	t.Helper()

	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not found in PATH; JS suite skipped")
	}

	out, err := exec.Command(node, "--test", "--test-reporter=tap", path).CombinedOutput()
	if err != nil {
		t.Fatalf("node --test %s failed: %v\n%s", path, err, out)
	}

	var passed int

	for line := range strings.SplitSeq(string(out), "\n") {
		if strings.HasPrefix(line, "ok ") {
			passed++
		}

		if strings.HasPrefix(line, "not ok ") {
			t.Errorf("JS subtest failed: %s", line)
		}
	}

	t.Logf("node --test %s: %d subtests ok", path, passed)
}
