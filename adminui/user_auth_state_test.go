package adminui

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
)

// The stubs below satisfy the published identity-model provider interfaces
// with the smallest surface that makes the Service's capability probes
// (HasWebAuthn/HasTOTP/HasOAuth2) true. They never perform real ceremonies.

// stubWebAuthnProvider is a WebAuthnProvider in name only: the probes and
// read paths under test never invoke the ceremonies.
type stubWebAuthnProvider struct{}

func (stubWebAuthnProvider) BeginRegistration(context.Context, []byte) ([]byte, []byte, error) {
	return nil, nil, nil
}

func (stubWebAuthnProvider) FinishRegistration(context.Context, []byte, []byte, []byte) ([]byte, error) {
	return nil, nil
}

func (stubWebAuthnProvider) BeginLogin(context.Context, []byte) ([]byte, []byte, error) {
	return nil, nil, nil
}

func (stubWebAuthnProvider) FinishLogin(context.Context, []byte, []byte, []byte) ([]byte, error) {
	return nil, nil
}

// stubTOTPProvider accepts exactly one code, so a test can walk the real
// EnableTOTP → VerifyTOTPSetup path to a TOTP-enabled user.
type stubTOTPProvider struct{}

func (stubTOTPProvider) GenerateSecret(string) ([]byte, string, string, error) {
	return []byte{0x01}, "ABCD", "otpauth://totp/x", nil
}

func (stubTOTPProvider) ValidateCode([]byte, string) bool { return true }

// stubOAuth2Provider is an OAuth2Provider that can enumerate its names, the
// richer of the two provider shapes ConfiguredOAuth2Providers supports.
type stubOAuth2Provider struct{ names []string }

func (*stubOAuth2Provider) BeginLogin(context.Context, string, string) (string, string, error) {
	return "", "", nil
}

func (*stubOAuth2Provider) FinishLogin(context.Context, string, string, string) ([]byte, error) {
	return nil, nil
}

func (p *stubOAuth2Provider) Names() []string { return p.names }

// newAuthStatePanel wraps a caller-built Service (the provider-rich ones the
// default newTestPanel cannot construct) in the panel behind an injected
// session user.
func newAuthStatePanel(t *testing.T, svc *usermgmt.Service, user *identitymodel.User) http.Handler {
	t.Helper()
	panel, err := New(Config{Service: svc, Authorizer: RequireAuthenticated()})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return http.StripPrefix("/admin", func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(usermgmt.WithUser(r.Context(), user)))
		})
	}(panel.Handler()))
}

func newAuthStateService(t *testing.T) *usermgmt.Service {
	t.Helper()
	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{
		AuditLog: usermgmt.NewAuditLog(),
		WebAuthn: stubWebAuthnProvider{},
		TOTP:     stubTOTPProvider{},
		OAuth2:   &stubOAuth2Provider{names: []string{"github", "google"}},
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	return svc
}

// TestAuthMethods_Probes pins the capability builder to the Service's
// published probes: a bare service reports every channel false with nil
// providers; a provider-rich service reports them true with the names the
// OAuth2 provider enumerates.
func TestAuthMethods_Probes(t *testing.T) {
	bare, err := usermgmt.NewService(usermgmt.ServiceConfig{AuditLog: usermgmt.NewAuditLog()})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	got := authMethodsFromService(bare)
	if got.HasWebAuthn || got.HasTOTP || got.HasOAuth2 {
		t.Errorf("bare service reported capabilities: %+v", got)
	}
	if got.Providers != nil {
		t.Errorf("bare service reported providers %v, want nil", got.Providers)
	}

	full := newAuthStateService(t)
	got = authMethodsFromService(full)
	if !got.HasWebAuthn || !got.HasTOTP || !got.HasOAuth2 {
		t.Errorf("provider-rich service missed capabilities: %+v", got)
	}
	if len(got.Providers) != 2 || got.Providers[0] != "github" || got.Providers[1] != "google" {
		t.Errorf("providers = %v, want [github google]", got.Providers)
	}
}

// TestCannotSignInTruthTable is the lockout truth table: a user locks out of
// their own account only when they hold NO sign-in channel — no WebAuthn
// credential, no linked external account, no TOTP. Each single channel is
// enough to keep them able to sign in.
func TestCannotSignInTruthTable(t *testing.T) {
	cases := []struct {
		name string
		seed func(u *identitymodel.User)
		want bool
	}{
		{"credential-only", func(u *identitymodel.User) {
			u.Credentials = []identitymodel.WebAuthnCredential{{ID: []byte{0x01}, Name: "key"}}
		}, false},
		{"external-only", func(u *identitymodel.User) {
			u.ExternalAccounts = []identitymodel.ExternalAccount{
				identitymodel.NewExternalAccount("google", "sub", "g@x.com", "G", time.Now()),
			}
		}, false},
		{"TOTP-only", func(u *identitymodel.User) { u.TOTPEnabled = true }, false},
		{"none", func(u *identitymodel.User) {}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			u := mustUser(t, "lockout@example.com")
			tc.seed(u)
			if got := cannotSignIn(u); got != tc.want {
				t.Errorf("cannotSignIn(%s) = %v, want %v", tc.name, got, tc.want)
			}
		})
	}
}

// TestAuthState_UserDetailShowsCards flips the dead-card gate: with all
// three providers configured the passkeys card, external-accounts card, and
// TOTP details row render — and the lockout badge still fires, because a
// freshly registered user holds no channel yet (capability ≠ credential).
func TestAuthState_UserDetailShowsCards(t *testing.T) {
	ctx := context.Background()
	svc := newAuthStateService(t)
	admin := mustUser(t, "admin@example.com")
	h := newAuthStatePanel(t, svc, admin)

	target, err := svc.Register(ctx, usermgmt.RegisterRequest{
		ID: identitymodel.SyntheticUserID("u-probes"), Email: "probes@example.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	body := renderUserDetail(t, h, target.User.ID.Get().String())

	for _, want := range []string{
		"Passkeys", "No passkeys enrolled", "Credentials",
		"External accounts", "No external accounts linked.", "github", "google",
		"TOTP MFA",
		"Cannot sign in",
	} {
		if !contains(body, want) {
			t.Errorf("provider-rich user detail missing %q", want)
		}
	}
}

// TestAuthState_LockoutBadgeRendersTruthTable renders the user-detail
// content once per truth-table row: the badge appears exactly when the user
// holds no sign-in channel, with the escalation copy the last-passkey
// confirm uses. The external-only row is built directly because the public
// Service API links external accounts only inside the OAuth2 login flow.
func TestAuthState_LockoutBadgeRendersTruthTable(t *testing.T) {
	cases := []struct {
		name      string
		seed      func(u *identitymodel.User)
		wantBadge bool
	}{
		{"credential-only", func(u *identitymodel.User) {
			u.Credentials = []identitymodel.WebAuthnCredential{{ID: []byte{0x01}, Name: "key"}}
		}, false},
		{"external-only", func(u *identitymodel.User) {
			u.ExternalAccounts = []identitymodel.ExternalAccount{
				identitymodel.NewExternalAccount("google", "sub", "g@x.com", "G", time.Now()),
			}
		}, false},
		{"TOTP-only", func(u *identitymodel.User) { u.TOTPEnabled = true }, false},
		{"none", func(u *identitymodel.User) {}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := userDetailData{
				User:        mustUser(t, "badge@example.com"),
				BasePath:    "/admin",
				AuthMethods: authMethods{HasWebAuthn: true, HasTOTP: true, HasOAuth2: true},
			}
			tc.seed(d.User)
			body := renderComponent(t, userDetailContent(pageData{}, d))
			got := contains(body, "Cannot sign in")
			if got != tc.wantBadge {
				t.Errorf("%s: badge rendered = %v, want %v", tc.name, got, tc.wantBadge)
			}
			if tc.wantBadge && !contains(body, "locked out of their account") {
				t.Errorf("%s: badge missing escalation copy", tc.name)
			}
		})
	}
}

// TestAuthState_BadgeClearsAfterCredential walks the real handler: adding a
// passkey through the published AddCredential path clears the lockout badge
// on the next render.
func TestAuthState_BadgeClearsAfterCredential(t *testing.T) {
	ctx := context.Background()
	svc := newAuthStateService(t)
	admin := mustUser(t, "admin@example.com")
	h := newAuthStatePanel(t, svc, admin)

	target, err := svc.Register(ctx, usermgmt.RegisterRequest{
		ID: identitymodel.SyntheticUserID("u-rescue"), Email: "rescue@example.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	id := target.User.ID.Get().String()
	if body := renderUserDetail(t, h, id); !contains(body, "Cannot sign in") {
		t.Fatal("fresh user should render the lockout badge")
	}
	if err := svc.AddCredential(ctx, target.User.ID, identitymodel.WebAuthnCredential{
		ID: []byte{0x01, 0x02}, Name: "YubiKey", CreatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("AddCredential: %v", err)
	}
	if body := renderUserDetail(t, h, id); contains(body, "Cannot sign in") {
		t.Error("lockout badge still rendered after a credential was added")
	}
}

// contains is a shorthand so the assertion loops above stay one-line.
func contains(haystack, needle string) bool { return strings.Contains(haystack, needle) }
