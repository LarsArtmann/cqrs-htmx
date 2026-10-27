package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fixtureTable builds a small explicit rulesTable for the rewrite tests
// (mirrors what deriveMappings derives from the real alias files).
func fixtureTable() rulesTable {
	return rulesTable{
		rootImportPath: {
			"CSRFMiddleware": {targetImportPath: httputilImportPath, targetName: "CSRFMiddleware", rule: "R1"},
			"CSRFConfig":     {targetImportPath: httputilImportPath, targetName: "CSRFConfig", rule: "R1"},
			"CSRFToken":      {targetImportPath: httputilImportPath, targetName: "CSRFToken", rule: "R10"},
			"SSEEvent":       {targetImportPath: gosseImportPath, targetName: "Event", rule: "R2"},
		},
		usermgmtImportPath: {
			"UserRegistered": {targetImportPath: identityImportPath, targetName: "UserRegistered", rule: "R9"},
			"EventUserRegistered": {
				targetImportPath: identityImportPath,
				targetName:       "EventUserRegistered",
				rule:             "R10",
			},
		},
	}
}

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()

	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	return path
}

// TestR1_RequalifiesAndSwapsImport: a consumer using ONLY deprecated
// re-exports gets every reference requalified and the import swapped.
func TestR1_RequalifiesAndSwapsImport(t *testing.T) {
	dir := t.TempDir()

	src := `package app

import (
	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
)

func wire(mux *http.ServeMux) {
	var cfg cqrshtmx.CSRFConfig
	_ = cqrshtmx.CSRFMiddleware(cfg)
}
`
	path := writeFile(t, dir, "app.go", src)

	rep, err := planFile(path, fixtureTable())
	if err != nil {
		t.Fatal(err)
	}

	if len(rep.edits) != 2 {
		t.Fatalf("edits = %d (%+v), want 2 symbol requalifications", len(rep.edits), rep.edits)
	}

	want := `package app

import (
	"github.com/larsartmann/httputil"
)

func wire(mux *http.ServeMux) {
	var cfg httputil.CSRFConfig
	_ = httputil.CSRFMiddleware(cfg)
}
`
	if rep.applied != want {
		t.Errorf("applied:\n%s\nwant:\n%s", rep.applied, want)
	}

	for _, e := range rep.edits {
		if e.Rule != "R1" {
			t.Errorf("expected only R1-labeled edits, got %+v", rep.edits)

			break
		}
	}
}

// TestR1_KeepsImportWhenLiveSymbolsRemain: the source import stays when a
// non-deprecated symbol still references it.
func TestR1_KeepsImportWhenLiveSymbolsRemain(t *testing.T) {
	dir := t.TempDir()

	src := `package app

import (
	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
)

func wire(mux *http.ServeMux) {
	mux.Handle("/x", cqrshtmx.CSRFMiddleware(cqrshtmx.CSRFConfig{}))
	mux.Handle("/y", cqrshtmx.HTMXScriptHandler())
}
`
	path := writeFile(t, dir, "app.go", src)

	rep, err := planFile(path, fixtureTable())
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(rep.applied, `cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"`) {
		t.Errorf("source import must stay (HTMXScriptHandler remains):\n%s", rep.applied)
	}

	if !strings.Contains(rep.applied, `"github.com/larsartmann/httputil"`) {
		t.Errorf("target import must be added:\n%s", rep.applied)
	}

	if !strings.Contains(rep.applied, "httputil.CSRFMiddleware(httputil.CSRFConfig{})") {
		t.Errorf("references must be requalified:\n%s", rep.applied)
	}

	if !strings.Contains(rep.applied, "cqrshtmx.HTMXScriptHandler()") {
		t.Errorf("live symbol must be untouched:\n%s", rep.applied)
	}
}

// TestR2_RenamesTargetSymbol: SSEEvent -> sse.Event requalification with the
// target qualifier from the file's own existing sse import, if present.
func TestR2_RenamesTargetSymbol(t *testing.T) {
	dir := t.TempDir()

	src := `package app

import (
	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
)

func emit() cqrshtmx.SSEEvent {
	return cqrshtmx.SSEEvent{}
}
`
	path := writeFile(t, dir, "app.go", src)

	rep, err := planFile(path, fixtureTable())
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(rep.applied, "func emit() sse.Event {") ||
		!strings.Contains(rep.applied, "return sse.Event{}") {
		t.Errorf("SSEEvent must become sse.Event:\n%s", rep.applied)
	}

	if !strings.Contains(rep.applied, `"github.com/larsartmann/go-sse"`) {
		t.Errorf("go-sse import must be added:\n%s", rep.applied)
	}
}

// TestR3_ConstructorRenameAndRawReport: NewBroadcasterFromRaw renames;
// a .Raw() call is a MANUAL finding, never rewritten.
func TestR3_ConstructorRenameAndRawReport(t *testing.T) {
	dir := t.TempDir()

	src := `package app

import (
	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
)

var b = cqrshtmx.NewBroadcasterFromRaw(nil)

func raw() any { return b.Raw() }
`
	path := writeFile(t, dir, "app.go", src)

	rep, err := planFile(path, fixtureTable())
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(rep.applied, "cqrshtmx.NewBroadcasterFromHub(nil)") {
		t.Errorf("constructor must rename in place:\n%s", rep.applied)
	}

	if strings.Contains(rep.applied, ".Hub()") {
		t.Errorf("Raw() must NOT be auto-rewritten:\n%s", rep.applied)
	}

	foundManual := false

	for _, m := range rep.manual {
		if m.Rule == "R3" && strings.Contains(m.Note, ".Raw()") {
			foundManual = true
		}
	}

	if !foundManual {
		t.Errorf("expected an R3 manual finding for .Raw(), got %+v", rep.manual)
	}
}

// TestR9_R10_UsermgmtAliasRequalify: identity-model requalification with the
// canonical identitymodel qualifier.
func TestR9_R10_UsermgmtAliasRequalify(t *testing.T) {
	dir := t.TempDir()

	src := `package app

import (
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
)

var kinds = []string{usermgmt.UserRegistered{}.Type()}
var evtType = usermgmt.EventUserRegistered
`
	path := writeFile(t, dir, "app.go", src)

	rep, err := planFile(path, fixtureTable())
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(rep.applied, "identitymodel.UserRegistered{}") ||
		!strings.Contains(rep.applied, "identitymodel.EventUserRegistered") {
		t.Errorf("aliases must requalify to identitymodel:\n%s", rep.applied)
	}

	if !strings.Contains(rep.applied, `"github.com/larsartmann/cqrs-htmx/identity-model/v4"`) {
		t.Errorf("identity-model import must be added:\n%s", rep.applied)
	}

	if strings.Contains(rep.applied, "usermgmt/v4") {
		t.Errorf("fully-migrated source import must be dropped:\n%s", rep.applied)
	}
}

// TestAliasedConsumerImport: consumers that import the source package under
// a custom name rewrite identically.
func TestAliasedConsumerImport(t *testing.T) {
	dir := t.TempDir()

	src := `package app

import (
	ch "github.com/larsartmann/cqrs-htmx/v4"
)

var cfg = ch.CSRFConfig{}
`
	path := writeFile(t, dir, "app.go", src)

	rep, err := planFile(path, fixtureTable())
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(rep.applied, "httputil.CSRFConfig{}") {
		t.Errorf("aliased import must still requalify:\n%s", rep.applied)
	}
}

// TestPlanFile_UnrelatedFileUntouched: files with no deprecated references
// produce no edits and no applied text.
func TestPlanFile_UnrelatedFileUntouched(t *testing.T) {
	dir := t.TempDir()

	path := writeFile(t, dir, "plain.go", "package app\n\nvar x = 1\n")

	rep, err := planFile(path, fixtureTable())
	if err != nil {
		t.Fatal(err)
	}

	if len(rep.edits) != 0 || len(rep.manual) != 0 || rep.applied != "" {
		t.Errorf("expected untouched file, got %+v", rep)
	}
}

// TestDeriveFromFiles_FixtureAliasFiles pins the derivation itself: type
// aliases report R9, value re-exports R10, non-alias decls are ignored.
func TestDeriveFromFiles_FixtureAliasFiles(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, dir, "es_fixture.go", `package usermgmt

import (
	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
)

type (
	UserRegistered = identitymodel.UserRegistered
	// NOT an alias: a defined type — must be skipped.
	Service struct{ core int }
)

var (
	EventUserRegistered = identitymodel.EventUserRegistered
)
`)

	table, err := deriveFromFiles(dir, []string{"es_fixture.go"}, identityImportPath, "R9")
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := table["UserRegistered"]; !ok {
		t.Errorf("type alias missing from derived table: %+v", table)
	}

	if got, ok := table["UserRegistered"]; !ok || got.rule != "R9" {
		t.Errorf("type alias rule = %+v (ok=%v), want R9", got, ok)
	}

	if got, ok := table["EventUserRegistered"]; !ok || got.rule != "R10" {
		t.Errorf("value re-export rule = %+v (ok=%v), want R10", got, ok)
	}

	if _, ok := table["Service"]; ok {
		t.Errorf("non-alias defined type must not enter the table: %+v", table)
	}
}

// TestGoldenConsumer_R1RoundTrip is the plan's R1 golden: a realistic
// consumer fixture rewrites to exactly the golden output (round-trips
// end-to-end through planFile).
func TestGoldenConsumer_R1RoundTrip(t *testing.T) {
	dir := t.TempDir()

	input, err := os.ReadFile(filepath.Join("testdata", "consumer_r1_in.go.txt"))
	if err != nil {
		t.Fatal(err)
	}

	golden, err := os.ReadFile(filepath.Join("testdata", "consumer_r1_want.go.txt"))
	if err != nil {
		t.Fatal(err)
	}

	path := writeFile(t, dir, "consumer.go", string(input))

	rep, err := planFile(path, fixtureTable())
	if err != nil {
		t.Fatal(err)
	}

	if rep.applied != string(golden) {
		t.Errorf("golden round-trip drifted:\n--- got ---\n%s\n--- want ---\n%s", rep.applied, golden)
	}
}

// TestDeriveMappings_LiveModules proves the derivation against the REAL
// module tree (via `go list -m`): the root re-export tables and the
// usermgmt identity-model layer must derive with their documented scale
// (class 1 ~39 symbols; class 3 ~161 symbols per the v5 removal inventory).
func TestDeriveMappings_LiveModules(t *testing.T) {
	table, err := deriveMappings()
	if err != nil {
		t.Skipf("live module graph unavailable (go list -m failed): %v", err)
	}

	root := table[rootImportPath]
	if len(root) < 25 {
		t.Errorf(
			"derived root table has %d symbols, want >= 25 (class 1 re-exports; exact count floats with the library version)",
			len(root),
		)
	}

	if _, ok := root["CSRFMiddleware"]; !ok {
		t.Error("CSRFMiddleware missing from the derived R1 table")
	}

	identity := table[usermgmtImportPath]
	if len(identity) < 100 {
		t.Errorf("derived usermgmt table has %d symbols, want >= 100 (class 3 alias layer)", len(identity))
	}

	r9, r10 := 0, 0

	for _, m := range identity {
		switch m.rule {
		case "R9":
			r9++
		case "R10":
			r10++
		}
	}

	if r9 == 0 || r10 == 0 {
		t.Errorf("expected both R9 (types, got %d) and R10 (values, got %d) in the usermgmt table", r9, r10)
	}
}

// TestR1_SingleLineImportStaysParseable: a file whose ONLY source-package
// import is a single-line (unparenthesized) `import "..."` must come out
// parseable — the fuse path used to emit the bare spec line, dropping the
// `import` keyword and corrupting the consumer file.
func TestR1_SingleLineImportStaysParseable(t *testing.T) {
	dir := t.TempDir()

	src := `package app

import "github.com/larsartmann/cqrs-htmx/v4"

func wire(mux *http.ServeMux) {
	var cfg cqrshtmx.CSRFConfig
	_ = cqrshtmx.CSRFMiddleware(cfg)
}
`
	path := writeFile(t, dir, "app.go", src)

	rep, err := planFile(path, fixtureTable())
	if err != nil {
		t.Fatal(err)
	}

	want := `package app

import (
	"github.com/larsartmann/httputil"
)

func wire(mux *http.ServeMux) {
	var cfg httputil.CSRFConfig
	_ = httputil.CSRFMiddleware(cfg)
}
`
	if rep.applied != want {
		t.Errorf("applied:\n%s\nwant:\n%s", rep.applied, want)
	}
}

// TestR1_BlankImportIsNeverDropped: `_` imports carry side-effect
// registration — the rewrite must never remove their line.
func TestR1_BlankImportIsNeverDropped(t *testing.T) {
	dir := t.TempDir()

	src := `package app

import (
	_ "github.com/larsartmann/cqrs-htmx/v4"
	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
)

func wire() {
	var cfg cqrshtmx.CSRFConfig
	_ = cqrshtmx.CSRFMiddleware(cfg)
}
`
	path := writeFile(t, dir, "app.go", src)

	rep, err := planFile(path, fixtureTable())
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(rep.applied, `_ "github.com/larsartmann/cqrs-htmx/v4"`) {
		t.Errorf("blank import was dropped or rewritten:\n%s", rep.applied)
	}
}
