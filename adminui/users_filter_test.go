package adminui

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
)

func TestFlow_UsersFilterChips(t *testing.T) {
	ctx := context.Background()
	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)

	for _, s := range []struct{ id, email string }{
		{"u-v1", "fresh1@example.com"},
		{"u-v2", "fresh2@example.com"},
	} {
		if _, err := svc.Register(ctx, usermgmt.RegisterRequest{
			ID: identitymodel.SyntheticUserID(s.id), Email: s.email,
		}); err != nil {
			t.Fatal(err)
		}
	}

	// Both seeds are unverified (no verification configured on the Service),
	// so the unverified chip keeps both and the verified chip empties the list.
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/users?filter=unverified", nil))
	body := rec.Body.String()
	if !strings.Contains(body, "fresh1@example.com") || !strings.Contains(body, "fresh2@example.com") {
		t.Error("unverified filter must keep unverified users")
	}
	for _, chip := range []string{"Verified", "Unverified", "MFA on", "Has provider"} {
		if !strings.Contains(body, chip) {
			t.Errorf("users page missing the %q chip", chip)
		}
	}

	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, httptest.NewRequest(http.MethodGet, "/admin/users?filter=verified", nil))
	if strings.Contains(rec2.Body.String(), "fresh1@example.com") {
		t.Error("verified filter must exclude unverified users")
	}
}

func TestFlow_UsersFilterComposeWithSearch(t *testing.T) {
	ctx := context.Background()
	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)

	for _, s := range []struct{ id, email string }{
		{"u-comp1", "alpha@example.com"},
		{"u-comp2", "beta@example.com"},
	} {
		if _, err := svc.Register(ctx, usermgmt.RegisterRequest{
			ID: identitymodel.SyntheticUserID(s.id), Email: s.email,
		}); err != nil {
			t.Fatal(err)
		}
	}

	// filter=mfa keeps neither (TOTP off for both) — but the URL must survive.
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/users?q=alpha&filter=mfa", nil))
	body := rec.Body.String()
	if strings.Contains(body, "alpha@example.com") {
		t.Error("mfa filter must exclude users without TOTP")
	}
	if !strings.Contains(body, "No users match") && !strings.Contains(body, "No users found") {
		t.Error("empty filtered result should render an empty state")
	}
}

func TestUserFilterChipsList_PreservesSearch(t *testing.T) {
	d := usersListData{BasePath: "/admin", Search: "ada", Filter: userFilterVerified}
	chips := userFilterChipsList(d)
	// Five status chips: verified, unverified, MFA, has-provider, administrators.
	if len(chips) != 5 {
		t.Fatalf("got %d chips, want 5", len(chips))
	}
	active := 0
	for _, c := range chips {
		if c.Active {
			active++
			if !strings.Contains(c.Href, "q=ada") {
				t.Errorf("active chip must keep the search: %q", c.Href)
			}
			if strings.Contains(c.Href, "filter=") {
				t.Errorf("active chip must toggle the filter OFF: %q", c.Href)
			}
		} else if !strings.Contains(c.Href, "q=ada") || !strings.Contains(c.Href, "filter=") {
			t.Errorf("inactive chip must keep search and add filter: %q", c.Href)
		}
	}
	if active != 1 {
		t.Errorf("expected exactly 1 active chip, got %d", active)
	}
}

func TestParseUserFilter_RefusesUnknown(t *testing.T) {
	for raw, want := range map[string]userFilter{
		"":           userFilterNone,
		"verified":   userFilterVerified,
		"mfa":        userFilterMFA,
		"external":   userFilterExternal,
		"unverified": userFilterUnverified,
		"admin":      userFilterAdmin,
		"garbage":    userFilterNone,
		"<script>":   userFilterNone,
	} {
		if got := parseUserFilter(raw); got != want {
			t.Errorf("parseUserFilter(%q) = %q, want %q", raw, got, want)
		}
	}
}
