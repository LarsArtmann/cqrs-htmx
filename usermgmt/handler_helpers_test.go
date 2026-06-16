package usermgmt

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func setupMux(t *testing.T) (*Service, *http.ServeMux) {
	t.Helper()
	svc := newTestServiceWithAuthz(t)
	h := NewAuthHandler(svc, HandlerConfig{Secure: new(bool)})
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	return svc, mux
}

func registerUser(t *testing.T, mux *http.ServeMux) *RegisterResponse {
	t.Helper()
	w := postJSON(t, mux, "/auth/register",
		`{"id":"u1","email":"a@b.com","display_name":"Test"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("register: expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var resp RegisterResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal register response: %v", err)
	}
	return &resp
}

func assertCookie(
	t *testing.T,
	w *httptest.ResponseRecorder,
	name string,
	check func(*http.Cookie) bool,
) {
	t.Helper()
	for _, c := range w.Result().Cookies() {
		if c.Name == name && check(c) {
			return
		}
	}
	t.Errorf("expected cookie %q matching condition", name)
}

func assertStatusCode(t *testing.T, w *httptest.ResponseRecorder, expected int) {
	t.Helper()
	if w.Code != expected {
		t.Errorf("expected %d, got %d: %s", expected, w.Code, w.Body.String())
	}
}
