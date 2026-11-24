package adminui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// T20 F146–F151 pins: the users list filter bar renders the saved-views +
// display-prefs toolbar, and admin.js carries the namespaced versioned
// localStorage store, the URL-navigation preset contract, and the
// container-class column/density contracts.

func TestSavedViews_PrefsBarRenderedOnUsersList(t *testing.T) {
	t.Parallel()

	user := mustUser(t, "views@example.com")
	h, _ := newTestPanel(t, user)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/users", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /admin/users: status %d", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{
		`data-users-prefs`,               // the prefs toolbar shell
		`data-prefs-save`,                // F146: save UI
		`data-prefs-name`,                // F146: preset name input
		`data-prefs-presets`,             // F147: saved list host
		`data-prefs-col="access"`,        // F148: column visibility
		`data-prefs-col="lastseen"`,      // F148: column visibility
		`data-prefs-density`,             // F149: density toggle
		`data-prefs-base="/admin/users"`, // F147: apply target base
	} {
		if !strings.Contains(body, want) {
			t.Errorf("users list filter bar must render %q", want)
		}
	}
}

func TestSavedViews_StoreContractPinnedInJS(t *testing.T) {
	t.Parallel()

	js := panelAdminJS(t)
	for _, want := range []string{
		// F146: namespaced, versioned localStorage key.
		"cqrshtmx:adminui:users-prefs:v1",
		`"version"`,
		// F147: preset = serialized URL query; apply = navigate.
		"location.search",
		"location.href",
		// F148: container-class column contract (never touches table markup).
		"admin-cols-hide-access",
		"admin-cols-hide-lastseen",
		// F149: density class on the list container.
		"admin-density-compact",
	} {
		if !strings.Contains(js, want) {
			t.Errorf("admin.js missing saved-views hook %q", want)
		}
	}
}
