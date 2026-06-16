package usermgmt

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandlers_Register(t *testing.T) {
	_, mux := setupMux(t)

	resp := registerUser(t, mux)
	if resp.User.ID != NewUserID("u1") {
		t.Errorf("expected user ID u1, got %s", resp.User.ID)
	}

	req := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader("{}"))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
}

func TestHandlers_Register_SetsCookie(t *testing.T) {
	_, mux := setupMux(t)
	w := postJSON(t, mux, "/auth/register",
		`{"id":"u1","email":"cookie@test.com"}`)
	assertStatusCode(t, w, http.StatusCreated)
	assertCookie(t, w, "session_token", func(c *http.Cookie) bool {
		return c.Value != "" && c.HttpOnly && c.SameSite == http.SameSiteStrictMode && !c.Secure
	})
}

func TestHandlers_Register_DuplicateEmail(t *testing.T) {
	_, mux := setupMux(t)
	registerUser(t, mux)
	w := postJSON(t, mux, "/auth/register",
		`{"id":"u2","email":"a@b.com"}`)
	assertStatusCode(t, w, http.StatusConflict)
}

func TestHandlers_BadRequestBody(t *testing.T) {
	_, mux := setupMux(t)
	w := postJSON(t, mux, "/auth/register", "not json")
	assertStatusCode(t, w, http.StatusBadRequest)
}

func wrapSessionMiddleware(
	t *testing.T,
	svc *Service,
	token string,
	handler http.HandlerFunc,
) *httptest.ResponseRecorder {
	t.Helper()
	h := NewSessionMiddleware(svc, "session_token")(handler)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if token != "" {
		req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}
