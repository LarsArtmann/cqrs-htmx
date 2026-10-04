package usermgmt

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func newWebAuthnHandler(t *testing.T) (*AuthHandler, *Service) {
	t.Helper()
	svc := newWebAuthnTestService(t)
	h := NewAuthHandler(svc, HandlerConfig{Secure: new(bool)})
	return h, svc
}

func TestHandler_WebAuthnBeginRegistration_Success(t *testing.T) {
	h, svc := newWebAuthnHandler(t)
	reg := registerTestUser(t, svc, "u1", "hbr@test.com")

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	w := authenticatedRequest(t, mux, http.MethodPost, "/auth/webauthn/register/begin",
		reg.Session.Token, fmt.Sprintf(`{"user_id":%q}`, NewUserID("u1").Get().String()))
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandler_WebAuthnBeginRegistration_BadBody(t *testing.T) {
	h, svc := newWebAuthnHandler(t)
	reg := registerTestUser(t, svc, "u1", "bbb@test.com")
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	// Authenticated so the decode error path (not the session gate) is hit.
	w := authenticatedRequest(t, mux, http.MethodPost, "/auth/webauthn/register/begin", reg.Session.Token, `invalid`)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandler_WebAuthnBegin_UserNotFound(t *testing.T) {
	cases := []struct {
		name  string
		path  string
		body  string
		token string // session cookie token; empty = unauthenticated
		want  int
	}{
		// Registration is owner-session-gated: a target that is not the session
		// user is rejected with 403 before any user lookup.
		{"registration mismatch", "/auth/webauthn/register/begin", `{"user_id":"ghost"}`, "auth", http.StatusForbidden},
		{"login", "/auth/webauthn/login/begin", `{"email":"nobody@test.com"}`, "", http.StatusNotFound},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h, svc := newWebAuthnHandler(t)
			token := ""
			if tc.token != "" {
				token = registerTestUser(t, svc, "u1", "hbr@test.com").Session.Token
			}
			mux := http.NewServeMux()
			h.RegisterRoutes(mux)

			w := authenticatedRequest(t, mux, http.MethodPost, tc.path, token, tc.body)
			if w.Code != tc.want {
				t.Errorf("status = %d, want %d", w.Code, tc.want)
			}
		})
	}
}

func TestHandler_WebAuthnBeginLogin_Success(t *testing.T) {
	h, svc := newWebAuthnHandler(t)
	registerTestUser(t, svc, "u1", "hbl@test.com")

	cred := WebAuthnCredential{
		CredentialCore: CredentialCore{
			ID: []byte{1, 2, 3}, PublicKey: []byte{4, 5, 6}, AttestationType: "none",
		},
	}
	addTestCredential(t, svc, NewUserID("u1"), cred)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	w := postJSON(t, mux, "/auth/webauthn/login/begin", `{"email":"hbl@test.com"}`)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestHandler_WebAuthnBeginLogin_NoCredentials(t *testing.T) {
	h, svc := newWebAuthnHandler(t)
	registerTestUser(t, svc, "u1", "nocred@test.com")

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	w := postJSON(t, mux, "/auth/webauthn/login/begin", `{"email":"nocred@test.com"}`)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d (no credentials)", w.Code, http.StatusNotFound)
	}
}

func TestHandler_WebAuthnBeginLogin_BadBody(t *testing.T) {
	h, _ := newWebAuthnHandler(t)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	w := postJSON(t, mux, "/auth/webauthn/login/begin", strings.Repeat("x", 2<<20))
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandler_WebAuthnNotConfigured_BeginRegistration(t *testing.T) {
	svc := newTestService(t)
	reg := registerTestUser(t, svc, "u1", "noconfig@test.com")

	h := NewAuthHandler(svc, HandlerConfig{Secure: new(bool)})
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	w := authenticatedRequest(t, mux, http.MethodPost, "/auth/webauthn/register/begin",
		reg.Session.Token, fmt.Sprintf(`{"user_id":%q}`, NewUserID("u1").Get().String()))
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d (webauthn not configured)", w.Code, http.StatusUnauthorized)
	}
}

// --- Finish{Registration,Login} guard tests (table-driven) ---

func TestHandler_WebAuthnFinish_NoSession(t *testing.T) {
	cases := []struct {
		name        string
		email       string
		registerID  string
		path        string
		queryString string
	}{
		{"register", "hfr@test.com", "u1", "/auth/webauthn/register/finish", "user_id=u1&credential_name=key1"},
		{"login", "hfl@test.com", "u1", "/auth/webauthn/login/finish", "user_id=u1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h, svc := newWebAuthnHandler(t)
			registerTestUser(t, svc, tc.registerID, tc.email)

			mux := http.NewServeMux()
			h.RegisterRoutes(mux)

			w := postJSON(t, mux, tc.path+"?"+tc.queryString, "")
			if w.Code != http.StatusUnauthorized {
				t.Errorf("status = %d, want %d (no session)", w.Code, http.StatusUnauthorized)
			}
		})
	}
}

func TestHandler_WebAuthnFinish_MissingUserID(t *testing.T) {
	cases := []struct {
		name  string
		path  string
		token string // registration finish is session-gated; login finish is public
	}{
		{"register", "/auth/webauthn/register/finish", "auth"},
		{"login", "/auth/webauthn/login/finish", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h, svc := newWebAuthnHandler(t)
			token := ""
			if tc.token != "" {
				token = registerTestUser(t, svc, "u1", "mfu@test.com").Session.Token
			}
			mux := http.NewServeMux()
			h.RegisterRoutes(mux)

			w := authenticatedRequest(t, mux, http.MethodPost, tc.path, token, "")
			if w.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d (missing user_id)", w.Code, http.StatusBadRequest)
			}
		})
	}
}
