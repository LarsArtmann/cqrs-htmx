package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// captureStdout swaps os.Stdout for a pipe, runs fn, and returns what it
// printed. The CLI's report functions bind fmt.Print* to os.Stdout at call
// time, so the swap is observed.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	old := os.Stdout

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}

	os.Stdout = w

	done := make(chan string)

	go func() {
		var buf bytes.Buffer

		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()

	defer func() { os.Stdout = old }()

	fn()

	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	out := <-done

	return out
}

// consumerFixture is the R1 consumer source reused by the run/apply tests
// (same shape as the golden round-trip fixture).
const consumerFixture = `package app

import (
	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
)

func wire(mux *http.ServeMux) {
	var cfg cqrshtmx.CSRFConfig
	_ = cqrshtmx.CSRFMiddleware(cfg)
}
`

func TestCollectGoFiles_WalkSkipsSortsAndAcceptsSingleFiles(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, dir, "a.go", "package app")
	writeFile(t, dir, "notes.txt", "not go")
	writeFile(t, dir, filepath.Join("sub", "b.go"), "package sub")
	writeFile(t, dir, filepath.Join("vendor", "v.go"), "package vendor")
	writeFile(t, dir, filepath.Join("testdata", "t.go"), "package testdata")
	writeFile(t, dir, filepath.Join(".git", "g.go"), "package git")

	files, err := collectGoFiles([]string{dir})
	if err != nil {
		t.Fatal(err)
	}

	want := []string{
		filepath.Join(dir, "a.go"),
		filepath.Join(dir, "sub", "b.go"),
	}

	if len(files) != len(want) {
		t.Fatalf("files = %v, want %v (vendor/testdata/.git must be skipped)", files, want)
	}

	for i := range want {
		if files[i] != want[i] {
			t.Errorf("files[%d] = %s, want %s (output must be sorted)", i, files[i], want[i])
		}
	}

	single := filepath.Join(dir, "a.go")

	files, err = collectGoFiles([]string{single})
	if err != nil {
		t.Fatal(err)
	}

	if len(files) != 1 || files[0] != single {
		t.Fatalf("single-file pass-through = %v, want [%s]", files, single)
	}

	nonGo := filepath.Join(dir, "notes.txt")

	files, err = collectGoFiles([]string{nonGo})
	if err != nil {
		t.Fatal(err)
	}

	if len(files) != 0 {
		t.Fatalf("non-.go file must be ignored, got %v", files)
	}

	if _, err := collectGoFiles([]string{filepath.Join(dir, "missing")}); err == nil {
		t.Fatal("nonexistent path must error")
	}
}

func TestRun_DryRunLeavesFilesApplyWrites(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "app.go", consumerFixture)

	n, reports, err := run([]string{dir}, fixtureTable(), false)
	if err != nil {
		t.Fatal(err)
	}

	if n != 1 || len(reports) != 1 {
		t.Fatalf("scanned=%d reports=%d, want 1/1", n, len(reports))
	}

	if len(reports[0].edits) == 0 {
		t.Fatalf("fixture must plan edits, got %+v", reports[0])
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if string(after) != consumerFixture {
		t.Fatal("dry run must not write the file")
	}

	if _, _, err := run([]string{filepath.Join(dir, "missing")}, fixtureTable(), false); err == nil {
		t.Fatal("nonexistent path must error")
	}

	n, reports, err = run([]string{dir}, fixtureTable(), true)
	if err != nil {
		t.Fatal(err)
	}

	if n != 1 || len(reports) != 1 {
		t.Fatalf("apply scanned=%d reports=%d, want 1/1", n, len(reports))
	}

	after, err = os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if string(after) == consumerFixture {
		t.Fatal("apply must rewrite the file")
	}

	if !strings.Contains(string(after), "httputil.CSRFConfig") {
		t.Fatalf("rewritten source must use httputil.CSRFConfig, got:\n%s", after)
	}
}

func TestRunCLI_FixtureEndToEnd(t *testing.T) {
	if _, err := deriveMappings(); err != nil {
		t.Skipf("live module graph unavailable (workspace-mode test): %v", err)
	}

	dir := t.TempDir()
	writeFile(t, dir, "app.go", consumerFixture)

	out := captureStdout(t, func() {
		if err := runCLI([]string{dir}, false, true); err != nil {
			t.Errorf("runCLI: %v", err)
		}
	})

	if !strings.Contains(out, "app.go") {
		t.Fatalf("report must name the scanned file, got:\n%s", out)
	}

	if !strings.Contains(out, "1 file(s) scanned") {
		t.Fatalf("report must carry the summary line, got:\n%s", out)
	}

	if !strings.Contains(out, "dry run") {
		t.Fatalf("default mode must be dry run, got:\n%s", out)
	}
}

func TestPrintReport_Modes(t *testing.T) {
	rep := fileReport{
		path: "app.go",
		edits: []plannedEdit{
			{Rule: "R1", Old: "cqrshtmx.CSRFConfig", New: "httputil.CSRFConfig"},
		},
		manual: []manualFinding{
			{Rule: "R3", Note: "audit .Raw() call sites"},
		},
		applied: "package app\n",
	}

	out := captureStdout(t, func() {
		printReport(nil, false, false)
	})

	if !strings.Contains(out, "nothing to upgrade") {
		t.Fatalf("empty dry-report must print the nothing-to-upgrade line, got:\n%s", out)
	}

	out = captureStdout(t, func() {
		printReport([]fileReport{rep}, false, false)
	})

	if !strings.Contains(out, "app.go:") || !strings.Contains(out, "R1") || !strings.Contains(out, "MANUAL") {
		t.Fatalf("findings must list edits and manual notes, got:\n%s", out)
	}

	if !strings.Contains(out, "re-run with -apply") {
		t.Fatalf("dry run with edits must hint at -apply, got:\n%s", out)
	}

	out = captureStdout(t, func() {
		printReport([]fileReport{rep}, true, true)
	})

	if !strings.Contains(out, "APPLY") {
		t.Fatalf("apply mode must be announced, got:\n%s", out)
	}

	if strings.Contains(out, "nothing to upgrade") || strings.Contains(out, "re-run with -apply") {
		t.Fatalf("apply/verbose modes must not carry dry-run hints, got:\n%s", out)
	}
}

func TestRootFileClassLists(t *testing.T) {
	httputilFiles := rootHTTputilFiles()
	if len(httputilFiles) == 0 {
		t.Fatal("rootHTTputilFiles must list the class-1 re-export files")
	}

	for _, f := range httputilFiles {
		if !strings.HasSuffix(f, ".go") {
			t.Errorf("rootHTTputilFiles entries must be .go names, got %q", f)
		}
	}

	sseFiles := rootGoSSEFiles()
	for _, want := range []string{"sse_event.go", "sse_store.go", "event_store_sse.go"} {
		if !slices.Contains(sseFiles, want) {
			t.Errorf("rootGoSSEFiles must contain %q, got %v", want, sseFiles)
		}
	}
}
