package dashboardui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	memorystorage "github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/larsartmann/templ-components/icons"
)

func TestValidate_RejectsExtraNavWithEmptyLabel(t *testing.T) {
	store := memorystorage.NewMemoryStore()

	_, err := Config{
		EventSource: store,
		ExtraNav:    []NavEntry{{Href: "/event-catalog"}},
	}.Validate()
	if err == nil {
		t.Fatal("Validate() with an empty ExtraNav label should fail")
	}

	if errorfamily.Classify(err) != errorfamily.Rejection {
		t.Errorf("Validate error family = %v; want Rejection; got %v", errorfamily.Classify(err), err)
	}
}

func TestValidate_RejectsExtraNavWithEmptyHref(t *testing.T) {
	store := memorystorage.NewMemoryStore()

	_, err := Config{
		EventSource: store,
		ExtraNav:    []NavEntry{{Label: "Event Catalog"}},
	}.Validate()
	if err == nil {
		t.Fatal("Validate() with an empty ExtraNav href should fail")
	}
}

func TestNew_AppendsExtraNavAfterCapabilityItems(t *testing.T) {
	store := memorystorage.NewMemoryStore()

	d, err := New(Config{
		EventSource: store,
		ExtraNav: []NavEntry{
			{Label: "Event Catalog", Href: "/event-catalog"},
			{Label: "Docs", Href: "https://docs.example.com", Icon: icons.Book},
		},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if len(d.nav) < 3 {
		t.Fatalf("nav = %+v, want capability items plus two extra entries", d.nav)
	}

	last := d.nav[len(d.nav)-2:]
	if last[0].Label != "Event Catalog" || last[0].Href != "/event-catalog" {
		t.Errorf("extra nav[0] = %+v, want Event Catalog at /event-catalog", last[0])
	}

	if last[0].Icon != icons.Link {
		t.Errorf("extra nav[0].Icon = %q, want the default link icon", last[0].Icon)
	}

	if last[1].Icon != icons.Book {
		t.Errorf("extra nav[1].Icon = %q, want the explicit %q icon", last[1].Icon, icons.Book)
	}

	for _, n := range d.nav[:len(d.nav)-2] {
		if strings.HasPrefix(n.Href, "http") || strings.Contains(n.Href, "event-catalog") {
			t.Errorf("capability nav entry unexpectedly changed: %+v", n)
		}
	}
}

func TestExtraNav_RendersInBuiltInSidebarWithVerbatimHref(t *testing.T) {
	store := memorystorage.NewMemoryStore()

	d, err := New(Config{
		EventSource: store,
		BasePath:    "/dashboard",
		ExtraNav:    []NavEntry{{Label: "Event Catalog", Href: "/event-catalog"}},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	mux := http.NewServeMux()
	d.Mount(mux, "/dashboard/")

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/dashboard/", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}

	body := rec.Body.String()

	if !strings.Contains(body, `href="/event-catalog"`) {
		t.Error("extra nav entry href missing or not verbatim (BasePath must NOT be prefixed)")
	}

	if strings.Contains(body, `href="/dashboard/event-catalog"`) {
		t.Error("extra nav href was BasePath-prefixed; it must render verbatim")
	}

	if !strings.Contains(body, "Event Catalog") {
		t.Error("extra nav label missing from the sidebar")
	}
}

func TestExtraNav_FlowsToPageMeta(t *testing.T) {
	store := memorystorage.NewMemoryStore()
	shell := &embedShell{}

	d, err := New(Config{
		EventSource: store,
		Journal:     store,
		BasePath:    "/obs",
		Layout:      shell.render,
		ExtraNav:    []NavEntry{{Label: "Event Catalog", Href: "/event-catalog", Icon: icons.ArchiveBox}},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	mux := http.NewServeMux()
	d.Mount(mux, "/obs/")

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/obs/", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}

	nav, found := findNav(shell.got.Nav, "/event-catalog")
	if !found {
		t.Fatal("extra nav entry missing from PageMeta.Nav")
	}

	if nav.Label != "Event Catalog" || nav.Icon != icons.ArchiveBox || nav.Active {
		t.Errorf("PageMeta extra nav = %+v, want inactive Event Catalog with the archive icon", nav)
	}

	if !nav.Verbatim {
		t.Error("PageMeta extra nav.Verbatim = false; a custom shell would prefix BasePath onto a sibling-surface href")
	}

	for _, builtIn := range shell.got.Nav {
		if builtIn.Href == "/event-catalog" {
			continue
		}

		if builtIn.Verbatim {
			t.Errorf("built-in nav entry %+v marked Verbatim; only ExtraNav entries are rendered as-is", builtIn)
		}
	}
}

// TestExtraNav_InternalHrefHighlightsWhenActive proves an ExtraNav entry that
// points at a dashboard page (basePath + panel path) is marked Active exactly
// like the built-in entry, while a sibling-surface entry never is.
func TestExtraNav_InternalHrefHighlightsWhenActive(t *testing.T) {
	store := memorystorage.NewMemoryStore()
	shell := &embedShell{}

	d, err := New(Config{
		EventSource: store,
		Journal:     store,
		BasePath:    "/obs",
		Layout:      shell.render,
		ExtraNav: []NavEntry{
			{Label: "Events (custom)", Href: "/obs/events"},
			{Label: "Event Catalog", Href: "/event-catalog"},
		},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	mux := http.NewServeMux()
	d.Mount(mux, "/obs/")

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/obs/events", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}

	internal, found := findNav(shell.got.Nav, "/obs/events")
	if !found {
		t.Fatal("internal ExtraNav entry missing from PageMeta.Nav")
	}

	if !internal.Active {
		t.Errorf("internal ExtraNav entry = %+v, want Active on the page it links to", internal)
	}

	external, found := findNav(shell.got.Nav, "/event-catalog")
	if !found {
		t.Fatal("sibling-surface ExtraNav entry missing from PageMeta.Nav")
	}

	if external.Active {
		t.Errorf("sibling-surface ExtraNav entry = %+v, want never Active", external)
	}
}

// TestExtraNav_EscapesLabelAndHref pins the injection surface: a consumer nav
// entry is developer-supplied config, but it is rendered into a page served to
// every dashboard user, so a quote in the label or href must be escaped by the
// templ attribute/render path rather than breaking out of the markup.
func TestExtraNav_EscapesLabelAndHref(t *testing.T) {
	store := memorystorage.NewMemoryStore()

	d, err := New(Config{
		EventSource: store,
		Journal:     store,
		BasePath:    "/obs",
		ExtraNav: []NavEntry{{
			Label: `<img src=x onerror="alert(1)">`,
			Href:  `/x" onmouseover="alert(1)`,
		}},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	mux := http.NewServeMux()
	d.Mount(mux, "/obs/")

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/obs/", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}

	body := rec.Body.String()

	if strings.Contains(body, `onmouseover="alert(1)"`) {
		t.Error("extra nav href broke out of the attribute")
	}

	if strings.Contains(body, `<img src=x onerror=`) {
		t.Error("extra nav label rendered as raw markup")
	}

	if !strings.Contains(body, "&lt;img") {
		t.Error("extra nav label is missing from the sidebar")
	}

	if !strings.Contains(body, `href="/x&#34; onmouseover=&#34;alert(1)"`) {
		t.Error("extra nav href is missing from the sidebar, or was not quote-escaped")
	}
}
