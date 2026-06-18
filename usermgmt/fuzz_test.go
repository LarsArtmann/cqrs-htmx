package usermgmt

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func FuzzRegisterRequest_Validate(f *testing.F) {
	f.Add("u1", "test@example.com", "Alice")
	f.Add("", "", "")
	f.Add("u2", "not-an-email", strings.Repeat("x", 101))
	f.Add("u3", "a@b.com", "")
	f.Add("u4", "valid@email.com", "OK")

	f.Fuzz(func(_ *testing.T, id, email, displayName string) {
		req := RegisterRequest{
			ID:          NewUserID(id),
			Email:       email,
			DisplayName: displayName,
		}
		_ = req.Validate()
	})
}

// fuzzWebAuthnPost exercises a WebAuthn endpoint with a fuzz-supplied body
// and asserts the handler returns a non-zero status code (i.e. does not panic).
func fuzzWebAuthnPost(t *testing.T, path, body string) {
	t.Helper()
	svc := newTestService(t)
	h := NewAuthHandler(svc)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	r := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code == 0 {
		t.Errorf("zero status code for body %q", body)
	}
}

// addFuzzSeeds is a tiny helper that registers a list of seed strings
// with the fuzz target. The repeated f.Add("...") pattern in
// FuzzWebAuthnBegin{Registration,Login}_Body was identical except for
// the literal bodies.
func addFuzzSeeds(f *testing.F, seeds []string) {
	f.Helper()
	for _, s := range seeds {
		f.Add(s)
	}
}

// fuzzWebAuthnEndpoint is the body shared by FuzzWebAuthnBeginRegistration_Body
// and FuzzWebAuthnBeginLogin_Body: seed the corpus with the given bodies,
// then for each fuzz-supplied body POST it to the given path and assert
// the handler returns a non-zero status code.
func fuzzWebAuthnEndpoint(f *testing.F, path string, seeds []string) {
	f.Helper()
	addFuzzSeeds(f, seeds)
	f.Fuzz(func(t *testing.T, body string) {
		fuzzWebAuthnPost(t, path, body)
	})
}

func FuzzWebAuthnBeginRegistration_Body(f *testing.F) {
	fuzzWebAuthnEndpoint(f, "/auth/webauthn/register/begin", []string{
		`{"user_id":"abc"}`,
		`{}`,
		``,
		`{"user_id":""}`,
		`not json at all`,
		`{"user_id":"` + strings.Repeat("x", 1000) + `"}`,
	})
}

func FuzzWebAuthnBeginLogin_Body(f *testing.F) {
	fuzzWebAuthnEndpoint(f, "/auth/webauthn/login/begin", []string{
		`{"email":"a@test.com"}`,
		`{}`,
		``,
		`{"email":""}`,
		`not json`,
		`{"email":"` + strings.Repeat("x", 500) + `@test.com"}`,
	})
}

func FuzzCredentialID_Base64Decode(f *testing.F) {
	f.Add("abc123")
	f.Add("")
	f.Add("!!!")
	f.Add("AAAAAAAAAAAAAAAAAAAAAA")
	f.Add("../../etc/passwd")

	f.Fuzz(func(t *testing.T, encodedID string) {
		svc := newTestService(t)
		h := NewAuthHandler(svc)
		mux := http.NewServeMux()
		h.RegisterRoutes(mux)

		r := httptest.NewRequest(http.MethodDelete, "/auth/credentials/"+url.PathEscape(encodedID), nil)
		r = r.WithContext(WithUser(r.Context(), &User{ID: NewUserID("01HK1549P84T9XF8R94E960633")}))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		if w.Code == 0 {
			t.Errorf("zero status code for id %q", encodedID)
		}
	})
}
