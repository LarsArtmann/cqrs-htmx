package usermgmt

import (
	"encoding/base64"
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"testing"
)

// bareAuthMux registers auth routes on a BARE mux — no external session
// middleware — the mount shape that used to leave every currentUser-reading
// route dead and the enrollment ceremonies unauthenticated.
func bareAuthMux(t *testing.T, config ServiceConfig) (*Service, *http.ServeMux, string) {
	t.Helper()
	svc, err := NewService(config)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	t.Cleanup(svc.Stop)
	reg := registerTestUser(t, svc, "gateu1", "gateu1@test.com")
	mux := http.NewServeMux()
	NewAuthHandler(svc, HandlerConfig{Secure: new(bool)}).RegisterRoutes(mux)
	return svc, mux, reg.Session.Token
}

func bareWebAuthnAuthMux(t *testing.T) (*Service, *http.ServeMux, string, string) {
	t.Helper()
	svc, mux, token := bareAuthMux(t, ServiceConfig{WebAuthn: testWebAuthnProvider{}})
	return svc, mux, token, NewUserID("gateu1").Get().String()
}

// bareFullAuthMux is bareAuthMux with every auth strategy configured, so the
// session-dependent subset can be exercised through real handler paths.
func bareFullAuthMux(t *testing.T) (*Service, *http.ServeMux, string) {
	t.Helper()
	svc, mux, token := bareAuthMux(t, ServiceConfig{
		WebAuthn:          testWebAuthnProvider{},
		EmailVerification: &EmailVerificationConfig{},
		TOTP:              newTestTOTPProvider("Test"),
		OAuth2:            testOAuth2Provider{},
	})
	return svc, mux, token
}

// --- M4: owner-match gate on both enrollment ceremonies ---

func TestSessionGate_RegisterBegin_NoSession(t *testing.T) {
	_, mux, _, _ := bareWebAuthnAuthMux(t)

	w := authenticatedRequest(t, mux, http.MethodPost, "/auth/webauthn/register/begin", "", `{"user_id":"anyone"}`)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d (no session)", w.Code, http.StatusUnauthorized)
	}
}

func TestSessionGate_RegisterBegin_MismatchedTarget(t *testing.T) {
	_, mux, token, _ := bareWebAuthnAuthMux(t)
	other := NewUserID("gateu2").Get().String()

	w := authenticatedRequest(t, mux, http.MethodPost, "/auth/webauthn/register/begin",
		token, `{"user_id":"`+other+`"}`)
	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d (target is another user)", w.Code, http.StatusForbidden)
	}
}

func TestSessionGate_RegisterBegin_SelfTarget(t *testing.T) {
	_, mux, token, self := bareWebAuthnAuthMux(t)

	w := authenticatedRequest(t, mux, http.MethodPost, "/auth/webauthn/register/begin",
		token, `{"user_id":"`+self+`"}`)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d (self target), body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestSessionGate_RegisterFinish_MirrorsBeginGates(t *testing.T) {
	_, mux, token, self := bareWebAuthnAuthMux(t)
	other := NewUserID("gateu2").Get().String()

	// The 200 path needs a live ceremony: begin first (same session).
	if w := authenticatedRequest(t, mux, http.MethodPost, "/auth/webauthn/register/begin",
		token, `{"user_id":"`+self+`"}`); w.Code != http.StatusOK {
		t.Fatalf("begin ceremony setup: status = %d, want 200", w.Code)
	}

	cases := []struct {
		name   string
		token  string
		query  string
		want   int
		reason string
	}{
		{"no session", "", "user_id=" + self, http.StatusUnauthorized, "no session"},
		{"mismatched target", token, "user_id=" + other, http.StatusForbidden, "target is another user"},
		{"missing target", token, "", http.StatusBadRequest, "user_id is required"},
		{"self target", token, "user_id=" + self, http.StatusOK, "self target passes the gate"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := authenticatedRequest(t, mux, http.MethodPost,
				"/auth/webauthn/register/finish?"+tc.query, tc.token, "{}")
			if w.Code != tc.want {
				t.Errorf("status = %d, want %d (%s)", w.Code, tc.want, tc.reason)
			}
		})
	}
}

// TestBareMount_LoginPageFlow_WorksUnderGate proves the first-party
// constraint from ADR-0055: the login page drives register → begin → finish
// in one browser flow, and its fetches carry the session cookie
// (credentials: "same-origin"). A cookie-jar client against the bare mount is
// the HTTP-level equivalent of that browser behavior — the gate must let the
// whole dance through.
func TestBareMount_LoginPageFlow_WorksUnderGate(t *testing.T) {
	_, mux, _, _ := bareWebAuthnAuthMux(t)
	server := httptest.NewServer(mux)
	defer server.Close()

	client := &http.Client{Jar: newJar(t)}

	regStatus, regBody := postJSONStatus(t, client, server.URL+"/auth/register",
		`{"email":"lp-flow@test.com","display_name":"LP Flow"}`)
	if regStatus != http.StatusCreated {
		t.Fatalf("register: status = %d, want 201", regStatus)
	}
	var reg struct {
		User struct {
			ID string `json:"id"`
		} `json:"user"`
	}
	if err := json.Unmarshal(regBody, &reg); err != nil {
		t.Fatalf("decode register response: %v", err)
	}

	beginStatus, _ := postJSONStatus(t, client, server.URL+"/auth/webauthn/register/begin",
		`{"user_id":"`+reg.User.ID+`"}`)
	if beginStatus != http.StatusOK {
		t.Fatalf("begin under gate with register-issued cookie: status = %d, want 200", beginStatus)
	}

	finishStatus, _ := postJSONStatus(t, client,
		server.URL+"/auth/webauthn/register/finish?user_id="+reg.User.ID+"&credential_name=Passkey", "{}")
	if finishStatus != http.StatusOK {
		t.Fatalf("finish under gate: status = %d, want 200", finishStatus)
	}
}

func TestBareMount_SessionRoutes_ReachableWithCookie(t *testing.T) {
	_, mux, token := bareFullAuthMux(t)
	svc, wmux, wtoken, _ := bareWebAuthnAuthMux(t)
	addTestCredential(t, svc, NewUserID("gateu1"), WebAuthnCredential{
		CredentialCore: CredentialCore{
			ID: []byte{1, 2, 3}, PublicKey: []byte{4, 5, 6}, AttestationType: "none",
		},
	})
	credID := base64.RawURLEncoding.EncodeToString([]byte{1, 2, 3})

	exact := []struct {
		name   string
		mux    *http.ServeMux
		token  string
		method string
		path   string
		want   int
	}{
		{"me", mux, token, http.MethodGet, "/auth/me", http.StatusOK},
		{"credentials list", mux, token, http.MethodGet, "/auth/credentials", http.StatusOK},
		{"email verify send", mux, token, http.MethodPost, "/auth/email/verify/send", http.StatusOK},
		{"totp setup", mux, token, http.MethodPost, "/auth/totp/setup", http.StatusOK},
		{"oauth2 unlink", mux, token, http.MethodPost, "/auth/oauth/github/unlink", http.StatusNotFound},
		{"credentials delete", wmux, wtoken, http.MethodDelete, "/auth/credentials/" + credID, http.StatusOK},
	}
	for _, tc := range exact {
		t.Run(tc.name, func(t *testing.T) {
			w := authenticatedRequest(t, tc.mux, tc.method, tc.path, tc.token, "")
			if w.Code != tc.want {
				t.Errorf("status = %d, want %d (bare mount + valid cookie), body: %s",
					w.Code, tc.want, w.Body.String())
			}
		})
	}
}

// TestBareMount_SessionRoutes_FailClosedWithoutCookie pins the complement:
// without a cookie every route in the session-dependent subset answers 401 —
// the handlers keep their fail-closed behavior; the wrapper only enriches.
func TestBareMount_SessionRoutes_FailClosedWithoutCookie(t *testing.T) {
	_, mux, _, _ := bareWebAuthnAuthMux(t)

	cases := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/auth/me"},
		{http.MethodGet, "/auth/credentials"},
		{http.MethodDelete, "/auth/credentials/AAAA"},
		{http.MethodPost, "/auth/webauthn/register/begin"},
		{http.MethodPost, "/auth/webauthn/register/finish"},
		{http.MethodPost, "/auth/totp/setup"},
		{http.MethodPost, "/auth/totp/verify"},
		{http.MethodPost, "/auth/totp/disable"},
		{http.MethodPost, "/auth/email/verify/send"},
		{http.MethodGet, "/auth/export"},
		{http.MethodPost, "/auth/import"},
		{http.MethodPost, "/auth/oauth/github/unlink"},
	}
	for _, tc := range cases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			w := authenticatedRequest(t, mux, tc.method, tc.path, "", "")
			if w.Code != http.StatusUnauthorized {
				t.Errorf("status = %d, want %d (no cookie, fail-closed)", w.Code, http.StatusUnauthorized)
			}
		})
	}
}
