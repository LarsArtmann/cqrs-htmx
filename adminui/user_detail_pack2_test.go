package adminui

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
)

// User-detail pack 2 (F167–F169, F171): the OAuth identity card's enrichment
// (provider glyph, subject id, locale linked-at), the verification-TTL row's
// honest "consumer-configured" answer, and the self-service card that renders
// only on the signed-in admin's own page. F170 (credential restore from
// backup JSON) is CUT: the published Service method
// (AddCredential(ctx, UserID, WebAuthnCredential)) and the fully JSON-tagged
// identitymodel.WebAuthnCredential make the proxy honest in principle, but
// wiring it needs a POST route this task is not allowed to add, and a card
// without a route is dead UI.

func pack2User(t *testing.T) *identitymodel.User {
	t.Helper()
	u := mustUser(t, "pack2@example.com")
	u.ExternalAccounts = []identitymodel.ExternalAccount{
		identitymodel.NewExternalAccount(
			"google",
			"109876543210987654321-long-subject",
			"g@x.com", "G User",
			time.Date(2026, 11, 24, 10, 30, 0, 0, time.UTC),
		),
	}
	return u
}

// TestOAuthCard_EnrichedIdentity pins F167: each linked row names the provider
// with a glyph, shows the subject id (truncated for display, full id in the
// title attribute — the value is never rewritten), and carries the linked-at
// timestamp rendered through the locale seam (p.dateTime), not a raw Format.
func TestOAuthCard_EnrichedIdentity(t *testing.T) {
	d := userDetailData{
		User:                pack2User(t),
		BasePath:            "/admin",
		ConfiguredProviders: []string{"google"},
	}

	html := renderComponent(t, externalAccountsCard(pageData{}, d))

	if !strings.Contains(html, "Subject") {
		t.Error("expected a Subject column header")
	}
	if !strings.Contains(html, "Subject: 109876543210987654321-long-subject") {
		t.Error("expected the FULL subject id in the title attribute")
	}
	// Truncated display: first 14 chars + ellipsis, never the whole value.
	if !strings.Contains(html, "10987654321098&hellip;") && !strings.Contains(html, "10987654321098…") {
		t.Errorf("expected truncated subject display in HTML: %s", htmlSnippet(html, "font-mono"))
	}
	// The provider cell pairs the generic link glyph (an inline SVG) with the
	// provider badge; no brand mark exists, so the generic glyph is the
	// honest stand-in.
	if !strings.Contains(html, "<svg") {
		t.Error("expected the provider glyph SVG in the identity row")
	}
	if !strings.Contains(html, "google") {
		t.Error("expected provider name in the identity row")
	}
	// Linked-at goes through p.dateTime (locale layout + render location);
	// the OR of the three default-locale shapes keeps the pin
	// version-stable without pinning one layout spelling.
	for _, shape := range []string{"11/24/2026", "Nov 24, 2026", "2026-11-24"} {
		if strings.Contains(html, shape) {
			return
		}
	}
	t.Error("expected a locale-shaped linked-at timestamp")
	if strings.Contains(html, `LinkedAt.Format`) {
		t.Error("linked-at must not bypass the locale seam")
	}
}

// TestOAuthCard_ShortSubjectUntruncated pins the boundary: a subject at or
// under the display width renders whole, with no ellipsis.
func TestOAuthCard_ShortSubjectUntruncated(t *testing.T) {
	u := mustUser(t, "shortsub@example.com")
	u.ExternalAccounts = []identitymodel.ExternalAccount{
		identitymodel.NewExternalAccount("github", "short-sub", "gh@x.com", "GH", time.Now()),
	}
	d := userDetailData{User: u, BasePath: "/admin"}

	html := renderComponent(t, externalAccountsCard(pageData{}, d))
	if !strings.Contains(html, "short-sub") {
		t.Error("expected the short subject rendered in full")
	}
	if strings.Contains(html, "short-sub&hellip;") || strings.Contains(html, "short-sub…") {
		t.Error("short subject must not be truncated")
	}
}

// TestVerificationTTLRow_ConsumerConfigured pins F168 at helper and render
// level: the Details card says the token lifetime is consumer-configured and
// names the library default — it never displays a concrete configured TTL,
// because the published Service surface exposes none.
func TestVerificationTTLRow_ConsumerConfigured(t *testing.T) {
	if got := verificationTTLCopy(); !strings.Contains(got, "Consumer-configured") || !strings.Contains(got, "24h") {
		t.Errorf("verificationTTLCopy = %q, want consumer-configured honesty with the default named", got)
	}

	d := userDetailData{
		User:     mustUser(t, "ttl@example.com"),
		BasePath: "/admin",
	}
	html := renderComponent(t, userDetailContent(pageData{}, d))
	if !strings.Contains(html, "Verification token") {
		t.Error("expected a Verification token row in the Details card")
	}
	if !strings.Contains(html, "Consumer-configured") {
		t.Error("expected the consumer-configured copy on the page")
	}
	// No invented value: the only number the row may carry is the library
	// default, clearly labeled as such.
	if strings.Contains(html, "TTL: 24h") || strings.Contains(html, "expires in") {
		t.Error("verification row invented a configured TTL")
	}
}

// TestSelfServiceCard_RendersForSelf pins F169 on the signed-in admin's own
// page: the card renders with the display-name form (distinct input id, same
// existing route), the grants read-back, and the sign-out link to the
// consumer-mounted endpoint.
func TestSelfServiceCard_RendersForSelf(t *testing.T) {
	d := userDetailData{
		User:          mustUser(t, "self@example.com"),
		BasePath:      "/admin",
		IsSelf:        true,
		PlatformRoles: []identitymodel.Role{identitymodel.RoleSuperAdmin},
		SignOutURL:    "/logout",
	}
	d.User.DisplayName = "Self Admin"

	html := renderComponent(t, userDetailContent(pageData{}, d))

	for _, want := range []string{
		"Your account",
		`id="self_display_name"`,
		"/admin/users/" + d.User.ID.Get().String() + "/display-name",
		"Your platform roles",
		"super_admin",
		"You are not a member of any tenant.",
		"Sign out",
		`href="/logout"`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("self-service card missing %q", want)
		}
	}
}

// TestSelfServiceCard_AbsentForOthers pins the gate: another admin's detail
// page never renders the self-service card or its sign-out affordance.
func TestSelfServiceCard_AbsentForOthers(t *testing.T) {
	d := userDetailData{
		User:       mustUser(t, "other@example.com"),
		BasePath:   "/admin",
		IsSelf:     false,
		SignOutURL: "/logout",
	}
	html := renderComponent(t, userDetailContent(pageData{}, d))
	for _, absent := range []string{"Your account", `id="self_display_name"`, "Your platform roles"} {
		if strings.Contains(html, absent) {
			t.Errorf("self-service card leaked onto another user's page: %q", absent)
		}
	}
}

// TestSelfServiceCard_NoSignOutConfiguredIsHonest pins the honest fallback:
// with no consumer-mounted sign-out URL the card names the missing
// Config.LogoutURL instead of rendering a link that would 404.
func TestSelfServiceCard_NoSignOutConfiguredIsHonest(t *testing.T) {
	d := userDetailData{
		User:     mustUser(t, "nosignout@example.com"),
		BasePath: "/admin",
		IsSelf:   true,
	}
	html := renderComponent(t, selfServiceCard(d))
	if !strings.Contains(html, signOutAbsentCopy()) {
		t.Error("expected the honest no-sign-out copy")
	}
	if strings.Contains(html, `href="/logout"`) {
		t.Error("no sign-out link may render without a configured URL")
	}
}

// TestSelfServiceCard_GrantsListed walks the real handler: the signed-in
// admin's own detail page carries the self-service card with their sign-out
// link (the test panel's Config.LogoutURL), while a second admin viewing the
// same user does not.
func TestSelfServiceCard_GrantsListed(t *testing.T) {
	admin := mustUser(t, "walker@example.com")
	svc := newPack2Service(t)
	if _, err := svc.Register(context.Background(), usermgmt.RegisterRequest{
		ID: admin.ID, Email: admin.Email,
	}); err != nil {
		t.Fatalf("register: %v", err)
	}
	h, _ := newTestPanel(t, admin, Config{Service: svc, LogoutURL: "/logout-test"})

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/users/"+admin.ID.Get().String(), nil))
	body := rec.Body.String()
	if !strings.Contains(body, "Your account") || !strings.Contains(body, `href="/logout-test"`) {
		t.Errorf("own detail page missing self-service card or sign-out link; status=%d", rec.Code)
	}

	viewer := mustUser(t, "viewer@example.com")
	svc2 := newPack2Service(t)
	if _, err := svc2.Register(context.Background(), usermgmt.RegisterRequest{
		ID: viewer.ID, Email: viewer.Email,
	}); err != nil {
		t.Fatalf("register viewer: %v", err)
	}
	h2, _ := newTestPanel(t, viewer, Config{Service: svc2, LogoutURL: "/logout-test"})
	rec2 := httptest.NewRecorder()
	h2.ServeHTTP(rec2, httptest.NewRequest(http.MethodGet, "/admin/users/"+admin.ID.Get().String(), nil))
	if strings.Contains(rec2.Body.String(), "Your account") {
		t.Error("self-service card rendered for a viewer of someone else's page")
	}
}

// TestPanel_HandlerPopulatesSignOutURL pins the wiring: with an unset
// Config.LogoutURL, the signed-in admin's own detail page renders the honest
// absence copy instead of a sign-out link.
func TestPanel_HandlerPopulatesSignOutURL(t *testing.T) {
	admin := mustUser(t, "wire@example.com")
	svc := newPack2Service(t)
	if _, err := svc.Register(context.Background(), usermgmt.RegisterRequest{
		ID: admin.ID, Email: admin.Email,
	}); err != nil {
		t.Fatalf("register: %v", err)
	}
	h, _ := newTestPanel(t, admin, Config{Service: svc})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/users/"+admin.ID.Get().String(), nil))
	if !strings.Contains(rec.Body.String(), signOutAbsentCopy()) {
		t.Errorf("unset LogoutURL must render the honest absence copy; status=%d", rec.Code)
	}
}

// newPack2Service builds the bare in-memory service the handler-level pack-2
// tests register their users into.
func newPack2Service(t *testing.T) *usermgmt.Service {
	t.Helper()
	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{AuditLog: usermgmt.NewAuditLog()})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	return svc
}

// htmlSnippet returns the ~120 chars around the first occurrence of needle —
// a failure-message aid, not an assertion.
func htmlSnippet(html, needle string) string {
	i := strings.Index(html, needle)
	if i < 0 {
		return "(not found)"
	}
	start := i - 40
	if start < 0 {
		start = 0
	}
	end := i + 120
	if end > len(html) {
		end = len(html)
	}
	return html[start:end]
}
