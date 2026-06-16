package usermgmt

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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

	h := NewAuthHandler(svc, HandlerConfig{Secure: new(bool)})
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	wrapped := NewSessionMiddleware(svc, "session_token")(mux)
	return svc, wrapped, reg.Session.Token
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
