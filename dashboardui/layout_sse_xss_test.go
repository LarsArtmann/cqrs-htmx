package dashboardui

import (
	"html"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	memorystorage "github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

// TestDashboardJSRowBuilderNeverBuildsHTML pins the SSE XSS fix (M01/F01+F02):
// the live row builder must construct every cell through
// createElement/textContent so event metadata arriving over the feed is
// rendered as text, never parsed as HTML. The pre-fix builder concatenated
// innerHTML strings with raw payload values (type, streamId, version,
// eventId), so any event payload containing markup executed in the dashboard
// origin.
func TestDashboardJSRowBuilderNeverBuildsHTML(t *testing.T) {
	if strings.Contains(dashboardJS, "innerHTML") {
		t.Fatalf("dashboardJS must not use innerHTML anywhere: it renders untrusted SSE event metadata")
	}

	for _, want := range []string{
		`document.createElement("td")`,
		`td.textContent = text == null ? "" : String(text)`,
		`encodeURIComponent(data.eventId)`,
		`code.textContent = data.type || "";`,
	} {
		if !strings.Contains(dashboardJS, want) {
			t.Errorf("dashboardJS row builder must contain %q", want)
		}
	}
}

// TestDashboardJSRowBuilderFiveCells pins the column alignment (M01/F03): the
// events table header row has five columns (Time, Type, Stream ID, Stream
// Type, Version — see events.templ), and the live row must append exactly
// those cells in that order, including the previously missing Stream Type cell
// emitted from the EventPayload.streamType field the SSE envelope already
// carries.
func TestDashboardJSRowBuilderFiveCells(t *testing.T) {
	tests := []struct{ name, cellExpression string }{
		{"time", `row.appendChild(cell(now.toLocaleTimeString(), "mono"));`},
		{"stream id", `row.appendChild(cell(String(data.streamId || "").substring(0, 20), "mono"));`},
		{"stream type", `row.appendChild(cell(data.streamType || ""));`},
		{"version", `row.appendChild(cell(data.version || ""));`},
	}
	for _, tt := range tests {
		if !strings.Contains(dashboardJS, tt.cellExpression) {
			t.Errorf("dashboardJS row builder must emit the %s cell via %q", tt.name, tt.cellExpression)
		}
	}
}

// TestDashboardJSParseErrorIsLogged pins M01/F04: a malformed SSE payload must
// surface as a console.warn, not be silently swallowed — the pre-fix
// `catch (err) {}` hid feed corruption from operators.
func TestDashboardJSParseErrorIsLogged(t *testing.T) {
	if !strings.Contains(dashboardJS, `console.warn("dashboard: failed to parse SSE event payload", err)`) {
		t.Fatalf("dashboardJS must log SSE payload parse failures via console.warn")
	}
	if strings.Contains(dashboardJS, "catch (err) {}") {
		t.Fatalf("dashboardJS must not swallow SSE parse errors silently")
	}
}

// TestDashboardJSEventCountResetsOnViewSwap pins M01/F04: the received-events
// counter must reset whenever HTMX swaps a view, so the badge counts events
// received on the current page instead of accumulating since first load.
func TestDashboardJSEventCountResetsOnViewSwap(t *testing.T) {
	for _, want := range []string{
		`document.addEventListener("htmx:afterSwap", function() {`,
		"eventCount = 0;",
	} {
		if !strings.Contains(dashboardJS, want) {
			t.Errorf("dashboardJS must reset the SSE event counter on HTMX view swaps (missing %q)", want)
		}
	}
}

// TestDashboardJSInjectionSmoke is the M01/F05 hostile-payload proof. The Go
// test suite has no JS runtime, so it pins the contract from both sides:
//
//  1. Behavioral (this test): the textContent rendering contract is
//     html.EscapeString in Go — hostile markup comes out as inert text — and
//     the href contract is encodeURIComponent — a javascript:-style payload
//     loses its scheme and becomes an inert relative path segment.
//  2. Structural (TestDashboardJSRowBuilderNeverBuildsHTML): the served script
//     implements that contract with createElement/textContent and contains
//     zero innerHTML, so no payload value can re-enter the HTML parser.
//
// Pre-fix, the "<img onerror>" payload executed because the builder spliced it
// into an innerHTML string, and the raw eventId went into the href unescaped.
func TestDashboardJSInjectionSmoke(t *testing.T) {
	markupPayloads := []string{
		`<img src=x onerror="alert('SSE-XSS')">`,
		`<script>alert(1)</script>`,
		`</td></tr><script>alert(document.domain)</script>`,
	}
	for _, payload := range markupPayloads {
		escaped := html.EscapeString(payload)

		if strings.ContainsAny(escaped, "<>") {
			t.Errorf(
				"markup payload %q still contains tags after textContent-equivalent rendering: %q",
				payload,
				escaped,
			)
		}
		if escaped == payload {
			t.Errorf("markup payload %q was not neutralized by textContent-equivalent rendering", payload)
		}
	}

	// The href sink: encodeURIComponent mangles the scheme separator and
	// quotes, so a "javascript:" payload cannot survive as a URI in the
	// detail link.
	for _, payload := range []string{`javascript:alert(1)`, `../../etc/passwd`, `x" onmouseover="alert(1)`} {
		encoded := url.QueryEscape(payload)

		if strings.ContainsAny(encoded, `:"/\ ?&=#`) {
			t.Errorf(
				"href payload %q must lose scheme/quote/separator characters under encodeURIComponent, got %q",
				payload,
				encoded,
			)
		}
	}
}

// TestServedDashboardJSMatchesXSSContract verifies the wire surface: the JS
// actually served at /-/dashboard.js must be the hardened script (no
// innerHTML, textContent builder present), not just the in-memory constant.
func TestServedDashboardJSMatchesXSSContract(t *testing.T) {
	store := memorystorage.NewMemoryStore()

	d, err := New(Config{
		EventSource: store,
		Journal:     store,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	mux := http.NewServeMux()
	d.Mount(mux, "/dashboard/")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/dashboard/-/dashboard.js", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "text/javascript; charset=utf-8" {
		t.Errorf("Content-Type = %q, want text/javascript; charset=utf-8", got)
	}

	body := rec.Body.String()
	if strings.Contains(body, "innerHTML") {
		t.Errorf("served dashboard.js still uses innerHTML")
	}
	if !strings.Contains(body, `td.textContent = text == null ? "" : String(text)`) {
		t.Errorf("served dashboard.js lacks the textContent row builder")
	}
}
