package adminui

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
)

// registerClientPackUser registers one user through the Service so the
// directory has a row to render (the test-panel service starts empty).
func registerClientPackUser(t *testing.T, svc *usermgmt.Service, id, email string) {
	t.Helper()
	if _, err := svc.Register(context.Background(), usermgmt.RegisterRequest{
		ID: identitymodelSynthetic(id), Email: email,
	}); err != nil {
		t.Fatalf("register %s: %v", email, err)
	}
}

// identitymodelSynthetic is a narrow alias so this file stands alone without
// importing identity-model twice.
func identitymodelSynthetic(id string) interface{} {
	return identityUserID(id)
}

// TestClientPack_SkeletonHiddenInServerRender pins F124's honesty rule: the
// skeleton region is server-rendered PRESENT (so the first swap has an
// indicator to light up) but HIDDEN (htmx-indicator opacity convention) —
// a server render of already-loaded data must never visibly show a skeleton.
func TestClientPack_SkeletonHiddenInServerRender(t *testing.T) {
	t.Parallel()
	user := mustUser(t, "skeleton@example.com")
	svc := newTestService(t)
	registerClientPackUser(t, svc, user.ID.Get().String(), user.Email)
	h, _ := newTestPanel(t, user, Config{Service: svc})

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/users", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("users page status = %d", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{
		`id="users-skeleton"`,                    // region present for the first swap
		`class="htmx-indicator"`,                 // …but hidden until a request
		`hx-indicator="#users-skeleton"`,         // F125: the wire targets it
		"admin-skeleton",                         // shimmer bars carry the reduced-motion-guarded class
		`aria-label="Loading users"`,             // announced as a status region
	} {
		if !strings.Contains(body, want) {
			t.Errorf("users page missing %q", want)
		}
	}
}

// TestClientPack_SortableHeaders pins F126's markup contract: the loaded
// window's table carries client-sortable headers (button + data-sort-key +
// aria-sort) with no href (no server round-trip from the header), the
// URL-carried server sort seeds aria-sort ascending on the matching column,
// and rows carry the data-sort-* values the JS reads.
func TestClientPack_SortableHeaders(t *testing.T) {
	t.Parallel()
	user := mustUser(t, "sortable@example.com")
	svc := newTestService(t)
	registerClientPackUser(t, svc, user.ID.Get().String(), user.Email)
	h, _ := newTestPanel(t, user, Config{Service: svc})

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/users?sort=email", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("users page status = %d", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{
		`data-sort-key="email"`,
		`data-sort-key="created"`,
		`aria-sort="ascending"`, // seeded from ?sort=email (the T03 default)
		`aria-sort="none"`,      // the non-active sortable column
		`data-sort-indicator`,
		`data-sort-email="sortable@example.com"`,
		`data-sort-created="`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("sortable markup missing %q", want)
		}
	}
	if strings.Contains(body, `data-sort-key="email" href=`) {
		t.Error("client sort headers must NOT be links — no server round-trip")
	}
}

// TestClientPack_DirtyGuardAttrs pins F127's seam: the detail page's edit
// forms (display name + email, plus the self-service display-name twin)
// carry data-dirty-guard for admin.js's stateless beforeunload guard — at
// least two on any detail page, three on the admin's own.
func TestClientPack_DirtyGuardAttrs(t *testing.T) {
	t.Parallel()
	admin := mustUser(t, "dirty@example.com")
	svc := newTestService(t)
	registerClientPackUser(t, svc, admin.ID.Get().String(), admin.Email)
	h, _ := newTestPanel(t, admin, Config{Service: svc})

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/users/"+admin.ID.Get().String(), nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("detail page status = %d", rec.Code)
	}
	body := rec.Body.String()
	if guards := strings.Count(body, "data-dirty-guard"); guards < 3 {
		t.Errorf("own detail page must guard display-name + email + self-service forms, got %d", guards)
	}
	if !strings.Contains(body, `data-dirty-guard>`) {
		t.Error("guarded form markup missing")
	}
}

// TestClientPack_CopyRowAsJSON pins F128: the detail page carries the
// copy-as-JSON support action whose payload is the SERVER-composed snapshot
// (id, email, display_name, email_verified, created_at) — the client never
// re-derives the facts — plus the honest denial message for a refused
// clipboard.
func TestClientPack_CopyRowAsJSON(t *testing.T) {
	t.Parallel()
	admin := mustUser(t, "copyjson@example.com")
	svc := newTestService(t)
	registerClientPackUser(t, svc, admin.ID.Get().String(), admin.Email)
	h, _ := newTestPanel(t, admin, Config{Service: svc})

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/users/"+admin.ID.Get().String(), nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("detail page status = %d", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{
		`data-copy-json="{&quot;id&quot;:`,
		`"email":"copyjson@example.com"`,
		`"display_name":""`,
		`"email_verified":false`,
		`"created_at":"`,
		"Copy as JSON",
		`data-copy-err=`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("detail page missing %q", want)
		}
	}
}
