package usermgmt

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
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
	if err := svc.UpdateRoles(context.Background(), reg.User.ID, []Role{RoleAdmin}, "test"); err != nil {
		t.Fatalf("UpdateRoles: %v", err)
	}

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

func TestHandlers_TOTPVerify(t *testing.T) {
	svc, h, token := setupAuthenticatedHandler(t, ServiceConfig{
		TOTPConfig: &TOTPConfig{Issuer: "Test", Window: 1},
	})
	setup, err := svc.EnableTOTP(context.Background(), NewUserID("authu1"))
	if err != nil {
		t.Fatalf("EnableTOTP: %v", err)
	}
	code := currentTOTPCode(t, decodeSecret(t, setup.Secret))
	if err := svc.VerifyTOTPSetup(context.Background(), NewUserID("authu1"), code); err != nil {
		t.Fatalf("VerifyTOTPSetup: %v", err)
	}

	code2 := currentTOTPCode(t, decodeSecret(t, setup.Secret))
	w := authenticatedRequest(t, h, http.MethodPost, "/auth/totp/verify", token,
		`{"code":"`+code2+`"}`)
	assertStatusCode(t, w, http.StatusOK)
}

func TestHandlers_TOTPDisable(t *testing.T) {
	svc, h, token := setupAuthenticatedHandler(t, ServiceConfig{
		TOTPConfig: &TOTPConfig{Issuer: "Test", Window: 1},
	})
	setup, err := svc.EnableTOTP(context.Background(), NewUserID("authu1"))
	if err != nil {
		t.Fatalf("EnableTOTP: %v", err)
	}
	code := currentTOTPCode(t, decodeSecret(t, setup.Secret))
	if err := svc.VerifyTOTPSetup(context.Background(), NewUserID("authu1"), code); err != nil {
		t.Fatalf("VerifyTOTPSetup: %v", err)
	}

	w := authenticatedRequest(t, h, http.MethodPost, "/auth/totp/disable", token, "")
	assertStatusCode(t, w, http.StatusOK)
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

func TestHandlers_ImportUsers_JSON(t *testing.T) {
	_, h, token := setupAuthenticatedHandler(t, ServiceConfig{})
	body := `[{"email":"imported1@test.com","display_name":"Imported One"}]`
	w := authenticatedRequest(t, h, http.MethodPost, "/auth/import?format=json", token, body)
	assertStatusCode(t, w, http.StatusOK)

	var result ImportResult
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result.Imported != 1 {
		t.Errorf("imported = %d, want 1", result.Imported)
	}
}

func TestHandlers_ImportUsers_CSV(t *testing.T) {
	_, h, token := setupAuthenticatedHandler(t, ServiceConfig{})
	body := "email,display_name\ncsvimport@test.com,CSV User\n"
	w := authenticatedRequest(t, h, http.MethodPost, "/auth/import?format=csv", token, body)
	assertStatusCode(t, w, http.StatusOK)

	var result ImportResult
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result.Imported != 1 {
		t.Errorf("imported = %d, want 1", result.Imported)
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
	if err := svc.UpdateRoles(context.Background(), reg.User.ID, []Role{RoleAdmin}, "test"); err != nil {
		t.Fatalf("UpdateRoles: %v", err)
	}
	sess, err := svc.sessions.Create(context.Background(), reg.User.ID, defaultSessionTTL)
	if err != nil {
		t.Fatalf("Create session: %v", err)
	}

	h := NewAuthHandler(svc, HandlerConfig{
		Secure:           new(bool),
		ImportRateLimit:  RegistrationRateLimitConfig{Enabled: true, MaxRequests: 1, Window: time.Minute},
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
		Secure:       new(bool),
		TOTPRateLimit: RegistrationRateLimitConfig{Enabled: true, MaxRequests: 1, Window: time.Minute},
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
		Secure:               new(bool),
		VerificationRateLimit: RegistrationRateLimitConfig{Enabled: true, MaxRequests: 1, Window: time.Minute},
	})
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	wrapped := NewSessionMiddleware(svc, "session_token")(mux)

	w1 := authenticatedRequest(t, wrapped, http.MethodPost, "/auth/email/verify/send", sess.Token, "")
	assertStatusCode(t, w1, http.StatusOK)

	w2 := authenticatedRequest(t, wrapped, http.MethodPost, "/auth/email/verify/send", sess.Token, "")
	assertStatusCode(t, w2, http.StatusTooManyRequests)
}
