package adminui

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/a-h/templ"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
)

// renderComponent renders a templ component to a string for assertions.
func renderComponent(t *testing.T, component templ.Component) string {
	t.Helper()

	var buf bytes.Buffer

	if err := component.Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}

	return buf.String()
}

// TestExternalAccountsCard_RendersLinkUnlinkView verifies the populated card:
// each linked provider shows an Unlink button with the correct hx-post URL, and
// configured (unlinked) providers appear as badges. This is the view test for
// the "Admin UI: OAuth2 link/unlink views" TODO item.
func TestExternalAccountsCard_RendersLinkUnlinkView(t *testing.T) {
	u := mustUser(t, "linked@example.com")
	u.ExternalAccounts = []usermgmt.ExternalAccount{
		usermgmt.NewExternalAccount("google", "sub-123", "g@x.com", "G User", time.Now()),
	}

	data := userDetailData{
		User:                u,
		BasePath:            "/admin",
		UnlinkExternalBase:  "/admin/users/" + u.ID.Get().String() + "/external",
		ConfiguredProviders: []string{"google", "github"},
	}

	html := renderComponent(t, externalAccountsCard(data))

	wantUnlinkURL := "/admin/users/" + u.ID.Get().String() + "/external/google/unlink"
	if !strings.Contains(html, wantUnlinkURL) {
		t.Errorf("expected unlink hx-post URL %q in HTML", wantUnlinkURL)
	}

	if !strings.Contains(html, "Unlink") {
		t.Error("expected 'Unlink' button label")
	}

	if !strings.Contains(html, "g@x.com") {
		t.Error("expected linked email in card")
	}

	for _, provider := range []string{"google", "github"} {
		if !strings.Contains(html, provider) {
			t.Errorf("expected configured provider badge %q", provider)
		}
	}

	if !strings.Contains(html, "Linking is done via the user's own OAuth2 login flow") {
		t.Error("expected help text explaining linking is user-side")
	}
}

// TestExternalAccountsCard_EmptyState renders the no-accounts note when the
// user has no linked providers.
func TestExternalAccountsCard_EmptyState(t *testing.T) {
	data := userDetailData{
		User:     mustUser(t, "bare@example.com"),
		BasePath: "/admin",
	}

	html := renderComponent(t, externalAccountsCard(data))

	if !strings.Contains(html, "No external accounts linked.") {
		t.Error("expected empty-state note")
	}
}

// TestPanel_UserDetailShowsExternalCard verifies the full page render includes
// the external-accounts card heading for a registered user (empty state).
func TestPanel_UserDetailShowsExternalCard(t *testing.T) {
	ctx := context.Background()
	admin := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, admin)

	created, err := svc.Register(ctx, usermgmt.RegisterRequest{
		ID:    usermgmt.SyntheticUserID("u-dave@x.dev"),
		Email: "dave@x.dev",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/users/"+created.User.ID.Get().String(), nil))

	body := rec.Body.String()
	if !strings.Contains(body, "External accounts") {
		t.Error("user detail page missing 'External accounts' card heading")
	}
	if !strings.Contains(body, "No external accounts linked.") {
		t.Error("user detail page missing empty-state note for unlinked user")
	}
}

// TestPanel_UnlinkRouteWired verifies the unlink route is mounted and calls the
// Service. For a user with no matching provider link, UnlinkExternalAccount
// returns an error, so the handler responds 400 with an error toast. This
// proves the route → handler → service path is wired end-to-end.
func TestPanel_UnlinkRouteWired(t *testing.T) {
	ctx := context.Background()
	admin := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, admin)

	created, err := svc.Register(ctx, usermgmt.RegisterRequest{
		ID:    usermgmt.SyntheticUserID("u-eve@x.dev"),
		Email: "eve@x.dev",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	rec := httptest.NewRecorder()
	unlinkURL := "/admin/users/" + created.User.ID.Get().String() + "/external/google/unlink"
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, unlinkURL, nil))

	if rec.Code != http.StatusBadRequest {
		t.Errorf("unlink (no matching provider): status = %d, want 400; body=%s",
			rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Header().Get("HX-Trigger"), "adminui:toast") {
		t.Error("expected error toast HX-Trigger header")
	}
}

// TestPanel_UnlinkInvalidUserID verifies the unlink handler rejects a malformed
// user id with 400 (and does NOT call the service).
func TestPanel_UnlinkInvalidUserID(t *testing.T) {
	admin := mustUser(t, "admin@example.com")
	h, _ := newTestPanel(t, admin)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/admin/users/not-a-ulid/external/google/unlink", nil))

	if rec.Code != http.StatusBadRequest {
		t.Errorf("invalid user id: status = %d, want 400", rec.Code)
	}
}
