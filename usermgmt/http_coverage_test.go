package usermgmt

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParseImportFormat(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		format      string
		contentType string
		want        UserDataFormat
	}{
		{"query csv", "csv", "", UserDataFormatCSV},
		{"query json", "json", "", UserDataFormatJSON},
		{"content-type csv", "", "text/csv", UserDataFormatCSV},
		{"content-type csv with charset", "", "text/csv; charset=utf-8", UserDataFormatCSV},
		{"default json when unspecified", "", "", UserDataFormatJSON},
		{"uppercase query param normalized", "CSV", "", UserDataFormatCSV},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodPost, "/auth/import?format="+tc.format, nil)
			if tc.contentType != "" {
				req.Header.Set("Content-Type", tc.contentType)
			}
			if got := parseImportFormat(req); got != tc.want {
				t.Errorf("parseImportFormat() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestHandlers_ExportCSV(t *testing.T) {
	svc, h, token := setupAuthenticatedHandler(t, ServiceConfig{})

	// Export as CSV — covers the UserDataFormatCSV branch of handleExportUsers.
	w := authenticatedRequest(t, h, http.MethodGet, "/auth/export?format=csv", token, "")
	assertStatusCode(t, w, http.StatusOK)
	if got := w.Header().Get("Content-Type"); got != "text/csv; charset=utf-8" {
		t.Errorf("Content-Type = %q, want text/csv", got)
	}

	// The registered admin user should appear in the CSV body.
	if body := w.Body.String(); body == "" {
		t.Error("expected non-empty CSV export")
	}

	_ = svc // keep service alive for the handler
}

func TestHandlers_ImportCSV(t *testing.T) {
	_, h, token := setupAuthenticatedHandler(t, ServiceConfig{})

	csvBody := "email,display_name\nimported-csv@test.com,Imported CSV\n"
	// Import as CSV via Content-Type — covers the CSV branch of handleImportUsers.
	req := httptest.NewRequest(http.MethodPost, "/auth/import", strings.NewReader(csvBody))
	req.Header.Set("Content-Type", "text/csv")
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})

	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	assertStatusCode(t, w, http.StatusOK)
}

// --- TOTP / verification handler error paths ---

func totpHandlerSetup(t *testing.T) (http.Handler, string) {
	t.Helper()
	_, h, token := setupAuthenticatedHandler(t, ServiceConfig{
		TOTP: newTestTOTPProvider("Test"),
	})
	return h, token
}

// assertInvalidJSONDecode posts malformed JSON to path and asserts that the
// handler returns 400 (decode error). Used to exercise the body-decode
// rejection branch on every JSON-decoding handler.
func assertInvalidJSONDecode(t *testing.T, path string) {
	t.Helper()
	h, token := totpHandlerSetup(t)
	w := authenticatedRequest(t, h, http.MethodPost, path, token, "{bad json")
	assertStatusCode(t, w, http.StatusBadRequest)
}

func TestHandlers_TOTPVerify_InvalidBody(t *testing.T) {
	// Invalid JSON body hits the decode-error branch of handleTOTPCode.
	assertInvalidJSONDecode(t, "/auth/totp/verify")
}

func TestHandlers_TOTPDisable_InvalidBody(t *testing.T) {
	assertInvalidJSONDecode(t, "/auth/totp/disable")
}

func TestHandlers_EmailVerify_InvalidBody(t *testing.T) {
	_, h, token := setupAuthenticatedHandler(t, ServiceConfig{
		EmailVerification: &EmailVerificationConfig{},
	})
	w := authenticatedRequest(t, h, http.MethodPost, "/auth/email/verify", token, "{bad json")
	assertStatusCode(t, w, http.StatusBadRequest)
}

func TestHandlers_EmailVerify_InvalidToken(t *testing.T) {
	_, h, token := setupAuthenticatedHandler(t, ServiceConfig{
		EmailVerification: &EmailVerificationConfig{},
	})
	// A token that was never issued hits the VerifyEmail error-return branch.
	w := authenticatedRequest(t, h, http.MethodPost, "/auth/email/verify", token, `{"token":"never-issued"}`)
	if w.Code == http.StatusOK {
		t.Errorf("expected non-200 for invalid token, got %d", w.Code)
	}
}

func TestHandlers_TOTPSetup_AlreadyEnabled(t *testing.T) {
	svc, h, token := setupAuthenticatedHandler(t, ServiceConfig{
		TOTP: newTestTOTPProvider("Test"),
	})
	// First setup succeeds.
	w1 := authenticatedRequest(t, h, http.MethodPost, "/auth/totp/setup", token, "")
	assertStatusCode(t, w1, http.StatusOK)

	// Enable TOTP fully (setup + verify) so a second setup returns an error.
	user, _ := svc.readModel.FindByUserID(NewUserID("authu1"))
	_, _ = svc.EnableTOTP(context.Background(), user.ID)
	_ = svc.VerifyTOTPSetup(context.Background(), user.ID, testTOTPValidCode)

	// Second setup now hits the EnableTOTP error path (already enabled → 409 Conflict).
	w2 := authenticatedRequest(t, h, http.MethodPost, "/auth/totp/setup", token, "")
	assertStatusCode(t, w2, http.StatusConflict)
}
