package usermgmt

import (
	"context"
	"encoding/base32"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
)

func setupAuthenticatedHandler(t *testing.T, cfg ServiceConfig) (*Service, http.Handler, string) {
	t.Helper()
	svc, err := NewService(cfg)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	t.Cleanup(svc.Stop)

	reg, err := svc.Register(context.Background(), RegisterRequest{
		ID: NewUserID("authu1"), Email: "authu1@test.com",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	// Grant admin role so import/export endpoints work by default.
	grantTestRole(t, svc, reg.User.ID, RoleAdmin)

	// UpdateRoles revokes sessions — create a fresh one.
	sess, err := svc.sessions.Create(context.Background(), reg.User.ID, defaultSessionTTL)
	if err != nil {
		t.Fatalf("Create session: %v", err)
	}

	h := NewAuthHandler(svc, HandlerConfig{Secure: new(bool)})
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	wrapped := NewSessionMiddleware(svc, "session_token")(mux)
	return svc, wrapped, sess.Token
}

func authenticatedRequest(t *testing.T, h http.Handler, method, path, token, body string) *httptest.ResponseRecorder {
	t.Helper()
	var bodyReader io.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, bodyReader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}

func TestHandlers_SendVerificationEmail(t *testing.T) {
	svc, h, token := setupAuthenticatedHandler(t, ServiceConfig{
		EmailVerification: &EmailVerificationConfig{},
	})
	w := authenticatedRequest(t, h, http.MethodPost, "/auth/email/verify/send", token, "")
	assertStatusCode(t, w, http.StatusOK)

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["token"] == "" {
		t.Error("expected non-empty token")
	}

	// Verify email via service
	if err := svc.VerifyEmail(context.Background(), resp["token"]); err != nil {
		t.Fatalf("VerifyEmail: %v", err)
	}
}

func TestHandlers_VerifyEmail(t *testing.T) {
	svc, h, token := setupAuthenticatedHandler(t, ServiceConfig{
		EmailVerification: &EmailVerificationConfig{},
	})
	tok, err := svc.SendVerificationEmail(context.Background(), NewUserID("authu1"))
	if err != nil {
		t.Fatalf("SendVerificationEmail: %v", err)
	}

	w := authenticatedRequest(t, h, http.MethodPost, "/auth/email/verify", token,
		`{"token":"`+tok+`"}`)
	assertStatusCode(t, w, http.StatusOK)
}

func TestHandlers_VerifyEmail_InvalidToken(t *testing.T) {
	_, h, _ := setupAuthenticatedHandler(t, ServiceConfig{
		EmailVerification: &EmailVerificationConfig{},
	})
	w := authenticatedRequest(t, h, http.MethodPost, "/auth/email/verify", "",
		`{"token":"any"}`)
	assertStatusCode(t, w, http.StatusBadRequest)
}

func TestHandlers_TOTPSetup(t *testing.T) {
	_, h, token := setupAuthenticatedHandler(t, ServiceConfig{
		TOTPConfig: &TOTPConfig{Issuer: "Test", Window: 1},
	})
	w := authenticatedRequest(t, h, http.MethodPost, "/auth/totp/setup", token, "")
	assertStatusCode(t, w, http.StatusOK)

	var resp TOTPSetupResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Secret == "" || resp.QRCodeURI == "" {
		t.Error("expected secret and qr_code_uri")
	}
}

func TestHandlers_TOTPSetup_Unauthenticated(t *testing.T) {
	_, h, _ := setupAuthenticatedHandler(t, ServiceConfig{
		TOTPConfig: &TOTPConfig{},
	})
	w := authenticatedRequest(t, h, http.MethodPost, "/auth/totp/setup", "", "")
	assertStatusCode(t, w, http.StatusUnauthorized)
}

// enableTOTPForUser enables TOTP and confirms setup with a current code.
// Returns the current valid TOTP code for subsequent verification calls.
func enableTOTPForUser(t *testing.T, svc *Service) string {
	t.Helper()
	setup, err := svc.EnableTOTP(context.Background(), NewUserID("authu1"))
	if err != nil {
		t.Fatalf("EnableTOTP: %v", err)
	}
	code := currentTOTPCode(t, decodeSecret(t, setup.Secret))
	if err := svc.VerifyTOTPSetup(context.Background(), NewUserID("authu1"), code); err != nil {
		t.Fatalf("VerifyTOTPSetup: %v", err)
	}
	return currentTOTPCode(t, decodeSecret(t, setup.Secret))
}

func TestHandlers_TOTPCodeVerifyAndDisable(t *testing.T) {
	cases := []struct {
		name string
		path string
	}{
		{"verify", "/auth/totp/verify"},
		{"disable", "/auth/totp/disable"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc, h, token := setupAuthenticatedHandler(t, ServiceConfig{
				TOTPConfig: &TOTPConfig{Issuer: "Test", Window: 1},
			})
			code := enableTOTPForUser(t, svc)
			w := authenticatedRequest(t, h, http.MethodPost, tc.path, token,
				`{"code":"`+code+`"}`)
			assertStatusCode(t, w, http.StatusOK)
		})
	}
}

func TestHandlers_ExportUsers_JSON(t *testing.T) {
	_, h, token := setupAuthenticatedHandler(t, ServiceConfig{})
	w := authenticatedRequest(t, h, http.MethodGet, "/auth/export?format=json", token, "")
	assertStatusCode(t, w, http.StatusOK)
	ct := w.Header().Get("Content-Type")
	if ct != contentTypeJSON {
		t.Errorf("Content-Type = %q, want %q", ct, contentTypeJSON)
	}
}

func TestHandlers_ExportUsers_CSV(t *testing.T) {
	_, h, token := setupAuthenticatedHandler(t, ServiceConfig{})
	w := authenticatedRequest(t, h, http.MethodGet, "/auth/export?format=csv", token, "")
	assertStatusCode(t, w, http.StatusOK)
	ct := w.Header().Get("Content-Type")
	if ct != "text/csv; charset=utf-8" {
		t.Errorf("Content-Type = %q, want text/csv", ct)
	}
}

func TestHandlers_ExportUsers_Unauthenticated(t *testing.T) {
	_, h, _ := setupAuthenticatedHandler(t, ServiceConfig{})
	w := authenticatedRequest(t, h, http.MethodGet, "/auth/export", "", "")
	assertStatusCode(t, w, http.StatusUnauthorized)
}

func TestHandlers_ImportUsers(t *testing.T) {
	cases := []struct {
		name string
		path string
		body string
	}{
		{"json", "/auth/import?format=json", `[{"email":"imported1@test.com","display_name":"Imported One"}]`},
		{"csv", "/auth/import?format=csv", "email,display_name\ncsvimport@test.com,CSV User\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, h, token := setupAuthenticatedHandler(t, ServiceConfig{})
			w := authenticatedRequest(t, h, http.MethodPost, tc.path, token, tc.body)
			assertStatusCode(t, w, http.StatusOK)

			var result ImportResult
			if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if result.Imported != 1 {
				t.Errorf("imported = %d, want 1", result.Imported)
			}
		})
	}
}

func TestHandlers_ImportUsers_Unauthenticated(t *testing.T) {
	_, h, _ := setupAuthenticatedHandler(t, ServiceConfig{})
	w := authenticatedRequest(t, h, http.MethodPost, "/auth/import", "", "[]")
	assertStatusCode(t, w, http.StatusUnauthorized)
}

func TestHandlers_ExportUsers_ForbiddenForNonAdmin(t *testing.T) {
	svc, err := NewService(ServiceConfig{})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	t.Cleanup(svc.Stop)

	reg, err := svc.Register(context.Background(), RegisterRequest{
		ID: NewUserID("nonadmin1"), Email: "nonadmin1@test.com",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	// User has RoleViewer + RoleUser only (not admin).

	h := NewAuthHandler(svc, HandlerConfig{Secure: new(bool)})
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	wrapped := NewSessionMiddleware(svc, "session_token")(mux)

	w := authenticatedRequest(t, wrapped, http.MethodGet, "/auth/export?format=json",
		reg.Session.Token, "")
	assertStatusCode(t, w, http.StatusForbidden)
}

func TestHandlers_ImportUsers_ForbiddenForNonAdmin(t *testing.T) {
	svc, err := NewService(ServiceConfig{})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	t.Cleanup(svc.Stop)

	reg, err := svc.Register(context.Background(), RegisterRequest{
		ID: NewUserID("nonadmin2"), Email: "nonadmin2@test.com",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	h := NewAuthHandler(svc, HandlerConfig{Secure: new(bool)})
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	wrapped := NewSessionMiddleware(svc, "session_token")(mux)

	w := authenticatedRequest(t, wrapped, http.MethodPost, "/auth/import?format=json",
		reg.Session.Token, `[{"email":"x@test.com"}]`)
	assertStatusCode(t, w, http.StatusForbidden)
}

func TestHandlers_ExportUsers_CustomAuthorizer(t *testing.T) {
	svc, err := NewService(ServiceConfig{})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	t.Cleanup(svc.Stop)

	reg, err := svc.Register(context.Background(), RegisterRequest{
		ID: NewUserID("custom1"), Email: "custom1@test.com",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	// Custom authorizer that always allows.
	h := NewAuthHandler(svc, HandlerConfig{
		Secure:                 new(bool),
		ImportExportAuthorizer: func(_ *User) error { return nil },
	})
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	wrapped := NewSessionMiddleware(svc, "session_token")(mux)

	w := authenticatedRequest(t, wrapped, http.MethodGet, "/auth/export?format=json",
		reg.Session.Token, "")
	assertStatusCode(t, w, http.StatusOK)
}

func TestHandlers_ImportRateLimit(t *testing.T) {
	svc, err := NewService(ServiceConfig{})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	t.Cleanup(svc.Stop)

	reg, err := svc.Register(context.Background(), RegisterRequest{
		ID: NewUserID("rl1"), Email: "rl1@test.com",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	grantTestRole(t, svc, reg.User.ID, RoleAdmin)
	sess, err := svc.sessions.Create(context.Background(), reg.User.ID, defaultSessionTTL)
	if err != nil {
		t.Fatalf("Create session: %v", err)
	}

	h := NewAuthHandler(svc, HandlerConfig{
		Secure:          new(bool),
		ImportRateLimit: RateLimitConfig{Enabled: true, MaxRequests: 1, Window: time.Minute},
	})
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	wrapped := NewSessionMiddleware(svc, "session_token")(mux)

	body := `[{"email":"imp1@test.com"}]`
	w1 := authenticatedRequest(t, wrapped, http.MethodPost, "/auth/import?format=json", sess.Token, body)
	assertStatusCode(t, w1, http.StatusOK)

	w2 := authenticatedRequest(t, wrapped, http.MethodPost, "/auth/import?format=json", sess.Token, body)
	assertStatusCode(t, w2, http.StatusTooManyRequests)
}

func TestHandlers_TOTPRateLimit(t *testing.T) {
	svc, err := NewService(ServiceConfig{
		TOTPConfig: &TOTPConfig{Issuer: "Test", Window: 1},
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	t.Cleanup(svc.Stop)

	reg, err := svc.Register(context.Background(), RegisterRequest{
		ID: NewUserID("rl2"), Email: "rl2@test.com",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	sess, err := svc.sessions.Create(context.Background(), reg.User.ID, defaultSessionTTL)
	if err != nil {
		t.Fatalf("Create session: %v", err)
	}

	h := NewAuthHandler(svc, HandlerConfig{
		Secure:        new(bool),
		TOTPRateLimit: RateLimitConfig{Enabled: true, MaxRequests: 1, Window: time.Minute},
	})
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	wrapped := NewSessionMiddleware(svc, "session_token")(mux)

	w1 := authenticatedRequest(t, wrapped, http.MethodPost, "/auth/totp/setup", sess.Token, "")
	assertStatusCode(t, w1, http.StatusOK)

	w2 := authenticatedRequest(t, wrapped, http.MethodPost, "/auth/totp/setup", sess.Token, "")
	assertStatusCode(t, w2, http.StatusTooManyRequests)
}

func TestHandlers_VerificationRateLimit(t *testing.T) {
	svc, err := NewService(ServiceConfig{
		EmailVerification: &EmailVerificationConfig{},
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	t.Cleanup(svc.Stop)

	reg, err := svc.Register(context.Background(), RegisterRequest{
		ID: NewUserID("rl3"), Email: "rl3@test.com",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	sess, err := svc.sessions.Create(context.Background(), reg.User.ID, defaultSessionTTL)
	if err != nil {
		t.Fatalf("Create session: %v", err)
	}

	h := NewAuthHandler(svc, HandlerConfig{
		Secure:                new(bool),
		VerificationRateLimit: RateLimitConfig{Enabled: true, MaxRequests: 1, Window: time.Minute},
	})
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	wrapped := NewSessionMiddleware(svc, "session_token")(mux)

	w1 := authenticatedRequest(t, wrapped, http.MethodPost, "/auth/email/verify/send", sess.Token, "")
	assertStatusCode(t, w1, http.StatusOK)

	w2 := authenticatedRequest(t, wrapped, http.MethodPost, "/auth/email/verify/send", sess.Token, "")
	assertStatusCode(t, w2, http.StatusTooManyRequests)
}

func TestHandlers_TOTPVerify_InvalidCode(t *testing.T) {
	svc, h, token := setupAuthenticatedHandler(t, ServiceConfig{
		TOTPConfig: &TOTPConfig{Issuer: "Test", Window: 1},
	})
	_ = enableTOTPForUser(t, svc)

	// Generate a code from a far-past time — guaranteed not to match the window.
	user, _ := svc.readModel.FindByUserID(NewUserID("authu1"))
	farPastTime := time.Now().Add(-100 * TOTPTimeStep)
	b32Secret := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(user.TOTPSecret)
	invalidCode, err := totp.GenerateCode(b32Secret, farPastTime)
	if err != nil {
		t.Fatalf("generate invalid code: %v", err)
	}

	w := authenticatedRequest(t, h, http.MethodPost, "/auth/totp/verify", token,
		`{"code":"`+invalidCode+`"}`)
	assertStatusCode(t, w, http.StatusUnauthorized)
}

func TestHandlers_TOTPNotEnabled(t *testing.T) {
	cases := []struct {
		name   string
		path   string
		body   string
		status int
	}{
		{"setup verify", "/auth/totp/setup/verify", `{"code":"000000"}`, http.StatusBadRequest},
		{"disable", "/auth/totp/disable", `{"code":"123456"}`, http.StatusBadRequest},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, h, token := setupAuthenticatedHandler(t, ServiceConfig{
				TOTPConfig: &TOTPConfig{Issuer: "Test", Window: 1},
			})
			w := authenticatedRequest(t, h, http.MethodPost, tc.path, token, tc.body)
			assertStatusCode(t, w, tc.status)
		})
	}
}

func TestHandlers_ImportUsers_InvalidEmail(t *testing.T) {
	_, h, token := setupAuthenticatedHandler(t, ServiceConfig{})
	body := `[{"email":"not-an-email","display_name":"Bad"}]`
	w := authenticatedRequest(t, h, http.MethodPost, "/auth/import?format=json", token, body)
	assertStatusCode(t, w, http.StatusOK)

	var result ImportResult
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result.Skipped != 1 {
		t.Errorf("skipped = %d, want 1", result.Skipped)
	}
	if result.Imported != 0 {
		t.Errorf("imported = %d, want 0", result.Imported)
	}
}

func TestHandlers_ImportUsers_EmptyArray(t *testing.T) {
	_, h, token := setupAuthenticatedHandler(t, ServiceConfig{})
	w := authenticatedRequest(t, h, http.MethodPost, "/auth/import?format=json", token, "[]")
	assertStatusCode(t, w, http.StatusOK)

	var result ImportResult
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result.Imported != 0 || result.Skipped != 0 {
		t.Errorf("imported=%d skipped=%d, want 0/0", result.Imported, result.Skipped)
	}
}

func TestHandlers_ImportUsers_DuplicateEmail(t *testing.T) {
	_, h, token := setupAuthenticatedHandler(t, ServiceConfig{})
	body := `[{"email":"authu1@test.com"}]`
	w := authenticatedRequest(t, h, http.MethodPost, "/auth/import?format=json", token, body)
	assertStatusCode(t, w, http.StatusOK)

	var result ImportResult
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result.Skipped != 1 {
		t.Errorf("skipped = %d, want 1 (duplicate email)", result.Skipped)
	}
}

func TestHandlers_VerifyEmail_AlreadyVerified(t *testing.T) {
	svc, h, token := setupAuthenticatedHandler(t, ServiceConfig{
		EmailVerification: &EmailVerificationConfig{},
	})
	tok, err := svc.SendVerificationEmail(context.Background(), NewUserID("authu1"))
	if err != nil {
		t.Fatalf("SendVerificationEmail: %v", err)
	}
	if err := svc.VerifyEmail(context.Background(), tok); err != nil {
		t.Fatalf("VerifyEmail: %v", err)
	}
	w := authenticatedRequest(t, h, http.MethodPost, "/auth/email/verify/send", token, "")
	assertStatusCode(t, w, http.StatusConflict)
}
