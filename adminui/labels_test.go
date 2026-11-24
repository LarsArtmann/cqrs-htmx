package adminui

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/a-h/templ"
)

// testPageData is the minimal page shell for component-level label tests:
// the merged default table, English, ltr, and a fixed UTC render location so
// date shapes pin deterministically.
func testPageData() pageData {
	return pageData{
		BasePath: "/admin",
		Labels:   mergeLabels(nil),
		Lang:     "en",
		Dir:      dirLTR,
		Loc:      time.UTC,
	}
}

// renderPageData renders one templ component to a string, failing the test on
// a render error (a half-rendered body would make the assertions lie).
func renderPageData(t *testing.T, c templ.Component) string {
	t.Helper()
	var b strings.Builder
	if err := c.Render(context.Background(), &b); err != nil {
		t.Fatalf("render: %v", err)
	}
	return b.String()
}

// TestLabels_OverrideReplacesTableValue pins the override semantics: an
// override REPLACES the table value for a known key, and the replacement is
// visible on every surface that reads the key (sidebar nav, page header,
// empty state) because they all read the ONE merged table.
func TestLabels_OverrideReplacesTableValue(t *testing.T) {
	t.Parallel()

	svc := mustService(t)
	user := mustUser(t, "admin@example.com")
	h, _ := newTestPanel(t, user, Config{
		Service: svc,
		Labels: map[string]string{
			labelNavUsers:            "Brukere",
			labelEmptyUsersNoneTitle: "Ingen brukere ennå",
		},
	})

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/users", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{"Brukere", "Ingen brukere ennå"} {
		if !strings.Contains(body, want) {
			t.Errorf("override %q not rendered", want)
		}
	}
	if strings.Contains(body, "No users yet") {
		t.Error("default label still rendered alongside the override")
	}
}

// TestLabels_UnknownOverrideRejected pins the honest validation choice:
// unknown override keys are REJECTED at New. An accepted-but-inert override
// would hide a typo forever — the page would render the default string with
// no signal that the override never landed.
func TestLabels_UnknownOverrideRejected(t *testing.T) {
	t.Parallel()

	_, err := New(Config{
		Service: mustService(t),
		Labels:  map[string]string{"nav.usres": "typo"},
	})
	if err == nil {
		t.Fatal("expected New to reject an unknown override key")
	}
	if !strings.Contains(err.Error(), "nav.usres") {
		t.Errorf("error should name the unknown key, got: %v", err)
	}
}

// TestLabels_FallbackRendersKey pins the fallback honesty: a label key with
// no table entry renders the KEY ITSELF, never blank — a blank button or
// header would be a silent lie about the surface.
func TestLabels_FallbackRendersKey(t *testing.T) {
	t.Parallel()

	// Free function.
	if got := label(map[string]string{}, "totally.missing"); got != "totally.missing" {
		t.Errorf("label fallback = %q, want the key itself", got)
	}
	if got := label(nil, "also.missing"); got != "also.missing" {
		t.Errorf("label(nil) fallback = %q, want the key itself", got)
	}

	// Render level: a sparse table (one entry only) renders the raw key for
	// every other covered string.
	p := testPageData()
	p.Labels = map[string]string{labelNavUsers: "Users"}
	body := renderPageData(t, usersTableContent(p, usersListData{BasePath: "/admin"}))
	if !strings.Contains(body, "empty.users.none.title") {
		t.Error("unmapped key must render as the key text in markup")
	}
	if strings.Contains(body, "No users yet") {
		t.Error("sparse table must not fall back to defaults; the key text is the honest fallback")
	}
}

// TestLangDirRender pins F119: Config.Locale drives the <html lang>
// attribute and Config.Dir the <html dir> attribute; the defaults render
// lang="en" dir="ltr" so the attribute pair is always present (WCAG 3.1.1
// baseline) instead of left to the browser's guess.
func TestLangDirRender(t *testing.T) {
	t.Parallel()

	user := mustUser(t, "admin@example.com")
	defaultPanel, _ := newTestPanel(t, user, Config{Service: mustService(t)})
	rec := httptest.NewRecorder()
	defaultPanel.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/", nil))
	body := rec.Body.String()
	if !strings.Contains(body, `lang="en"`) || !strings.Contains(body, `dir="ltr"`) {
		t.Errorf("default panel must render lang=\"en\" dir=\"ltr\"; html tag: %q", htmlTagOf(body))
	}

	rtlPanel, _ := newTestPanel(t, user, Config{Service: mustService(t), Locale: "ar", Dir: "rtl"})
	rec = httptest.NewRecorder()
	rtlPanel.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/", nil))
	body = rec.Body.String()
	if !strings.Contains(body, `lang="ar"`) || !strings.Contains(body, `dir="rtl"`) {
		t.Errorf("Locale=ar/Dir=rtl must render on the html element; got %q", htmlTagOf(body))
	}
}

// htmlTagOf extracts the <html …> open tag for failure messages.
func htmlTagOf(body string) string {
	start := strings.Index(body, "<html")
	if start < 0 {
		return "(no <html> tag)"
	}
	end := strings.Index(body[start:], ">")
	if end < 0 {
		return body[start:]
	}
	return body[start : start+end+1]
}

// TestConfig_LocaleAndDirValidation pins the config guards: a nonsense
// Locale would ship a lying lang attribute to screen readers, and a Dir
// outside ltr/rtl is a typo — both must fail at New.
func TestConfig_LocaleAndDirValidation(t *testing.T) {
	t.Parallel()

	svc := mustService(t)
	for _, tc := range []struct {
		name   string
		locale string
		dir    string
	}{
		{"bad locale", "not a locale!", ""},
		{"bad dir", "", "up"},
	} {
		_, err := New(Config{Service: svc, Locale: tc.locale, Dir: tc.dir})
		if err == nil {
			t.Errorf("%s: expected New to reject", tc.name)
		}
	}
	if _, err := New(Config{Service: svc, Locale: "pt-BR"}); err != nil {
		t.Errorf("region subtag should be accepted, got: %v", err)
	}
}

// TestLocaleDateShapes pins F120: the default (en) layouts keep the exact
// shapes the templates hardcoded before the seam — goldens must not churn
// unless a Locale is configured — while a configured locale picks its shape.
func TestLocaleDateShapes(t *testing.T) {
	t.Parallel()

	stamp := time.Date(2026, time.November, 24, 15, 4, 0, 0, time.UTC)

	for _, tc := range []struct {
		locale     string
		wantDate   string
		wantDTNoTZ string
	}{
		{"", "Nov 24, 2026", "Nov 24, 2026 15:04"},
		{"en", "Nov 24, 2026", "Nov 24, 2026 15:04"},
		{"en-US", "Nov 24, 2026", "Nov 24, 2026 15:04"},
		{"de", "24.11.2026", "24.11.2026 15:04"},
		{"fr", "2026-11-24", "2026-11-24 15:04"},
	} {
		dateLayout, dtLayout := localeDateLayouts(tc.locale)
		if got := stamp.In(time.UTC).Format(dateLayout); got != tc.wantDate {
			t.Errorf("locale %q date = %q, want %q", tc.locale, got, tc.wantDate)
		}
		if got := stamp.In(time.UTC).Format(dtLayout); !strings.HasPrefix(got, tc.wantDTNoTZ) {
			t.Errorf("locale %q dateTime = %q, want prefix %q", tc.locale, got, tc.wantDTNoTZ)
		}
	}

	// Render-level: the page helpers format in the page's location, so the
	// same instant renders in the viewer's configured zone.
	p := testPageData()
	p.Lang = "de"
	if got := p.date(stamp); got != "24.11.2026" {
		t.Errorf("p.date(de) = %q, want 24.11.2026", got)
	}
	p.Lang = "en"
	p.Loc = time.FixedZone("CET", 3600)
	if got := p.dateTime(stamp); got != "Nov 24, 2026 16:04 CET" {
		t.Errorf("p.dateTime(en, CET) = %q, want Nov 24, 2026 16:04 CET", got)
	}
}

// TestLabels_DefaultsMatchLegacyStrings spot-checks that the table values
// for a sample of keys equal the strings the templates hardcoded before the
// seam — the guarantee that default-config renders (and goldens) are
// unchanged.
func TestLabels_DefaultsMatchLegacyStrings(t *testing.T) {
	t.Parallel()

	for key, want := range map[string]string{
		labelNavAudit:            "Audit log",
		labelChromeSignOut:       "Sign out",
		labelChromeReadOnly:      "Read-only",
		labelConfirmTitle:        "Are you sure?",
		labelConfirmOK:           "Confirm",
		labelEmptyUsersNoneTitle: "No users yet",
		labelEmptyAuditNoneTitle: "No audit events",
		labelActionNewTenant:     "New tenant",
		labelHeaderMember:        "Member",
		labelConfirmMemberRemove: "Remove member",
		labelPageDirectory:       "Membership directory",
	} {
		if got := defaultLabels[key]; got != want {
			t.Errorf("label %q = %q, want %q (default renders must stay identical)", key, got, want)
		}
	}
}

// mergeLabels must never mutate the default table: two panels with different
// overrides in one process must not leak into each other.
func TestLabels_MergeDoesNotMutateDefaults(t *testing.T) {
	t.Parallel()

	merged := mergeLabels(map[string]string{labelNavUsers: "People"})
	if defaultLabels[labelNavUsers] != "Users" {
		t.Error("mergeLabels mutated the default table")
	}
	if merged[labelNavUsers] != "People" {
		t.Error("override not applied to the merged copy")
	}
	if len(merged) != len(defaultLabels) {
		t.Errorf("merged table should be one entry per table key: got %d, want %d", len(merged), len(defaultLabels))
	}
}

// labelf interpolates through the merged table: overrides control the whole
// sentence (verbs included), defaults carry the %s/%d verb form, an unknown
// key still renders the KEY ITSELF, and a no-arg call must not run Sprintf.
func TestLabelf_InterpolatesThroughTable(t *testing.T) {
	t.Parallel()

	merged := mergeLabels(map[string]string{
		labelConfirmUsersDeleteBody: "Benutzer %s löschen?",
	})
	if got := labelf(merged, labelConfirmUsersDeleteBody, "a@b.dev"); got != "Benutzer a@b.dev löschen?" {
		t.Errorf("override not interpolated: %q", got)
	}
	if got := labelf(merged, labelConfirmTenSuspendBody, 3); got != "Suspend this tenant? Its 3 member(s) lose access until it is reactivated." {
		t.Errorf("default verb form not interpolated: %q", got)
	}
	if got := labelf(merged, "no.such.key", 1); got != "no.such.key" {
		t.Errorf("unknown key must fall back to the key itself: %q", got)
	}
	if got := labelf(merged, labelConfirmTitle); got != "Are you sure?" {
		t.Errorf("no-arg call must return the table value untouched: %q", got)
	}
}
