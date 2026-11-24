package adminui

import (
	"context"
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
)

// suggestRow mirrors the endpoint's JSON shape for decoding in tests.
type suggestRow struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
}

// getSuggest hits the GET rung (the fallback twin) and returns the decoded
// rows plus the response.
func getSuggest(t *testing.T, h http.Handler, q string) ([]suggestRow, *httptest.ResponseRecorder) {
	t.Helper()

	target := "/admin/users/suggest"
	if q != "" {
		target += "?q=" + url.QueryEscape(q)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
	if rec.Code != http.StatusOK {
		return nil, rec
	}
	var rows []suggestRow
	if err := json.Unmarshal(rec.Body.Bytes(), &rows); err != nil {
		t.Fatalf("decode suggest response: %v (body %.200s)", err, rec.Body.String())
	}
	return rows, rec
}

// TestUserSuggest_Shape pins the response contract: a JSON array of objects
// carrying id, email, and display_name — enough to label a popup option and
// fill the form field, nothing more (no tenant map, no audit tail).
func TestUserSuggest_Shape(t *testing.T) {
	t.Parallel()

	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)

	registered := mustRegister(t, svc, "u-ada", "ada@example.com")

	rows, rec := getSuggest(t, h, "ada")
	if rec.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", rec.Header().Get("Content-Type"))
	}
	if len(rows) != 1 {
		t.Fatalf("got %d rows, want 1: %+v", len(rows), rows)
	}
	if rows[0].Email != "ada@example.com" {
		t.Errorf("email = %q", rows[0].Email)
	}
	if rows[0].ID != registered.User.ID.Get().String() {
		t.Errorf("id = %q, want the user id", rows[0].ID)
	}
}

// TestUserSuggest_PrefixMatchOnEmailAndDisplayName proves the match is a
// case-insensitive PREFIX on either field — a substring hit is not a match.
func TestUserSuggest_PrefixMatchOnEmailAndDisplayName(t *testing.T) {
	t.Parallel()

	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)

	mustRegister(t, svc, "u-1", "bridge@example.com")

	// A display-name prefix: register a user whose display name starts with
	// the query while the email does not.
	ctx := context.Background()
	if _, err := svc.Register(ctx, usermgmt.RegisterRequest{
		ID:          identitymodel.SyntheticUserID("u-2"),
		Email:       "zz-other@example.com",
		DisplayName: "Quinn Arbuckle",
	}); err != nil {
		t.Fatal(err)
	}

	if rows, _ := getSuggest(t, h, "BRID"); len(rows) != 1 || rows[0].Email != "bridge@example.com" {
		t.Errorf("case-insensitive email prefix failed: %+v", rows)
	}
	if rows, _ := getSuggest(t, h, "quinn"); len(rows) != 1 || rows[0].Email != "zz-other@example.com" {
		t.Errorf("display-name prefix failed: %+v", rows)
	}
	if rows, _ := getSuggest(t, h, "idge"); len(rows) != 0 {
		t.Errorf("substring must not match a prefix search: %+v", rows)
	}
}

// TestUserSuggest_CappedAtTwenty proves the response never grows with the
// directory: twenty-one matches yield exactly the cap (suggestLimit).
func TestUserSuggest_CappedAtTwenty(t *testing.T) {
	t.Parallel()

	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)

	ctx := context.Background()
	for i := range suggestLimit + 1 {
		if _, err := svc.Register(ctx, usermgmt.RegisterRequest{
			ID:    identitymodel.SyntheticUserID("u-cap-" + strings.Repeat("x", i+1)),
			Email: "cap" + strings.Repeat("x", i+1) + "@example.com",
		}); err != nil {
			t.Fatal(err)
		}
	}

	rows, _ := getSuggest(t, h, "cap")
	if len(rows) != suggestLimit {
		t.Fatalf("got %d rows, want the cap %d", len(rows), suggestLimit)
	}
}

// TestUserSuggest_EmptyQueryRefused proves the honest refusal: a missing or
// blank q is a named 400, not a whole-directory dump.
func TestUserSuggest_EmptyQueryRefused(t *testing.T) {
	t.Parallel()

	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)
	mustRegister(t, svc, "u-1", "one@example.com")

	for name, target := range map[string]string{
		"missing q": "/admin/users/suggest",
		"blank q":   "/admin/users/suggest?q=",
		"spaces q":  "/admin/users/suggest?q=%20%20",
	} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
		if rec.Code != http.StatusBadRequest {
			t.Errorf("%s: status %d, want 400", name, rec.Code)
			continue
		}
		if !strings.Contains(rec.Body.String(), "error") {
			t.Errorf("%s: refusal should name the error in JSON, got %.120s", name, rec.Body.String())
		}
	}
}

// TestUserSuggest_QueryRungParity proves the QUERY rung answers byte-identically
// to the GET twin, same as the search ladder.
func TestUserSuggest_QueryRungParity(t *testing.T) {
	t.Parallel()

	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)
	mustRegister(t, svc, "u-1", "parity@example.com")

	recQuery := httptest.NewRecorder()
	reqQuery := httptest.NewRequest(http.MethodGet, "/admin/users/suggest?q=parity", nil)
	reqQuery.Method = "QUERY"
	h.ServeHTTP(recQuery, reqQuery)

	recGet := httptest.NewRecorder()
	h.ServeHTTP(recGet, httptest.NewRequest(http.MethodGet, "/admin/users/suggest?q=parity", nil))

	if recQuery.Code != http.StatusOK {
		t.Fatalf("QUERY rung status %d", recQuery.Code)
	}
	if recQuery.Body.String() != recGet.Body.String() {
		t.Errorf("QUERY and GET rungs drifted:\nQUERY: %s\nGET:   %s", recQuery.Body.String(), recGet.Body.String())
	}
}

// TestUserSuggest_TenantModeAbsent proves the endpoint is super-admin-only:
// the tenant-admin panel mounts no Users routes, and a probe answers the
// guarded 404, not the directory.
func TestUserSuggest_TenantModeAbsent(t *testing.T) {
	t.Parallel()

	panel, err := New(Config{
		Service:  mustService(t),
		Mode:     ModeTenantAdmin,
		TenantID: identitymodel.NewTenantID("t-1"),
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	rec := httptest.NewRecorder()
	panel.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/users/suggest?q=x", nil))
	if rec.Code == http.StatusOK {
		t.Fatal("tenant mode must not serve /users/suggest")
	}
}

// TestSuggest_JSWired pins the client half (T08 F052/F054, panelAdminJS
// pattern): admin.js must own the combobox behavior — debounce, keyboard
// selection, pick, close — document-delegated and CSP-safe (no inline
// handlers), and the members page must carry the hook markers.
func TestSuggest_JSWired(t *testing.T) {
	t.Parallel()

	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)
	mustCreateTenant(t, svc, "acme")

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/tenants/acme", nil))
	body := rec.Body.String()
	for _, want := range []string{"data-suggest-input", "data-suggest-list", `role="combobox"`, `aria-expanded="false"`} {
		if !strings.Contains(body, want) {
			t.Errorf("members page missing combobox marker %q", want)
		}
	}

	js := panelAdminJS(t)
	for _, want := range []string{
		"data-suggest-input",    // delegation hook
		"setTimeout",            // debounce
		"aria-activedescendant", // selection semantics
		"ArrowDown",             // keyboard selection
		"Escape",                // close
		"?q=",                   // the query param the fetch targets
	} {
		if !strings.Contains(js, want) {
			t.Errorf("admin.js missing suggest hook %q", want)
		}
	}
}

// TestUsersList_TwoRolesOneTenant proves the access column counts DOMAINS,
// not policy rows (T08-adjacent fix): one membership granted two roles used
// to read "2 tenants" because each role appended the same domain to the
// render cache.
func TestUsersList_TwoRolesOneTenant(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)
	tenant := mustCreateTenant(t, svc, "acme")
	member := mustRegister(t, svc, "u-multi", "multi@example.com")
	mustAddMember(t, svc, member.User.ID, tenant.ID, identitymodel.RoleUser)

	// Add a second role to the SAME membership: the badge must stay "1 tenant".
	if _, err := svc.Dispatch(ctx, usermgmt.NewAddMemberToTenant(
		identitymodel.ActorIDFromUser(member.User.ID), member.User.ID, tenant.ID, identitymodel.RoleViewer,
	)); err != nil {
		t.Fatalf("add second role: %v", err)
	}

	body := renderUsersIndex(t, h, "/admin/users?q=multi@example.com")
	if strings.Contains(body, "2 tenants") {
		t.Error("two roles in one tenant must not read as '2 tenants'")
	}
	if !strings.Contains(body, "1 tenant") {
		t.Error("expected the '1 tenant' badge")
	}
}
