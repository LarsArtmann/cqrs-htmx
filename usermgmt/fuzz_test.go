package usermgmt

import (
	"net/http"
	"net/http/httptest"
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

func FuzzWebAuthnBeginRegistration_Body(f *testing.F) {
	f.Add(`{"user_id":"abc"}`)
	f.Add(`{}`)
	f.Add(``)
	f.Add(`{"user_id":""}`)
	f.Add(`not json at all`)
	f.Add(`{"user_id":"` + strings.Repeat("x", 1000) + `"}`)

	f.Fuzz(func(t *testing.T, body string) {
		svc := newTestService(t)
		h := NewAuthHandler(svc)
		mux := http.NewServeMux()
		h.RegisterRoutes(mux)

		r := httptest.NewRequest(http.MethodPost, "/auth/webauthn/register/begin", strings.NewReader(body))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		// Should never panic — only return 400/500
		if w.Code == 0 {
			t.Errorf("zero status code for body %q", body)
		}
	})
}

func FuzzWebAuthnBeginLogin_Body(f *testing.F) {
	f.Add(`{"email":"a@test.com"}`)
	f.Add(`{}`)
	f.Add(``)
	f.Add(`{"email":""}`)
	f.Add(`not json`)
	f.Add(`{"email":"` + strings.Repeat("x", 500) + `@test.com"}`)

	f.Fuzz(func(t *testing.T, body string) {
		svc := newTestService(t)
		h := NewAuthHandler(svc)
		mux := http.NewServeMux()
		h.RegisterRoutes(mux)

		r := httptest.NewRequest(http.MethodPost, "/auth/webauthn/login/begin", strings.NewReader(body))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		if w.Code == 0 {
			t.Errorf("zero status code for body %q", body)
		}
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

		r := httptest.NewRequest(http.MethodDelete, "/auth/credentials/"+encodedID, nil)
		r = r.WithContext(WithUser(r.Context(), &User{ID: NewUserID("01HK1549P84T9XF8R94E960633")}))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		if w.Code == 0 {
			t.Errorf("zero status code for id %q", encodedID)
		}
	})
}
