package eventtest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// AssertGolden compares got against the golden file at path.
// When update is true, it writes got to path instead of comparing.
// Each caller should register its own flag: `var update = flag.Bool("update", false, "update golden files")`
// and pass *update to this function.
func AssertGolden(t *testing.T, path string, got []byte, update bool) {
	t.Helper()

	if update {
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatalf("mkdir: %v", err)
		}

		if err := os.WriteFile(path, append(got, '\n'), 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}

		return
	}

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %s (run with -update to create): %v", path, err)
	}

	if strings.TrimSpace(string(got)) != strings.TrimSpace(string(want)) {
		t.Errorf("golden mismatch for %s (run with -update to refresh)", path)
	}
}
