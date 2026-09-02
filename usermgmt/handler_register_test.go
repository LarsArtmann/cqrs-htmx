package usermgmt

import (
	"encoding/json/v2"
	"fmt"
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
		fmt.Sprintf(`{"id":%q,"email":"cookie@test.com"}`, NewUserID("u1").Get().String()))
	assertStatusCode(t, w, http.StatusCreated)
	assertCookie(t, w, "session_token", func(c *http.Cookie) bool {
		return c.Value != "" && c.HttpOnly && c.SameSite == http.SameSiteStrictMode && !c.Secure
	})
}

func TestHandlers_Register_DuplicateEmail(t *testing.T) {
	_, mux := setupMux(t)
	registerUser(t, mux)
	w := postJSON(t, mux, "/auth/register",
		fmt.Sprintf(`{"id":%q,"email":"a@b.com"}`, NewUserID("u2").Get().String()))
	assertStatusCode(t, w, http.StatusConflict)
}

func TestHandlers_Register_AutoGenerateID(t *testing.T) {
	_, mux := setupMux(t)
	w := postJSON(t, mux, "/auth/register", `{"email":"autoid@test.com","display_name":"Auto"}`)
	assertStatusCode(t, w, http.StatusCreated)
	resp := decodeJSON[RegisterResponse](t, w)
	if resp.User.ID.IsZero() {
		t.Error("server should auto-generate a user ID when none is provided")
	}
}

func TestHandlers_Register_BadRequestBody(t *testing.T) {
	_, mux := setupMux(t)
	w := postJSON(t, mux, "/auth/register", "not json")
	assertStatusCode(t, w, http.StatusBadRequest)
}

func TestHandlers_Register_MaxUsersReached_Returns403(t *testing.T) {
	svc := newTestServiceWithConfig(t, ServiceConfig{Authz: newTestAuthz(t), MaxUsers: 1})
	h := NewAuthHandler(svc, HandlerConfig{Secure: new(bool)})
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	w := postJSON(t, mux, "/auth/register",
		fmt.Sprintf(`{"id":%q,"email":"first@test.com"}`, NewUserID("u1").Get().String()))
	assertStatusCode(t, w, http.StatusCreated)

	w = postJSON(t, mux, "/auth/register",
		fmt.Sprintf(`{"id":%q,"email":"second@test.com"}`, NewUserID("u2").Get().String()))
	assertStatusCode(t, w, http.StatusForbidden)

	var body struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal error body: %v", err)
	}
	if body.Error != friendlyRegistrationClosedMessage {
		t.Fatalf("error body = %q, want the friendly registration-closed message", body.Error)
	}
	if strings.Contains(body.Error, "usermgmt") {
		t.Fatalf("error body leaks the machine code: %q", body.Error)
	}
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
