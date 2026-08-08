# Snapshot Testing Options for cqrs-htmx

**Date:** 2026-05-19\
**Context:** 245 test specs across 19 test files. Many tests use verbose `w.Body.String()` + `ContainSubstring` and header-by-header assertions that are brittle and hard to maintain.

---

## The Problem

Current test patterns are repetitive and fragile:

```go
// integration_test.go — 4 assertions for one response
Expect(w.Code).To(Equal(http.StatusOK))
Expect(w.Header().Get("HX-Trigger")).To(Equal("userCreated"))
Expect(w.Header().Get("HX-Push-Url")).To(ContainSubstring("/users/"))
Expect(w.Body.String()).To(ContainSubstring("Alice"))

// logging_test.go — 5 assertions for one log line
Expect(logged).To(ContainSubstring("POST /users"))
Expect(logged).To(ContainSubstring("Created"))
Expect(logged).To(ContainSubstring("correlation=01HK1549P84T9XF8R94E960633"))
Expect(logged).To(ContainSubstring("user=01HK154ANGZHV2ZW0X3SKSNEN2"))

// csrf_test.go — manual HTML string maintenance
Expect(meta).To(Equal(`<meta name="csrf-token" content="test-token">`))
```

**Issues:**

- `ContainSubstring` is weak — misses extra/missing output, wrong ordering
- Adding a new header requires updating every test that checks headers
- JSON log assertions break when field order changes
- HTML snapshots must be manually updated when output changes

---

## Option 1: `go-snaps` (RECOMMENDED)

**Library:** `github.com/gkampitakis/go-snaps`\
**Stars:** ~260 | **Status:** Actively maintained (v0.5.x, 2026)\
**Go version:** 1.21+

Modern Jest-like snapshot testing with dynamic value masking, JSON/YAML formatters, and automatic snapshot cleanup.

### Installation

```bash
go get github.com/gkampitakis/go-snaps/snaps
```

### Example: HTTP Response Snapshot

```go
package cqrshtmx_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	cqrshtmx "github.com/larsartmann/cqrs-htmx"
	"github.com/gkampitakis/go-snaps/snaps"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// SnapResponse snapshots the full HTTP response (status + headers + body)
func SnapResponse(w *httptest.ResponseRecorder) {
	var b strings.Builder
	fmt.Fprintf(&b, "Status: %d %s\n", w.Code, http.StatusText(w.Code))
	fmt.Fprintln(&b, "Headers:")
	for name, values := range w.Result().Header {
		for _, v := range values {
			fmt.Fprintf(&b, "  %s: %s\n", name, v)
		}
	}
	fmt.Fprintln(&b, "Body:")
	b.WriteString(w.Body.String())

	snaps.MatchSnapshot(GinkgoT(), b.String())
}

// SnapJSON snapshots JSON responses with dynamic value masking
func SnapJSON(w *httptest.ResponseRecorder, matchers ...snaps.Match) {
	snaps.MatchSnapshot(GinkgoT(), w.Body.String(), matchers...)
}
```

### Usage in Tests

```go
// BEFORE (integration_test.go — 4 lines)
Expect(w.Code).To(Equal(http.StatusOK))
Expect(w.Header().Get("HX-Trigger")).To(Equal("userCreated"))
Expect(w.Header().Get("HX-Push-Url")).To(ContainSubstring("/users/"))
Expect(w.Body.String()).To(ContainSubstring("Alice"))

// AFTER (1 line)
SnapResponse(w)

// ---

// BEFORE (logging_test.go — 5 lines)
Expect(logged).To(ContainSubstring("POST /users"))
Expect(logged).To(ContainSubstring("Created"))
Expect(logged).To(ContainSubstring("correlation=01HK1549P84T9XF8R94E960633"))
Expect(logged).To(ContainSubstring("user=01HK154ANGZHV2ZW0X3SKSNEN2"))

// AFTER (with dynamic masking for timestamps/IDs)
import "github.com/gkampitakis/go-snaps/snaps/match"

SnapJSON(w, match.Any("duration"), match.Any("correlation_id"), match.Any("user_id"))

// ---

// BEFORE (csrf_test.go — manual HTML strings)
Expect(meta).To(Equal(`<meta name="csrf-token" content="test-token">`))

// AFTER
snaps.MatchSnapshot(GinkgoT(), meta)
```

### Dynamic Value Masking

Critical for this project since ULIDs and timestamps change per test run:

```go
import "github.com/gkampitakis/go-snaps/snaps/match"

// Mask any ULID-like value
snaps.MatchSnapshot(GinkgoT(), logged,
	match.Custom("correlation_id", match.ULIDRegex),  // built-in
	match.Custom("user_id", match.ULIDRegex),
	match.Custom("aggregate_id", `[A-Z0-9]{26}`),
)

// Mask by type
snaps.MatchSnapshot(GinkgoT(), response,
	match.Type[int]("status_code"),      // any int
	match.Type[string]("message"),       // any string
)
```

### Snapshot File Format

Stored in `__snapshots__/` adjacent to test files:

```
cqrs-htmx/
├── __snapshots__/
│   ├── integration_test_test.snap
│   ├── csrf_test_test.snap
│   └── logging_test_test.snap
```

Example `.snap` file:

```
[TestCQRSHTMX/Full_Integration/End-to-end_CQRS_+_HTMX_+_Casbin_flow/returns_command_result_with_HTMX_headers - 1]
Status: 200 OK
Headers:
  Content-Type: text/html; charset=utf-8
  Hx-Push-Url: /users/01HK1549P84T9XF8R94E960633
  Hx-Trigger: userCreated
Body:
<p>Alice</p>

---
```

### Updating Snapshots

```bash
# Update all snapshots
UPDATE_SNAPS=true go test ./...

# Or with our env pattern
GOWORK=off UPDATE_SNAPS=true go test ./... -count=1
```

### CI Integration

```yaml
# .github/workflows/ci.yml
- name: Test
  run: go test ./... -count=1
  env:
    CI: true # go-snaps auto-detects CI and fails on missing snapshots
```

### Pros

- **Best Ginkgo/Gomega integration** — uses `GinkgoT()` natively
- **Dynamic value masking** — ULIDs, timestamps, random values don't break snapshots
- **Built-in JSON/YAML formatters** — pretty-prints for readable diffs
- **Auto cleanup** — removes obsolete snapshots automatically
- **CI-friendly** — fails in CI if snapshots are missing or outdated
- **Inline snapshots** — experimental support for snapshots in source code

### Cons

- Adds 1 new dependency (`go-snaps` + `match` subpackage)
- Newer library (but actively maintained since 2022)
- Snapshot files add noise to git diffs (but are auto-managed)

---

## Option 2: `cupaloy`

**Library:** `github.com/bradleyjkemp/cupaloy/v2`\
**Stars:** ~330 | **Status:** Stable/maintenance mode (v2.8.0, 2022)

The original Go snapshot library. Extremely simple API.

### Installation

```bash
go get github.com/bradleyjkemp/cupaloy/v2
```

### Example

```go
import "github.com/bradleyjkemp/cupaloy/v2"

It("returns the correct HTML meta tag", func() {
	meta := cqrshtmx.CSRFTokenHTMLMeta(r)
	cupaloy.SnapshotT(GinkgoT(), meta)
})
```

### Updating Snapshots

```bash
UPDATE_SNAPS=true go test ./...
```

### Pros

- **Zero configuration** — one function call
- **Battle-tested** — used in production since 2017
- **Very small footprint** — single package, minimal deps
- **Simple mental model** — `SnapshotT(t, value)` is the entire API

### Cons

- **No dynamic value masking** — ULIDs and timestamps will break snapshots
- **No JSON formatting** — snapshots are raw string dumps
- **Maintenance mode** — last meaningful update 2022
- **Less flexible** — no custom matchers, no inline snapshots
- **Not ideal for HTTP responses** — would need custom serialization

### Verdict

Too limited for this project. Without dynamic masking, every test with a ULID or timestamp would need manual snapshot updates on every run.

---

## Option 3: Golden Files (Standard Library)

No dependencies. Store expected output in `testdata/` files.

### Implementation

```go
package cqrshtmx_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// assertGolden compares got against a golden file.
// Set UPDATE_GOLDEN=1 to update golden files.
func assertGolden(t testing.TB, name string, got []byte) {
	t.Helper()

	golden := filepath.Join("testdata", "golden", name+".golden")

	if os.Getenv("UPDATE_GOLDEN") == "1" {
		dir := filepath.Dir(golden)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(golden, got, 0644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		t.Logf("updated golden: %s", golden)
		return
	}

	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("read golden %s: %v", golden, err)
	}

	if diff := cmp.Diff(string(want), string(got)); diff != "" {
		t.Fatalf("golden mismatch (-want +got):\n%s", diff)
	}
}
```

### Usage

```go
It("returns HTMX notification response", func() {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/users", nil)
	// ... execute handler ...

	assertGolden(GinkgoT(), "create_user_success", w.Body.Bytes())
})
```

### Directory Structure

```
cqrs-htmx/
├── testdata/
│   └── golden/
│       ├── create_user_success.golden
│       ├── csrf_meta_tag.golden
│       ├── htmx_response_headers.golden
│       └── json_log_output.golden
```

### Dynamic Value Handling

Golden files don't support dynamic masking natively. You'd need to preprocess:

```go
// Strip dynamic values before comparison
func normalizeOutput(s string) string {
	// Replace ULIDs with placeholder
	s = ulidRegex.ReplaceAllString(s, "<ULID>")
	// Replace timestamps
	s = timestampRegex.ReplaceAllString(s, "<TIMESTAMP>")
	return s
}

assertGolden(GinkgoT(), "log_output", []byte(normalizeOutput(logged)))
```

### Pros

- **Zero dependencies** — uses only stdlib + `go-cmp`
- **Full control** — own the file format, comparison logic, update workflow
- **Git-friendly** — golden files are plain text, reviewable in PRs
- **No magic** — explicit, easy to understand

### Cons

- **Manual file management** — create directories, name files, handle updates
- **No built-in dynamic masking** — must implement preprocessing yourself
- **More boilerplate** — helper function, directory setup, env var handling
- **No auto-cleanup** — stale golden files linger unless manually deleted
- **Ginkgo integration is clunky** — `GinkgoT()` doesn't expose all `testing.T` methods

### Verdict

Good for teams that want zero dependencies and full control, but requires more upfront work and ongoing maintenance than `go-snaps`.

---

## Option 4: Custom `MatchJSON` Gomega Matcher (Lightweight)

Don't add a snapshot library at all. Use Gomega's built-in `MatchJSON` with a helper that loads expected JSON from files.

### Implementation

```go
package cqrshtmx_test

import (
	"encoding/json"
	"os"
	"path/filepath"

	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

// MatchGoldenJSON loads a golden JSON file and matches against it.
// Set UPDATE_GOLDEN=1 to update.
func MatchGoldenJSON(name string) types.GomegaMatcher {
	golden := filepath.Join("testdata", "golden", name+".json")

	if os.Getenv("UPDATE_GOLDEN") == "1" {
		return &goldenJSONUpdater{path: golden}
	}

	data, err := os.ReadFile(golden)
	if err != nil {
		return &goldenJSONFailure{err: err}
	}
	return MatchJSON(data)
}

type goldenJSONUpdater struct{ path string }

func (u *goldenJSONUpdater) Match(actual interface{}) (bool, error) {
	data, err := json.MarshalIndent(actual, "", "  ")
	if err != nil {
		return false, err
	}
	os.MkdirAll(filepath.Dir(u.path), 0755)
	os.WriteFile(u.path, data, 0644)
	return true, nil
}
func (u *goldenJSONUpdater) FailureMessage(actual interface{}) string {
	return "updated golden file: " + u.path
}
func (u *goldenJSONUpdater) NegatedFailureMessage(actual interface{}) string {
	return "should not update golden file"
}

type goldenJSONFailure struct{ err error }

func (f *goldenJSONFailure) Match(actual interface{}) (bool, error) {
	return false, f.err
}
func (f *goldenJSONFailure) FailureMessage(actual interface{}) string {
	return f.err.Error()
}
func (f *goldenJSONFailure) NegatedFailureMessage(actual interface{}) string {
	return f.err.Error()
}
```

### Usage

```go
It("returns structured log output", func() {
	var logged map[string]interface{}
	json.Unmarshal([]byte(raw), &logged)

	// Strip dynamic fields before matching
	delete(logged, "duration")
	delete(logged, "timestamp")

	Expect(logged).To(MatchGoldenJSON("json_log_output"))
})
```

### Pros

- **Zero new dependencies** — uses only Gomega
- **Native Gomega integration** — fits existing test style perfectly
- **JSON-aware** — `MatchJSON` ignores field ordering and whitespace

### Cons

- **Only works for JSON** — HTML, plain text, headers need different approach
- **Manual preprocessing** — must strip dynamic fields before matching
- **No HTTP response snapshotting** — doesn't solve the header-checking problem
- **More code to maintain** — custom matcher implementation

### Verdict

Too narrow. Only solves JSON assertions, leaving HTML, headers, and status codes as manual assertions.

---

## Option 5: Hybrid — `go-snaps` for Responses, Explicit for Behavior

**The pragmatic approach.** Use snapshot testing for "what did the system output?" and keep explicit assertions for "did the system do the right thing?"

### Snapshot These (output-heavy)

| Test Type              | Example                                   | Why Snapshot                                |
| ---------------------- | ----------------------------------------- | ------------------------------------------- |
| HTML/template output   | `CSRFTokenHTMLMeta`, `CSRFTokenHXHeaders` | HTML structure changes are visual diffs     |
| HTTP response bodies   | `w.Body.String()` with complex HTML/JSON  | Full output is the contract                 |
| Log output             | JSON log lines with many fields           | Field additions/changes are snapshot-worthy |
| Error messages         | `w.Body.String()` on error cases          | Error text is part of the UX                |
| Multi-header responses | HTMX responses with 3+ headers            | Header combinations are the contract        |

### Keep Explicit (behavioral)

| Test Type                   | Example                        | Why Explicit                             |
| --------------------------- | ------------------------------ | ---------------------------------------- |
| Status codes                | `w.Code == 200`                | Status codes are the API contract        |
| Mock call counts            | `dispatchCount == 1`           | Behavior, not output                     |
| Error types                 | `errors.Is(err, ErrForbidden)` | Type safety matters                      |
| Nil checks                  | `cfg.queryDecoder != nil`      | Preconditions                            |
| Simple single-value headers | `w.Header().Get("Location")`   | One assertion is cleaner than a snapshot |

### Migration Strategy

```go
// Step 1: Add go-snaps + helper
// Step 2: Pick ONE test file to migrate (recommend integration_test.go)
// Step 3: Convert the most verbose tests first
// Step 4: Run tests, review snapshots, commit
// Step 5: Repeat for csrf_test.go, logging_test.go, coverage_test.go
// Step 6: Leave unit tests (app_test.go, context_test.go) as explicit assertions
```

---

## Comparison Matrix

| Feature                  | go-snaps        | cupaloy     | Golden Files | Custom Matcher |
| ------------------------ | --------------- | ----------- | ------------ | -------------- |
| **Dynamic masking**      | ✅ Built-in     | ❌ None     | ⚠️ Manual     | ⚠️ Manual       |
| **Ginkgo native**        | ✅ Yes          | ✅ Yes      | ⚠️ Clunky     | ✅ Yes         |
| **JSON formatting**      | ✅ Yes          | ❌ No       | ⚠️ Manual     | ✅ Yes         |
| **Auto cleanup**         | ✅ Yes          | ✅ Yes      | ❌ No        | ❌ No          |
| **HTTP response helper** | ✅ Easy         | ⚠️ Custom    | ⚠️ Custom     | ❌ No          |
| **Zero deps**            | ❌ 1 dep        | ❌ 1 dep    | ✅ Yes       | ✅ Yes         |
| **Active maintenance**   | ✅ 2026         | ⚠️ 2022      | N/A (you)    | N/A (you)      |
| **CI integration**       | ✅ Built-in     | ✅ Built-in | ⚠️ Custom     | ⚠️ Custom       |
| **Snapshot inline**      | ✅ Experimental | ❌ No       | ❌ No        | ❌ No          |
| **Learning curve**       | Low             | Very low    | Medium       | Medium         |

---

## Recommendation

**Use `go-snaps` (Option 1) with the hybrid approach (Option 5).**

### Why

1. **Dynamic masking is non-negotiable** — Your tests generate ULIDs (`id.NewAggregateID()`, `id.NewCorrelationID()`) and timestamps on every run. Without masking, snapshots would fail constantly.
2. **Ginkgo is your test framework** — `go-snaps` is the only library designed with Ginkgo in mind (`GinkgoT()` integration).
3. **HTTP responses are your core domain** — This library builds HTTP handlers. Snapshotting full responses (status + headers + body) is the most valuable test improvement.
4. **Future-proof** — Actively maintained, modern API, inline snapshots coming.

### What to Snapshot First (Priority Order)

| Priority | File                  | Tests to Snapshot                                  | Impact                                           |
| -------- | --------------------- | -------------------------------------------------- | ------------------------------------------------ |
| P1       | `integration_test.go` | Full HTTP responses (status + all headers + body)  | Highest — 50+ assertions become ~15 snapshots    |
| P2       | `csrf_test.go`        | Template helper output (`CSRFTokenHTMLMeta`, etc.) | Medium — HTML string maintenance eliminated      |
| P2       | `logging_test.go`     | JSON log output lines                              | Medium — JSON field changes become visible diffs |
| P3       | `coverage_test.go`    | Error response bodies, notification triggers       | Medium — complex multi-field assertions          |
| P4       | `security_test.go`    | Header output                                      | Low — already simple, 1-2 assertions each        |
| —        | `app_test.go`         | Keep explicit                                      | Low — mostly behavior, not output                |
| —        | `context_test.go`     | Keep explicit                                      | Low — type-level tests, no HTTP output           |

### Files to Create

```
cqrs-htmx/
├── snapshot_test.go          # SnapResponse, SnapJSON helpers
├── __snapshots__/            # Auto-generated, git-tracked
│   ├── integration_test_test.snap
│   ├── csrf_test_test.snap
│   ├── logging_test_test.snap
│   └── coverage_test_test.snap
```

### Estimated Effort

| Task                                      | Est. Time   |
| ----------------------------------------- | ----------- |
| Add `go-snaps` dependency                 | 2 min       |
| Write `SnapResponse` + `SnapJSON` helpers | 10 min      |
| Migrate `integration_test.go`             | 20 min      |
| Migrate `csrf_test.go` template helpers   | 15 min      |
| Migrate `logging_test.go`                 | 10 min      |
| Review + update snapshots                 | 10 min      |
| **Total**                                 | **~70 min** |

### Risk Assessment

| Risk                           | Mitigation                                      |
| ------------------------------ | ----------------------------------------------- |
| Snapshot files grow large      | `go-snaps` auto-cleans obsolete snapshots       |
| ULIDs still break snapshots    | Use `match.Custom` with ULID regex              |
| Team unfamiliar with snapshots | Document update workflow in AGENTS.md           |
| CI fails on new snapshots      | Set `UPDATE_SNAPS=false` in CI, fail on missing |
| Accidental snapshot commits    | Review `.snap` files in PRs like any code       |

---

## Open Questions

1. **Should we snapshot the full HTTP response or just the body?** Full response captures headers too, but makes snapshots larger.
2. **Should we commit `.snap` files or `.gitignore` them?** Commit them — they are the test oracle.
3. **What about `go.work` / `GOWORK=off` compatibility?** `go-snaps` is a normal Go module, no special handling needed.
4. **Do we need a pre-commit hook check for uncommitted snapshots?** Yes — add to BuildFlow or a simple git hook.

---

_Document generated: 2026-05-19_\
_Next step: Decide on approach, then implement proof-of-concept on one test file_
