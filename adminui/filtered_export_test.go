package adminui

import (
	"context"
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
)

// exportHarness seeds a directory whose emails and display names sort
// differently from the registration order, so a filtered/sorted export is
// observably different from the full one.
func exportHarness(t *testing.T) (http.Handler, *usermgmt.Service) {
	t.Helper()
	user := mustUser(t, "admin@example.com")
	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{AuditLog: usermgmt.NewAuditLog()})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	for _, s := range []struct{ id, email, display string }{
		{"exp-1", "carol@x.dev", "Mallory"},
		{"exp-2", "alice@x.dev", "Zed"},
		{"exp-3", "bea@x.dev", "Amy"},
	} {
		if _, err := svc.Register(context.Background(), usermgmt.RegisterRequest{
			ID:          identitymodel.SyntheticUserID(s.id),
			Email:       s.email,
			DisplayName: s.display,
		}); err != nil {
			t.Fatalf("register %s: %v", s.email, err)
		}
	}
	return panelWithPageSize(t, user, svc, 50), svc
}

// TestPanel_UsersFilteredExport_UnfilteredParity pins the wire-shape
// contract: with every filter off, the filtered export's bytes are the full
// export's bytes — same header, same columns, same JSON shape — so the two
// buttons can never quietly diverge.
func TestPanel_UsersFilteredExport_UnfilteredParity(t *testing.T) {
	t.Parallel()
	h, _ := exportHarness(t)

	for _, path := range []string{"/admin/users/export.csv", "/admin/users/export.filtered.csv"} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s: status %d", path, rec.Code)
		}
		if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/csv") {
			t.Errorf("%s: content type %q", path, ct)
		}
	}

	full := httptest.NewRecorder()
	h.ServeHTTP(full, httptest.NewRequest(http.MethodGet, "/admin/users/export.csv", nil))
	filtered := httptest.NewRecorder()
	h.ServeHTTP(filtered, httptest.NewRequest(http.MethodGet, "/admin/users/export.filtered.csv", nil))
	if full.Body.String() != filtered.Body.String() {
		t.Errorf("unfiltered filtered-export drifted from the full export:\nfull: %q\nfiltered: %q",
			full.Body.String(), filtered.Body.String())
	}

	fullJSON := httptest.NewRecorder()
	h.ServeHTTP(fullJSON, httptest.NewRequest(http.MethodGet, "/admin/users/export.json", nil))
	filteredJSON := httptest.NewRecorder()
	h.ServeHTTP(filteredJSON, httptest.NewRequest(http.MethodGet, "/admin/users/export.filtered.json", nil))
	if fullJSON.Body.String() != filteredJSON.Body.String() {
		t.Errorf("unfiltered JSON export drifted from the full export")
	}
	if cd := filteredJSON.Header().Get("Content-Disposition"); !strings.Contains(cd, "users-export-filtered.json") {
		t.Errorf("filtered filename must say so: %q", cd)
	}
}

// TestPanel_UsersFilteredExport_HonorsState pins the round trip: q narrows,
// the status chip narrows, sort orders the rows, and the JSON leg carries
// the same set.
func TestPanel_UsersFilteredExport_HonorsState(t *testing.T) {
	t.Parallel()
	h, _ := exportHarness(t)

	get := func(path string) *httptest.ResponseRecorder {
		t.Helper()
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s: status %d", path, rec.Code)
		}
		return rec
	}

	// q narrows the CSV to the one match.
	csv := get("/admin/users/export.filtered.csv?q=alice").Body.String()
	if !strings.Contains(csv, "alice@x.dev") {
		t.Error("q=alice export must carry alice")
	}
	if strings.Contains(csv, "bea@x.dev") || strings.Contains(csv, "carol@x.dev") {
		t.Error("q=alice export must exclude the non-matches")
	}
	if !strings.HasPrefix(csv, "id,email,display_name,email_verified,totp_enabled\n") {
		t.Errorf("CSV must open with the Service export's header, got %.60q", csv)
	}

	// The status chip narrows too: mfa keeps nobody (no seeded user has TOTP).
	if got := get(
		"/admin/users/export.filtered.csv?filter=mfa",
	).Body.String(); got != "id,email,display_name,email_verified,totp_enabled\n" {
		t.Errorf("filter=mfa export must be header-only for a TOTP-less directory, got %q", got)
	}

	// sort orders the CSV rows: alice, bea, carol.
	sorted := get("/admin/users/export.filtered.csv?sort=email").Body.String()
	iAlice := strings.Index(sorted, "alice@x.dev")
	iBea := strings.Index(sorted, "bea@x.dev")
	iCarol := strings.Index(sorted, "carol@x.dev")
	if iAlice < 0 || iBea < 0 || iCarol < 0 || (iAlice >= iBea || iBea >= iCarol) {
		t.Errorf("sort=email must order the exported rows, got %q", sorted)
	}

	// The JSON leg parses and carries the same narrowed set.
	rec := get("/admin/users/export.filtered.json?q=bea")
	var users []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &users); err != nil {
		t.Fatalf("filtered JSON parse: %v; body %.200s", err, rec.Body.String())
	}
	if len(users) != 1 || users[0]["email"] != "bea@x.dev" {
		t.Errorf("q=bea JSON export must carry exactly bea, got %v", users)
	}
}

// TestPanel_UsersFilteredExport_EmptyResult pins the honest empty case: a
// filter that matches nobody exports a header-only CSV and an empty JSON
// array — status 200, a real file, never an error page.
func TestPanel_UsersFilteredExport_EmptyResult(t *testing.T) {
	t.Parallel()
	h, _ := exportHarness(t)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/users/export.filtered.csv?q=nomatch", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("empty CSV export: status %d", rec.Code)
	}
	if got := rec.Body.String(); got != "id,email,display_name,email_verified,totp_enabled\n" {
		t.Errorf("empty CSV export must be the header alone, got %q", got)
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/users/export.filtered.json?q=nomatch", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("empty JSON export: status %d", rec.Code)
	}
	if got := strings.TrimSpace(rec.Body.String()); got != "[]" {
		t.Errorf("empty JSON export must be an empty array, got %q", got)
	}
}

// TestPanel_AuditExport carries the audit window out in both formats, over
// the page's filter set, with the honest window caption leading the file.
func TestPanel_AuditExport(t *testing.T) {
	t.Parallel()
	h, svc := exportHarness(t)

	// The three registrations are audit entries about user actors.
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/audit/export.csv", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("audit CSV export: status %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/csv") {
		t.Errorf("audit CSV content type = %q", ct)
	}
	body := rec.Body.String()
	for _, want := range []string{
		"# Audit export",
		"# Filters: none", // an unfiltered export says so
		"# Audit window holds",
		"entries exported.",
		"action,actor_email,aggregate_id,occurred_at,actor_kind,detail",
		"register,alice@x.dev",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("audit CSV export missing %q", want)
		}
	}
	if strings.Contains(body, "0 entries exported.") {
		t.Error("an unfiltered window with three registrations must not export zero entries")
	}

	// The q filter narrows the file exactly like the page.
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/audit/export.csv?q=alice", nil))
	body = rec.Body.String()
	if !strings.Contains(body, "alice@x.dev") || strings.Contains(body, "bea@x.dev") {
		t.Errorf("q=alice audit export must narrow to alice's entries, got %s", body)
	}
	if !strings.Contains(body, "# Filters: q=alice") {
		t.Error("a narrowed audit export must caption its filter")
	}

	// The actor chip narrows: system-kind entries do not exist in this window.
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/audit/export.csv?actor=system", nil))
	if !strings.Contains(rec.Body.String(), "# 0 entries exported.") {
		t.Errorf("actor=system must narrow to nothing here, got %s", rec.Body.String())
	}

	// The JSON envelope carries the window facts, the filter state, and the
	// entries in one self-describing object.
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/audit/export.json", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("audit JSON export: status %d", rec.Code)
	}
	if cd := rec.Header().Get("Content-Disposition"); !strings.Contains(cd, "audit-export.json") {
		t.Errorf("audit JSON content disposition = %q", cd)
	}
	var doc struct {
		Window struct {
			Holds   int `json:"holds"`
			Limit   int `json:"limit"`
			Dropped int `json:"dropped"`
		} `json:"window"`
		Filters map[string]string `json:"filters"`
		Count   int               `json:"count"`
		Entries []map[string]any  `json:"entries"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &doc); err != nil {
		t.Fatalf("audit JSON parse: %v; body %.200s", err, rec.Body.String())
	}
	// Registration lands two window entries per user (register + the default
	// role grant), so the exact total is the Service's business — the export's
	// business is that count, entries, and window facts agree.
	if doc.Count != len(doc.Entries) || doc.Count < 3 {
		t.Errorf(
			"audit JSON count and entries must agree and be non-trivial, got count=%d entries=%d",
			doc.Count,
			len(doc.Entries),
		)
	}
	if doc.Window.Holds < doc.Count || doc.Window.Limit <= 0 {
		t.Errorf("audit JSON window facts must be present, got %+v", doc.Window)
	}
	if len(doc.Filters) != 0 {
		t.Errorf("unfiltered audit JSON must carry no filters, got %v", doc.Filters)
	}

	// A narrowed JSON export echoes the filter that produced it: every entry
	// belongs to the q match, and the filters object names the q.
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/audit/export.json?q=alice", nil))
	var doc2 struct {
		Filters map[string]string `json:"filters"`
		Count   int               `json:"count"`
		Entries []struct {
			Email string `json:"email"`
		} `json:"entries"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &doc2); err != nil {
		t.Fatalf("narrowed audit JSON parse: %v", err)
	}
	if doc2.Filters["q"] != "alice" {
		t.Errorf("narrowed audit JSON must echo its q filter, got %v", doc2.Filters)
	}
	if doc2.Count != len(doc2.Entries) || doc2.Count == 0 {
		t.Fatalf("narrowed audit JSON count/entries disagree: %d vs %d", doc2.Count, len(doc2.Entries))
	}
	for _, e := range doc2.Entries {
		if e.Email != "alice@x.dev" {
			t.Errorf("q=alice audit JSON must only carry alice entries, got %q", e.Email)
		}
	}
	_ = svc // the service's window is read through the handler; kept for clarity
}

// TestPanel_ExportButtons_StateCarrying pins the toolbar wiring: the users
// filtered-export buttons render only while the view is narrowed and carry
// the current filter set; the audit export buttons are always present
// (the window always exists) and carry the active filters.
func TestPanel_ExportButtons_StateCarrying(t *testing.T) {
	t.Parallel()
	h, _ := exportHarness(t)

	get := func(path string) string {
		t.Helper()
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s: status %d", path, rec.Code)
		}
		return rec.Body.String()
	}

	// Unfiltered users page: only the full export button, no filtered twins.
	page := get("/admin/users")
	if strings.Contains(page, "export.filtered") {
		t.Error("unfiltered users page must not offer the filtered exports")
	}
	if !strings.Contains(page, "/users/export.csv") {
		t.Error("users page must keep the full export button")
	}

	// A narrowed view carries its filter set into the export hrefs (the URL
	// builder emits params alphabetically; templ escapes & in attributes).
	narrowed := get("/admin/users?q=alice&filter=unverified&sort=email")
	for _, want := range []string{
		"/users/export.filtered.csv?filter=unverified&amp;q=alice&amp;sort=email",
		"/users/export.filtered.json?filter=unverified&amp;q=alice&amp;sort=email",
	} {
		if !strings.Contains(narrowed, want) {
			t.Errorf("narrowed users page missing export href %q", want)
		}
	}

	// The audit page always offers the window exports, filters included.
	audit := get("/admin/audit?actor=user")
	if !strings.Contains(audit, "/audit/export.csv?actor=user") {
		t.Error("audit page export button must carry the actor chip")
	}
	if !strings.Contains(audit, "/audit/export.json?actor=user") {
		t.Error("audit page JSON export button must carry the actor chip")
	}
}
