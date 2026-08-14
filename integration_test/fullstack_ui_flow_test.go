package integration_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/larsartmann/cqrs-htmx/loginpage/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	"github.com/stretchr/testify/require"
)

// authDeadline bounds polling loops below. Generous on purpose: projections
// apply asynchronously and CI runners can be slow.
const (
	authDeadline = 10 * time.Second
	testProvider = "google"
)

// TestFullstackUI_AdminRendersSeededUser registers a real user through
// Service.Register, grants super_admin through the real authz path, and
// asserts the admin panel's user list renders the seeded user's data.
func TestFullstackUI_AdminRendersSeededUser(t *testing.T) {
	t.Parallel()
	handler, svc := setupFullstackUI(t)

	const email = "seeded-user@example.com"

	resp, err := svc.Register(t.Context(), usermgmt.RegisterRequest{
		Email:       email,
		DisplayName: "Seeded User",
	})
	require.NoError(t, err)
	require.NotNil(t, resp.Session)

	require.NoError(t, svc.Authz().AddGroupPolicy(usermgmt.GroupPolicy{
		Subject: resp.User.ID.Get().String(),
		Role:    usermgmt.RoleSuperAdmin,
		Domain:  "*",
	}))

	// The session is valid immediately; the role policy may still be applying
	// in the Casbin projection, so poll until the panel admits the user.
	var body string

	deadline := time.Now().Add(authDeadline)

	for {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/admin/users", nil)
		r.AddCookie(&http.Cookie{Name: "session_token", Value: resp.Session.Token}) //nolint:gosec // test cookie
		handler.ServeHTTP(w, r)

		body = w.Body.String()

		if w.Code == http.StatusOK || time.Now().After(deadline) {
			require.Equal(t, http.StatusOK, w.Code, "body: %s", body)

			break
		}

		require.Equal(t, http.StatusUnauthorized, w.Code,
			"unexpected status while waiting for role policy, body: %s", body)

		time.Sleep(25 * time.Millisecond)
	}

	require.Contains(t, body, email, "seeded user should appear in the admin user list")
	require.Contains(t, body, "Seeded User")
}

// TestFullstackUI_DashboardShowsProjectionHealth asserts the dashboard's
// projections page renders live projection names from the Service's
// projection host (the health data, not just the page shell).
func TestFullstackUI_DashboardShowsProjectionHealth(t *testing.T) {
	t.Parallel()
	handler, _ := setupFullstackUI(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/dashboard/projections", nil)
	handler.ServeHTTP(w, r)

	body := w.Body.String()
	require.Equal(t, http.StatusOK, w.Code, "body: %s", body)
	require.Contains(t, body, "user-read-model", "projection names should render")
	require.Contains(t, body, "casbin-projection")
	require.Contains(t, body, "tenant-read-model")
}

// stubWebAuthnProvider satisfies identitymodel.WebAuthnProvider (via the
// usermgmt alias) with zero-value ceremonies. Only HasWebAuthn() matters here.
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

func (stubWebAuthnProvider) FinishLogin(context.Context, []byte, []byte, []byte) error {
	return nil
}

// stubOAuth2Provider satisfies identitymodel.OAuth2Provider plus the optional
// Names() enumeration loginpage uses to auto-generate sign-in buttons.
type stubOAuth2Provider struct{}

func (stubOAuth2Provider) BeginLogin(context.Context, string, string) (string, string, error) {
	return "", "", nil
}

func (stubOAuth2Provider) FinishLogin(context.Context, string, string, string) ([]byte, error) {
	return nil, nil
}

func (stubOAuth2Provider) Names() []string { return []string{testProvider} }

// loginPageFor builds a Service with the given auth provider and returns a
// handler serving only the login page at /.
func loginPageFor(t *testing.T, cfg usermgmt.ServiceConfig) http.Handler {
	t.Helper()

	svc, err := usermgmt.NewService(cfg)
	require.NoError(t, err)
	t.Cleanup(svc.Stop)

	login, err := loginpage.New(loginpage.Config{Service: svc, Title: "Auth Test"})
	require.NoError(t, err)

	mux := http.NewServeMux()
	login.Mount(mux, "/")

	return mux
}

// TestFullstackUI_LoginButtonsMatchAuthConfig asserts the login page renders
// exactly the auth options the Service is configured with: the passkey form
// appears only with a WebAuthn provider, OAuth2 buttons only with providers
// that enumerate names, and a bare Service (no providers) shows the setup
// hint instead. Note: TOTP deliberately has no login-page UI — it is a
// second-factor API flow (/auth/totp/*), so it is not asserted here.
func TestFullstackUI_LoginButtonsMatchAuthConfig(t *testing.T) {
	t.Parallel()

	getBody := func(t *testing.T, handler http.Handler) string {
		t.Helper()

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

		require.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())

		return w.Body.String()
	}

	t.Run("NoProviders", func(t *testing.T) {
		t.Parallel()
		body := getBody(t, loginPageFor(t, usermgmt.ServiceConfig{}))

		require.NotContains(t, body, "Sign in with passkey")
		require.NotContains(t, body, "Sign in with "+loginpage.ProviderDisplayName(testProvider))
		require.Contains(t, body, "Set up WebAuthn or OAuth2", "setup hint should render")
	})

	t.Run("WebAuthnOnly", func(t *testing.T) {
		t.Parallel()
		body := getBody(t, loginPageFor(t, usermgmt.ServiceConfig{
			WebAuthn: stubWebAuthnProvider{},
		}))

		require.Contains(t, body, "Sign in with passkey")
		require.NotContains(t, body, "Sign in with "+loginpage.ProviderDisplayName(testProvider))
	})

	t.Run("OAuth2Only", func(t *testing.T) {
		t.Parallel()
		body := getBody(t, loginPageFor(t, usermgmt.ServiceConfig{
			OAuth2: stubOAuth2Provider{},
		}))

		require.Contains(t, body, "Sign in with "+loginpage.ProviderDisplayName(testProvider))
		require.NotContains(t, body, "Sign in with passkey")
	})
}
