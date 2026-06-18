package usermgmt

import (
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
	registerTestUser(t, svc, "u1", "hbr@test.com")

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	w := postJSON(t, mux, "/auth/webauthn/register/begin", `{"user_id":"u1"}`)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandler_WebAuthnBeginRegistration_BadBody(t *testing.T) {
	h, _ := newWebAuthnHandler(t)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	w := postJSON(t, mux, "/auth/webauthn/register/begin", `invalid`)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandler_WebAuthnBegin_UserNotFound(t *testing.T) {
	cases := []struct {
		name string
		path string
		body string
	}{
		{"registration", "/auth/webauthn/register/begin", `{"user_id":"ghost"}`},
		{"login", "/auth/webauthn/login/begin", `{"email":"nobody@test.com"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h, _ := newWebAuthnHandler(t)
			mux := http.NewServeMux()
			h.RegisterRoutes(mux)

			w := postJSON(t, mux, tc.path, tc.body)
			if w.Code != http.StatusNotFound {
				t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
			}
		})
	}
}

func TestHandler_WebAuthnBeginLogin_Success(t *testing.T) {
	h, svc := newWebAuthnHandler(t)
	registerTestUser(t, svc, "u1", "hbl@test.com")

	cred := WebAuthnCredential{
		ID: []byte{1, 2, 3}, PublicKey: []byte{4, 5, 6}, AttestationType: "none",
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
	registerTestUser(t, svc, "u1", "noconfig@test.com")

	h := NewAuthHandler(svc, HandlerConfig{Secure: new(bool)})
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	w := postJSON(t, mux, "/auth/webauthn/register/begin", `{"user_id":"u1"}`)
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
		name string
		path string
	}{
		{"register", "/auth/webauthn/register/finish"},
		{"login", "/auth/webauthn/login/finish"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h, _ := newWebAuthnHandler(t)
			mux := http.NewServeMux()
			h.RegisterRoutes(mux)

			w := postJSON(t, mux, tc.path, "")
			if w.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d (missing user_id)", w.Code, http.StatusBadRequest)
			}
		})
	}
}
