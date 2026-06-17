package cataloghtmx_test

import (
	"encoding/json"
	"flag"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	cataloghtmx "github.com/larsartmann/cqrs-htmx/catalog/v2"
	"github.com/larsartmann/go-cqrs-lite/catalog/v2"
)

// updateGolden regenerates the golden files under testdata/ when set.
// Usage: go test ./... -count=1 -update-golden
//
//nolint:gochecknoglobals // standard Go golden-file update flag
var updateGolden = flag.Bool("update-golden", false, "regenerate golden files in testdata/")

// goldenCatalog returns a fixed, deterministic catalog used as the basis for
// golden output comparison. Changing the types or the registration below will
// require running with -update-golden to refresh the expected output.
func goldenCatalog() *cataloghtmx.Builder {
	b := cataloghtmx.New("Golden Service", "1.0.0", cataloghtmx.WithServiceID("golden-svc"))
	cataloghtmx.Command[testCmd](
		b, "create-thing",
		cataloghtmx.WithOperation("POST", "/api/things"),
	)
	cataloghtmx.Query[testQuery](
		b, "get-thing",
		cataloghtmx.WithOperation("GET", "/api/things/{id}"),
	)
	cataloghtmx.Event[testEvent](b, "thing.created", catalog.Sends)
	return b
}

func serveBody(t *testing.T, h http.HandlerFunc) []byte {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200 from handler, got %d", w.Code)
	}

	return w.Body.Bytes()
}

// canonicalJSON pretty-prints JSON with sorted keys so that map iteration order
// does not make the golden output non-deterministic.
func canonicalJSON(t *testing.T, raw []byte) []byte {
	t.Helper()

	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatalf("invalid JSON for canonicalization: %v", err)
	}

	pretty, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatalf("failed to re-marshal canonical JSON: %v", err)
	}

	return pretty
}

// compareGolden checks the output against a golden file under testdata/.
// Both the handler output and the golden file content are canonicalized
// via canonicalJSON before comparison, making tests immune to formatter
// whitespace changes from pre-commit hooks (oxfmt, etc.).
func compareGolden(t *testing.T, name string, got []byte) {
	t.Helper()

	goldenPath := filepath.Join("testdata", name)

	if *updateGolden {
		canonical := canonicalJSON(t, got)

		if err := os.MkdirAll("testdata", 0o750); err != nil {
			t.Fatalf("mkdir testdata: %v", err)
		}

		if err := os.WriteFile(goldenPath, append(canonical, '\n'), 0o600); err != nil {
			t.Fatalf("write golden %s: %v", goldenPath, err)
		}

		t.Logf("updated golden file: %s", goldenPath)

		return
	}

	// Path is constructed from test-internal constants, not user input.
	rawWant, err := os.ReadFile(goldenPath) //nolint:gosec // G304: test-controlled path
	if err != nil {
		t.Fatalf("golden file %s is missing (run `go test ./... -count=1 -update-golden` to create): %v",
			goldenPath, err)
	}

	// Canonicalize BOTH sides so formatter changes to the golden file don't
	// cause false failures.
	want := string(canonicalJSON(t, rawWant))
	gotCanonical := string(canonicalJSON(t, got))

	if want != gotCanonical {
		t.Errorf("golden output mismatch for %s\n"+
			"--- expected ---\n%s\n"+
			"--- actual ---\n%s\n"+
			"Run `go test ./... -count=1 -update-golden` if this change is intentional.",
			name, want, gotCanonical)
	}
}

// assertContainsAll checks that body contains all expected substrings.
// Used for formats (YAML, D2) that can't be canonicalized without adding
// a dedicated parser dependency. These smoke tests catch missing services,
// endpoints, or fields without being fragile to formatter whitespace changes.
func assertContainsAll(t *testing.T, name string, body []byte, want ...string) {
	t.Helper()

	s := string(body)

	for _, w := range want {
		if !strings.Contains(s, w) {
			t.Errorf("%s: expected output to contain %q", name, w)
		}
	}
}

// --- JSON golden tests (canonical — formatter-proof) ---

func TestGolden_OpenAPI(t *testing.T) {
	t.Parallel()

	cat := goldenCatalog().Build()
	body := serveBody(t, cataloghtmx.OpenAPIHandler(cat))
	compareGolden(t, "openapi.json", body)
}

func TestGolden_AsyncAPI(t *testing.T) {
	t.Parallel()

	cat := goldenCatalog().Build()
	body := serveBody(t, cataloghtmx.AsyncAPIHandler(cat))
	compareGolden(t, "asyncapi.json", body)
}

// --- YAML / D2 smoke tests (structural — formatter-proof) ---
//
// These formats can't be canonicalized without importing a YAML or D2 parser.
// Instead of byte-for-byte golden comparison (which breaks when pre-commit
// formatters reformat the golden file), we verify key structural markers.

func TestGolden_OpenAPIYAML(t *testing.T) {
	t.Parallel()

	cat := goldenCatalog().Build()
	handler := cataloghtmx.OpenAPIHandler(cat, cataloghtmx.WithFormat(cataloghtmx.FormatYAML))
	body := serveBody(t, handler)

	assertContainsAll(
		t, "OpenAPI YAML", body,
		"openapi: 3.0.3",
		"Golden Service",
		"create-thing",
		"thing.created",
		"get-thing",
		"email",
		"user_id",
	)
}

func TestGolden_D2(t *testing.T) {
	t.Parallel()

	cat := goldenCatalog().Build()
	body := serveBody(t, cataloghtmx.D2Handler(cat))

	assertContainsAll(
		t, "D2 diagram", body,
		"Golden Service",
		"golden_svc",
		"create_thing",
		"thing_created",
		"get_thing",
		"Email address",
		"User identifier",
	)
}
