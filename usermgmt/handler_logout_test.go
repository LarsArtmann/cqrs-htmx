package usermgmt

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandlers_Logout(t *testing.T) {
	_, mux := setupMux(t)
	resp := registerUser(t, mux)

	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: resp.Session.Token})
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	assertStatusCode(t, w, http.StatusOK)
	assertCookie(t, w, "session_token", func(c *http.Cookie) bool { return c.MaxAge == -1 })
}

func TestHandlers_Logout_NoSession(t *testing.T) {
	_, mux := setupMux(t)
	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	assertStatusCode(t, w, http.StatusUnauthorized)
}
