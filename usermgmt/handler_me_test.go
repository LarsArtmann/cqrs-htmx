package usermgmt

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandlers_Me_Unauthenticated(t *testing.T) {
	_, mux := setupMux(t)
	req := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	assertStatusCode(t, w, http.StatusUnauthorized)
}

func TestHandlers_Me_Authenticated(t *testing.T) {
	svc, mux := setupMux(t)
	regResp := registerUser(t, mux)

	_, err := svc.Authenticate(context.Background(), regResp.Session.Token)
	if err != nil {
		t.Fatalf("authenticate: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: regResp.Session.Token})

	meHandler := NewSessionMiddleware(svc, "session_token")(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u, ok := UserFromContext(r.Context())
			if !ok || u == nil {
				writeError(w, http.StatusUnauthorized, "not authenticated")
				return
			}
			writeJSON(w, http.StatusOK, u)
		}),
	)
	w := httptest.NewRecorder()
	meHandler.ServeHTTP(w, req)
	assertStatusCode(t, w, http.StatusOK)
}
