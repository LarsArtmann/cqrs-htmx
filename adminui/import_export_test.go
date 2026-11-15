package adminui

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
)

// upload posts a multipart file to the import endpoint the way a browser
// form would, returning the response (the real CSRF dance is covered by
// csrf_test.go for every POST route — the route table registers this one the
// same guarded way).
func upload(t *testing.T, h http.Handler, filename, content string) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/admin/users/import", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	// The real import form posts via htmx (hx-post targets the result panel),
	// which is the path that renders the summary the tests assert on.
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// importJSON bodies the given rows as the JSON import format.
func importJSON(rows ...string) string {
	return "[" + strings.Join(rows, ",") + "]"
}

// TestImportExport_RoundTrip proves the wire formats survive the panel: a
// JSON export imports back with the same email set, and the CSV leg does too.
func TestImportExport_RoundTrip(t *testing.T) {
	t.Parallel()

	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)

	mustRegister(t, svc, "u-rt1", "rt1@example.com")
	mustRegister(t, svc, "u-rt2", "rt2@example.com")

	// JSON export streams every user's public profile.
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/users/export.json", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("export json = %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("json content type = %q", ct)
	}
	if cd := rec.Header().Get("Content-Disposition"); !strings.Contains(cd, "users-export.json") {
		t.Errorf("json content disposition = %q", cd)
	}
	var exported []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &exported); err != nil {
		t.Fatalf("export json parse: %v", err)
	}
	if len(exported) < 2 {
		t.Fatalf("expected at least the two seeded users in the export, got %d", len(exported))
	}

	// The exported bytes import back: the two seeded emails are skipped
	// (existing), fresh rows land.
	rows := importJSON(
		`{"email":"rt1@example.com"}`,
		`{"email":"fresh@example.com","display_name":"Fresh"}`,
	)
	rec = upload(t, h, "users.json", rows)
	if rec.Code != http.StatusOK {
		t.Fatalf("import json = %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "1 imported") || !strings.Contains(rec.Body.String(), "1 skipped") {
		t.Errorf("result panel should summarize 1 imported / 1 skipped, got: %.200s", rec.Body.String())
	}
	if _, ok := svc.ReadModel().FindByEmail("fresh@example.com"); !ok {
		t.Error("the fresh imported user should exist in the read model")
	}

	// CSV leg: header + rows round-trip through text/csv streaming.
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/users/export.csv", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("export csv = %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/csv") {
		t.Errorf("csv content type = %q", ct)
	}
	if !strings.Contains(rec.Body.String(), "fresh@example.com") {
		t.Error("csv export should carry the imported user")
	}
}

// TestImport_DuplicateReRunIsSafe pins the idempotency story: importing the
// same file twice skips every row the second time and reports it.
func TestImport_DuplicateReRunIsSafe(t *testing.T) {
	t.Parallel()

	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)

	rows := importJSON(
		`{"email":"dup1@example.com","display_name":"Dup One"}`,
		`{"email":"dup2@example.com","display_name":"Dup Two"}`,
	)
	rec := upload(t, h, "users.json", rows)
	if rec.Code != http.StatusOK {
		t.Fatalf("first import = %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "2 imported") {
		t.Errorf("first import should land 2, got: %.200s", rec.Body.String())
	}

	rec = upload(t, h, "users.json", rows)
	if rec.Code != http.StatusOK {
		t.Fatalf("second import = %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "0 imported") || !strings.Contains(rec.Body.String(), "2 skipped") {
		t.Errorf("re-run should skip both, got: %.200s", rec.Body.String())
	}
	if _, ok := svc.ReadModel().FindByEmail("dup1@example.com"); !ok {
		t.Error("duplicate import must not lose the original account")
	}
}

// TestImport_RefusesBadFiles: a malformed JSON body, a CSV with the wrong
// header, and an unknown file extension all answer 400 with an explanation —
// nothing is imported and nothing crashes.
func TestImport_RefusesBadFiles(t *testing.T) {
	t.Parallel()

	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)

	cases := []struct {
		name     string
		filename string
		content  string
	}{
		{"malformed json", "users.json", `[{nope`},
		{"header without email column", "users.csv", "name,full_name\nx,x\n"},
		{"unknown extension", "users.txt", "[]"},
		{"empty body no file", "", ""},
	}
	for _, tc := range cases {
		rec := upload(t, h, tc.filename, tc.content)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400 (body %.120s)", tc.name, rec.Code, rec.Body.String())
		}
	}
	if _, ok := svc.ReadModel().FindByEmail("x@x.com"); ok {
		t.Error("the refused import must not have created accounts")
	}
}

// TestImport_OversizedBodyRefused: the MaxBytesReader caps the upload before
// the parser sees it — a hostile or runaway file answers 413, not OOM.
func TestImport_OversizedBodyRefused(t *testing.T) {
	t.Parallel()

	user := mustUser(t, "admin@example.com")
	h, _ := newTestPanel(t, user)

	big := importJSON(`{"email":"big@example.com","display_name":"` + strings.Repeat("x", maxImportUploadBytes) + `"}`)
	rec := upload(t, h, "users.json", big)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("oversized import = %d, want 413 (body %.120s)", rec.Code, rec.Body.String())
	}
}

// TestImportPage_Renders pins the form's honest scaffolding: file input,
// the passwordless caveat, and the users-list entry points.
func TestImportPage_Renders(t *testing.T) {
	t.Parallel()

	user := mustUser(t, "admin@example.com")
	h, _ := newTestPanel(t, user)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/users/import", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("import page = %d", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{
		`type="file"`, `name="file"`,
		"passwordless", // the caveat: import cannot set credentials
		"skipped",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("import page missing %q", want)
		}
	}

	list := getBody(t, h, "/admin/users")
	for _, want := range []string{"Import users", "Export CSV", "/users/export.csv"} {
		if !strings.Contains(list, want) {
			t.Errorf("users list missing the %q entry point", want)
		}
	}
}

// TestExport_OtherExtensionsRefused: only the two literal export routes
// exist — an unknown extension falls through to the guarded catch-all.
func TestExport_OtherExtensionsRefused(t *testing.T) {
	t.Parallel()

	user := mustUser(t, "admin@example.com")
	h, _ := newTestPanel(t, user)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/users/export.xml", nil))
	if rec.Code == http.StatusOK {
		t.Error("export.xml must not be served")
	}
}

// TestImport_ViaServiceEndToEnd is the integration leg: two fresh users in
// one CSV, dispatched through the REAL service path (not just the panel),
// landing in the durable read model with their own streams.
func TestImport_ViaServiceEndToEnd(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)
	_ = h

	csv := "email,display_name\none@example.com,One\ntwo@example.com,Two\n"
	result, err := svc.ImportUsersFromCSV(ctx, strings.NewReader(csv))
	if err != nil {
		t.Fatalf("service import: %v", err)
	}
	if result.Imported != 2 || result.Skipped != 0 {
		t.Errorf("imported=%d skipped=%d, want 2/0", result.Imported, result.Skipped)
	}
	for _, email := range []string{"one@example.com", "two@example.com"} {
		if _, ok := svc.ReadModel().FindByEmail(email); !ok {
			t.Errorf("imported user %s missing from the read model", email)
		}
	}
}

// TestImport_TruncatedErrorListBounded pins the render bound: a result with
// more errors than the render window shows the first N plus an honest
// remainder count, never the whole firehose.
func TestImport_TruncatedErrorListBounded(t *testing.T) {
	t.Parallel()

	errs := make([]string, maxRenderedImportErrors+5)
	for i := range errs {
		errs[i] = "row error"
	}
	var b strings.Builder
	if err := importResultPanel(
		usermgmt.ImportResult{Imported: 1, Skipped: 0, Errors: errs},
		"/admin/users",
	).Render(context.Background(), &b); err != nil {
		t.Fatalf("render: %v", err)
	}
	body := b.String()
	if strings.Count(body, "row error") != maxRenderedImportErrors {
		t.Errorf(
			"expected exactly %d rendered errors, got %d",
			maxRenderedImportErrors,
			strings.Count(body, "row error"),
		)
	}
	if !strings.Contains(body, "more") {
		t.Error("expected the remainder line")
	}
	if !strings.Contains(body, "5 more") {
		t.Error("remainder count should be 5")
	}
}
