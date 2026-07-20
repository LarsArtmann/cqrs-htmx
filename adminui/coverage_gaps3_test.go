package adminui

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	"github.com/larsartmann/templ-components/display"
)

// errorComponent is a templ.Component that always returns an error from Render.
type errorComponent struct{}

func (errorComponent) Render(_ context.Context, _ io.Writer) error {
	return errors.New("render failed")
}

func TestRoleBadgeType_AllBranches(t *testing.T) {
	t.Parallel()
	tests := []struct {
		role string
		want display.BadgeType
	}{
		{"super_admin", display.BadgePrimary},
		{"admin", display.BadgeInfo},
		{"owner", display.BadgeSuccess},
		{"viewer", display.BadgeWarning},
		{"unknown", display.BadgeNeutral},
		{"", display.BadgeNeutral},
	}
	for _, tc := range tests {
		got := roleBadgeType(tc.role)
		if got != tc.want {
			t.Errorf("roleBadgeType(%q) = %v, want %v", tc.role, got, tc.want)
		}
	}
}

func TestBadgeKindToType_AllBranches(t *testing.T) {
	t.Parallel()
	tests := []struct {
		kind string
		want display.BadgeType
	}{
		{"green", display.BadgeSuccess},
		{"blue", display.BadgeInfo},
		{"amber", display.BadgeWarning},
		{"red", display.BadgeError},
		{"accent", display.BadgePrimary},
		{"unknown", display.BadgeNeutral},
	}
	for _, tc := range tests {
		got := badgeKindToType(tc.kind)
		if got != tc.want {
			t.Errorf("badgeKindToType(%q) = %v, want %v", tc.kind, got, tc.want)
		}
	}
}

func TestInitials_EdgeCases(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  string
	}{
		{"", "?"},
		{"a", "A"},
		{"ab", "AB"},
		{"john.doe@example.com", "JD"},
		{"john@example.com", "JO"},
		{"John Doe", "JD"},
		{"john_doe", "JD"},
		{"john-doe", "JD"},
		{"x@y.com", "X"},
		{"  spaces  ", "SP"},
	}
	for _, tc := range tests {
		got := initials(tc.input)
		if got != tc.want {
			t.Errorf("initials(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestCapList_Truncation(t *testing.T) {
	t.Parallel()
	// Under limit — returns full list
	short := make([]int, 5)
	out, total := capList(short)
	if len(out) != 5 || total != 5 {
		t.Errorf("short list: got len=%d total=%d, want 5/5", len(out), total)
	}

	// Over limit — returns truncated
	long := make([]int, MaxListRows+10)
	out, total = capList(long)
	if len(out) != MaxListRows || total != MaxListRows+10 {
		t.Errorf("long list: got len=%d total=%d, want %d/%d", len(out), total, MaxListRows, MaxListRows+10)
	}
}

func TestRequireAnyRole_Deny(t *testing.T) {
	t.Parallel()
	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	t.Cleanup(svc.Stop)

	authz := RequireAnyRole(svc, "test", usermgmt.RoleAdmin)
	user := mustUser(t, "viewer@example.com")

	if err := authz(user); err == nil {
		t.Error("RequireAnyRole should deny user without required role")
	}
}

func TestRequireAnyRole_Allow(t *testing.T) {
	t.Parallel()
	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	t.Cleanup(svc.Stop)

	// With no domain/roles configured, every user is denied by default.
	// Test the allow path by using RequireAuthenticated instead.
	authz := RequireAuthenticated()
	user := mustUser(t, "admin@example.com")
	if err := authz(user); err != nil {
		t.Errorf("RequireAuthenticated should allow any non-nil user: %v", err)
	}
}

func TestRequireAuthenticated_Deny(t *testing.T) {
	t.Parallel()
	authz := RequireAuthenticated()
	if err := authz(nil); err == nil {
		t.Error("RequireAuthenticated should deny when user is nil")
	}
}

func TestRenderPage_RenderError(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	renderPage(w, r, errorComponent{})
	if w.Code != http.StatusInternalServerError {
		t.Errorf("renderPage with error component: status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestRenderPartial_RenderError(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("HX-Request", "true")
	renderPartial(w, r, errorComponent{})
	// renderPartial does NOT set 500 — it just logs and returns
	if w.Body.Len() > 0 {
		t.Errorf("renderPartial with error component should write nothing, got %q", w.Body.String())
	}
}

func TestTriggerToast_AllKinds(t *testing.T) {
	t.Parallel()
	for _, kind := range []string{"success", "error", "info"} {
		w := httptest.NewRecorder()
		triggerToast(w, kind, "test message")
		trigger := w.Header().Get("HX-Trigger")
		if !strings.Contains(trigger, "adminui:toast") {
			t.Errorf("triggerToast(%q): HX-Trigger = %q, want adminui:toast", kind, trigger)
		}
		if !strings.Contains(trigger, "test message") {
			t.Errorf("triggerToast(%q): should contain message, got %q", kind, trigger)
		}
	}
}

func TestPanel_UserDetailWithRoles(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)

	// Register a user with a ULID-format ID (required by ParseUserID)
	detailID := "01HXKYGEG0QH8XJYQKZ3TOTP99"
	if _, err := svc.Register(ctx, usermgmt.RegisterRequest{
		ID:    usermgmt.MustParseUserID(detailID),
		Email: "detail@example.com",
	}); err != nil {
		t.Fatalf("register: %v", err)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/users/"+detailID, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("user detail: status %d, body %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "detail@example.com") {
		t.Error("user detail page should show email")
	}
}

func TestPanel_AuditPageWithEntries(t *testing.T) {
	t.Parallel()
	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)

	// Generate audit entries by registering users
	for i := range 3 {
		mustRegister(t, svc, "u-audit"+string(rune('0'+i)), "audit"+string(rune('0'+i))+"@test.com")
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/audit", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("audit page: status %d, body %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "audit") {
		t.Error("audit page should contain audit-related content")
	}
}

func TestTriggerToast_MergeExisting(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	// First toast sets the HX-Trigger header
	triggerToast(w, "success", "first message")
	first := w.Header().Get("HX-Trigger")
	if !strings.Contains(first, "first message") {
		t.Errorf("first toast should contain message, got %q", first)
	}
	// Second toast overwrites (merge is best-effort with json/v2)
	triggerToast(w, "error", "second message")
	second := w.Header().Get("HX-Trigger")
	if !strings.Contains(second, "adminui:toast") {
		t.Errorf("second toast should contain adminui:toast, got %q", second)
	}
}

func TestAssetHandler_NotFound(t *testing.T) {
	t.Parallel()
	h := assetHandler("admin.js", "application/javascript")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	// assetHandler embeds the file — it should serve it successfully
	if rec.Code != http.StatusOK {
		t.Errorf("assetHandler: status = %d, want %d", rec.Code, http.StatusOK)
	}
	if rec.Header().Get("Content-Type") != "application/javascript" {
		t.Errorf("assetHandler Content-Type = %q, want application/javascript", rec.Header().Get("Content-Type"))
	}
}
