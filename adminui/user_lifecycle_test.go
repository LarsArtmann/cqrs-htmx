package adminui

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
)

func TestFlow_UserCreate(t *testing.T) {
	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)

	rec := postForm(h, "/admin/users", url.Values{
		"email": {"ada@example.com"}, "display_name": {"Ada Lovelace"},
	})
	if loc := rec.Header().Get("HX-Redirect"); !strings.HasPrefix(loc, "/admin/users/") {
		t.Fatalf("create HX-Redirect = %q, want /admin/users/{id}", loc)
	}
	created, ok := svc.ReadModel().FindByEmail("ada@example.com")
	if !ok {
		t.Fatal("user not created: FindByEmail missed ada@example.com")
	}
	if created.DisplayName != "Ada Lovelace" {
		t.Errorf("display name = %q", created.DisplayName)
	}
}

func TestFlow_UserCreateRejectsEmptyEmail(t *testing.T) {
	user := mustUser(t, "admin@example.com")
	h, _ := newTestPanel(t, user)

	rec := postForm(h, "/admin/users", url.Values{"display_name": {"No Email"}})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if !strings.Contains(rec.Header().Get("HX-Trigger"), "adminui:toast") {
		t.Error("expected error toast")
	}
}

func TestFlow_UserCreateRejectsDuplicateEmail(t *testing.T) {
	user := mustUser(t, "admin@example.com")
	h, _ := newTestPanel(t, user)

	postForm(h, "/admin/users", url.Values{"email": {"dupe@example.com"}})
	rec := postForm(h, "/admin/users", url.Values{"email": {"dupe@example.com"}})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("duplicate email status = %d, want 400 (rejection)", rec.Code)
	}
}

// TestFlow_UserProfileEdits covers both inline profile forms on the user
// detail page: display name and sign-in email. Each case registers a fresh
// user, posts the edit, and asserts the redirect lands back on the detail
// page with the field actually changed on the read side.
func TestFlow_UserProfileEdits(t *testing.T) {
	cases := []struct {
		name        string
		seedID      string
		seedEmail   string
		action      string
		formField   string
		formValue   string
		wantChanged func(u *identitymodel.User) bool
	}{
		{
			name:      "display name",
			seedID:    "u-rename",
			seedEmail: "rename@example.com",
			action:    "display-name",
			formField: "display_name",
			formValue: "Renamed",
			wantChanged: func(u *identitymodel.User) bool {
				return u.DisplayName == "Renamed"
			},
		},
		{
			name:      "email",
			seedID:    "u-reemail",
			seedEmail: "old@example.com",
			action:    "email",
			formField: "email",
			formValue: "new@example.com",
			wantChanged: func(u *identitymodel.User) bool {
				return u.Email == "new@example.com"
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			user := mustUser(t, "admin@example.com")
			h, svc := newTestPanel(t, user)

			target, err := svc.Register(ctx, usermgmt.RegisterRequest{
				ID: identitymodel.SyntheticUserID(tc.seedID), Email: tc.seedEmail,
			})
			if err != nil {
				t.Fatal(err)
			}
			id := target.User.ID.Get().String()

			rec := postForm(h, "/admin/users/"+id+"/"+tc.action, url.Values{tc.formField: {tc.formValue}})
			if rec.Header().Get("HX-Redirect") != "/admin/users/"+id {
				t.Fatalf("HX-Redirect = %q", rec.Header().Get("HX-Redirect"))
			}
			shown, err := svc.GetUser(ctx, target.User.ID)
			if err != nil || !tc.wantChanged(shown) {
				t.Fatalf("%s not updated: %v %+v", tc.name, err, shown)
			}
		})
	}
}

func TestFlow_PlatformRolesGrantAndRevoke(t *testing.T) {
	ctx := context.Background()
	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)

	target, err := svc.Register(ctx, usermgmt.RegisterRequest{
		ID: identitymodel.SyntheticUserID("u-grant"), Email: "grant@example.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	id := target.User.ID.Get().String()
	wildcard := identitymodel.NewTenantID(identitymodel.WildcardDomain)

	rec := postForm(h, "/admin/users/"+id+"/roles", url.Values{"roles": {"admin"}})
	if rec.Header().Get("HX-Redirect") != "/admin/users/"+id {
		t.Fatalf("grant HX-Redirect = %q", rec.Header().Get("HX-Redirect"))
	}
	roles, err := svc.Authz().RolesForUser(target.User.ID, wildcard)
	if err != nil || len(roles) != 1 || roles[0] != identitymodel.RoleAdmin {
		t.Fatalf("platform roles after grant = %v (err %v)", roles, err)
	}

	rec = postForm(h, "/admin/users/"+id+"/roles/revoke", nil)
	if rec.Header().Get("HX-Redirect") != "/admin/users/"+id {
		t.Fatalf("revoke HX-Redirect = %q", rec.Header().Get("HX-Redirect"))
	}
	roles, err = svc.Authz().RolesForUser(target.User.ID, wildcard)
	if err != nil || len(roles) != 0 {
		t.Fatalf("platform roles after revoke = %v (err %v)", roles, err)
	}
}

func TestFlow_PlatformRolesGrantRequiresOneRole(t *testing.T) {
	ctx := context.Background()
	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)

	target, err := svc.Register(ctx, usermgmt.RegisterRequest{
		ID: identitymodel.SyntheticUserID("u-empty"), Email: "empty@example.com",
	})
	if err != nil {
		t.Fatal(err)
	}

	rec := postForm(h, "/admin/users/"+target.User.ID.Get().String()+"/roles", url.Values{})
	if rec.Code == http.StatusOK || rec.Header().Get("HX-Redirect") != "" {
		t.Fatalf("empty grant should fail loudly, got %d / %q", rec.Code, rec.Header().Get("HX-Redirect"))
	}
	if !strings.Contains(rec.Header().Get("HX-Trigger"), "adminui:toast") {
		t.Error("expected error toast")
	}
}

func TestFlow_RemoveCredential(t *testing.T) {
	ctx := context.Background()
	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)

	target, err := svc.Register(ctx, usermgmt.RegisterRequest{
		ID: identitymodel.SyntheticUserID("u-passkey"), Email: "passkey@example.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.AddCredential(ctx, target.User.ID, identitymodel.WebAuthnCredential{
		ID: []byte{0x01, 0x02}, Name: "YubiKey",
		CreatedAt: time.Now(),
	}); err != nil {
		t.Fatal(err)
	}
	id := target.User.ID.Get().String()
	enc := base64.RawURLEncoding.EncodeToString([]byte{0x01, 0x02})

	rec := postForm(h, "/admin/users/"+id+"/credentials/"+enc+"/delete", nil)
	if rec.Header().Get("HX-Redirect") != "/admin/users/"+id {
		t.Fatalf("HX-Redirect = %q", rec.Header().Get("HX-Redirect"))
	}
	shown, err := svc.GetUser(ctx, target.User.ID)
	if err != nil || len(shown.Credentials) != 0 {
		t.Fatalf("credential not removed: %v %+v", err, shown.Credentials)
	}
}

func TestFlow_RemoveCredentialRejectsBadEncoding(t *testing.T) {
	ctx := context.Background()
	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)

	target, err := svc.Register(ctx, usermgmt.RegisterRequest{
		ID: identitymodel.SyntheticUserID("u-badb64"), Email: "badb64@example.com",
	})
	if err != nil {
		t.Fatal(err)
	}

	rec := postForm(h, "/admin/users/"+target.User.ID.Get().String()+"/credentials/not*base64/delete", nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestFlow_ResendVerificationNotConfigured(t *testing.T) {
	ctx := context.Background()
	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)

	target, err := svc.Register(ctx, usermgmt.RegisterRequest{
		ID: identitymodel.SyntheticUserID("u-verify"), Email: "verify@example.com",
	})
	if err != nil {
		t.Fatal(err)
	}

	rec := postForm(h, "/admin/users/"+target.User.ID.Get().String()+"/resend-verification", nil)
	// No email-verification configured on the Service: the action must fail
	// with a visible explanation, not a silent success or a bare 500.
	if rec.Header().Get("HX-Redirect") != "" {
		t.Fatalf("HX-Redirect = %q, want none (action must fail)", rec.Header().Get("HX-Redirect"))
	}
	if !strings.Contains(rec.Header().Get("HX-Trigger"), "adminui:toast") {
		t.Error("expected error toast")
	}
}

// renderUserDetail fetches the user-detail page HTML for assertion.
func renderUserDetail(t *testing.T, h http.Handler, id string) string {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/users/"+id, nil)
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("detail status = %d: %s", rec.Code, rec.Body.String())
	}
	return rec.Body.String()
}

func TestRender_UserNewPage(t *testing.T) {
	user := mustUser(t, "admin@example.com")
	h, _ := newTestPanel(t, user)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/users/new", nil)
	h.ServeHTTP(rec, req)
	body := rec.Body.String()
	for _, want := range []string{
		"New user", `hx-post="/admin/users"`, "email", "display_name",
		"cannot set credentials", "Create user",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("new-user page missing %q", want)
		}
	}
}

func TestRender_UserDetailLifecycleCards(t *testing.T) {
	ctx := context.Background()
	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)

	target, err := svc.Register(ctx, usermgmt.RegisterRequest{
		ID: identitymodel.SyntheticUserID("u-cards"), Email: "cards@example.com", DisplayName: "Cards",
	})
	if err != nil {
		t.Fatal(err)
	}
	body := renderUserDetail(t, h, target.User.ID.Get().String())

	for _, want := range []string{
		"Edit profile",
		`hx-post="/admin/users/` + target.User.ID.Get().String() + `/display-name"`,
		`hx-post="/admin/users/` + target.User.ID.Get().String() + `/email"`,
		"Passkeys", "No passkeys enrolled",
		"Platform access", `name="roles"`, "value=\"super_admin\"", "value=\"admin\"",
		"Resend verification",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("user detail missing %q", want)
		}
	}

	// The self-lockout warning must NOT appear for another user.
	if strings.Contains(body, "locks you out") {
		t.Error("self-lockout warning shown for a different user")
	}
}

func TestRender_UserDetailSelfLockoutWarning(t *testing.T) {
	ctx := context.Background()

	// The session user must BE the shown user and exist in the read model,
	// so register first and build the panel around that identity.
	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{AuditLog: usermgmt.NewAuditLog()})
	if err != nil {
		t.Fatal(err)
	}
	self, err := svc.Register(ctx, usermgmt.RegisterRequest{
		ID: identitymodel.SyntheticUserID("u-self"), Email: "self@example.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	panel, err := New(Config{Service: svc, Authorizer: RequireAuthenticated()})
	if err != nil {
		t.Fatal(err)
	}
	h := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(usermgmt.WithUser(r.Context(), self.User)))
		})
	}(http.StripPrefix("/admin", panel.Handler()))

	body := renderUserDetail(t, h, self.User.ID.Get().String())
	if !strings.Contains(body, "This is your own account") {
		t.Error("self-lockout warning missing on own detail page")
	}
}

func TestRender_PlatformRolesChecked(t *testing.T) {
	ctx := context.Background()
	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)

	target, err := svc.Register(ctx, usermgmt.RegisterRequest{
		ID: identitymodel.SyntheticUserID("u-checked"), Email: "checked@example.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.GrantRoles(ctx,
		identitymodel.ActorIDFromUser(target.User.ID),
		identitymodel.NewTenantID(identitymodel.WildcardDomain),
		[]identitymodel.Role{identitymodel.RoleSuperAdmin},
	); err != nil {
		t.Fatal(err)
	}

	body := renderUserDetail(t, h, target.User.ID.Get().String())
	if !strings.Contains(body, "checked") {
		t.Error("granted super_admin checkbox not pre-checked")
	}
	if !strings.Contains(body, "Revoke all") {
		t.Error("revoke-all action missing when platform roles exist")
	}
}
