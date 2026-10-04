package dashboardui

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/a-h/templ"
	"github.com/larsartmann/templ-components/icons"
	memorystorage "github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

// embedShell is a test layout that records the meta it receives and renders
// a recognizable consumer-chrome document around the content.
type embedShell struct {
	got   PageMeta
	calls int
}

func (s *embedShell) render(meta PageMeta, content templ.Component) templ.Component {
	s.got = meta
	s.calls++

	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		if _, err := w.Write([]byte("<html data-app-shell=true><nav data-app-nav>")); err != nil {
			return err
		}

		if _, err := w.Write([]byte("APP")); err != nil {
			return err
		}

		if _, err := w.Write([]byte("</nav>")); err != nil {
			return err
		}

		return content.Render(ctx, w)
	})
}

func newEmbedDashboard(t *testing.T) (*Dashboard, *embedShell) {
	t.Helper()

	store := memorystorage.NewMemoryStore()
	shell := &embedShell{}

	d, err := New(Config{
		EventSource: store,
		Journal:     store,
		Title:       "Test Brand",
		BasePath:    "/obs",
		ReadOnly:    true,
		Layout:      shell.render,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	return d, shell
}

func TestEmbed_LayoutReplacesBuiltInShell(t *testing.T) {
	d, shell := newEmbedDashboard(t)

	mux := http.NewServeMux()
	d.Mount(mux, "/obs/")

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/obs/", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}

	body := rec.Body.String()

	if !strings.Contains(body, `data-app-shell=true`) {
		t.Error("custom shell markup missing — Layout was not used for the full-page render")
	}

	if !strings.Contains(body, "stat-total-events") {
		t.Error("dashboard content missing under the custom shell")
	}

	if strings.Contains(body, `class="sidebar-nav"`) {
		t.Error("built-in sidebar rendered although a custom Layout is configured")
	}

	if strings.Contains(body, "dashboard.css") {
		t.Error("built-in stylesheet link rendered although a custom Layout owns the head")
	}

	if shell.calls != 1 {
		t.Errorf("layout calls = %d, want 1", shell.calls)
	}
}

func TestEmbed_MetaCarriesPageAndNavData(t *testing.T) {
	d, shell := newEmbedDashboard(t)

	mux := http.NewServeMux()
	d.Mount(mux, "/obs/")

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/obs/events", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}

	meta := shell.got

	if meta.Title != "Events" {
		t.Errorf("meta.Title = %q, want Events", meta.Title)
	}

	if meta.FullTitle != "Events · Test Brand" {
		t.Errorf("meta.FullTitle = %q, want %q", meta.FullTitle, "Events · Test Brand")
	}

	if meta.BasePath != "/obs" {
		t.Errorf("meta.BasePath = %q, want /obs", meta.BasePath)
	}

	if !meta.ReadOnly {
		t.Error("meta.ReadOnly should mirror Config.ReadOnly")
	}

	if !meta.Capabilities.EventSource || !meta.Capabilities.HasEventRead() {
		t.Errorf("meta.Capabilities = %+v, want event read active", meta.Capabilities)
	}

	if len(meta.Nav) == 0 {
		t.Fatal("meta.Nav is empty — capability-filtered nav entries expected")
	}

	if nav := meta.Nav[0]; nav.Href != "/" || nav.Label != "Overview" || nav.Active {
		t.Errorf("nav[0] = %+v, want inactive Overview at /", nav)
	}

	eventsNav, found := findNav(meta.Nav, "/events")
	if !found {
		t.Fatal("events nav entry missing from meta.Nav")
	}

	if !eventsNav.Active {
		t.Error("events nav entry should be Active on /events")
	}

	if eventsNav.Icon != icons.QueueList {
		t.Errorf("events nav icon = %q, want the templ-components queue icon", eventsNav.Icon)
	}

	wantCSS := []string{"/obs/-/dashboard.css", "/obs/-/dashboard-tw.css"}
	if len(meta.CSSURLs) != len(wantCSS) {
		t.Fatalf("meta.CSSURLs = %v, want %v", meta.CSSURLs, wantCSS)
	}

	for i, u := range wantCSS {
		if meta.CSSURLs[i] != u {
			t.Errorf("meta.CSSURLs[%d] = %q, want %q", i, meta.CSSURLs[i], u)
		}
	}

	wantScripts := []string{"/obs/-/htmx.js", "/obs/-/dashboard.js"}
	if len(meta.ScriptURLs) != len(wantScripts) {
		t.Fatalf("meta.ScriptURLs = %v, want %v", meta.ScriptURLs, wantScripts)
	}
}

func TestEmbed_HTMXSwapsBypassTheShell(t *testing.T) {
	d, shell := newEmbedDashboard(t)

	mux := http.NewServeMux()
	d.Mount(mux, "/obs/")

	req := httptest.NewRequest(http.MethodGet, "/obs/", nil)
	req.Header.Set("HX-Request", "true")

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	body := rec.Body.String()

	if !strings.Contains(body, `id="main-content"`) {
		t.Error("HTMX fragment should keep the built-in main-content wrapper")
	}

	if strings.Contains(body, `data-app-shell=true`) {
		t.Error("custom shell must not render for HTMX partial requests")
	}

	if shell.calls != 0 {
		t.Errorf("layout calls = %d, want 0 for HTMX partial", shell.calls)
	}
}

func findNav(nav []NavLink, href string) (NavLink, bool) {
	for _, n := range nav {
		if n.Href == href {
			return n, true
		}
	}

	return NavLink{}, false
}
