package usermgmt

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestHandlers_Login(t *testing.T) {
	_, mux := setupMux(t)
	registerUser(t, mux)

	w := postJSON(t, mux, "/auth/login", `{"email":"a@b.com","password":"secret12"}`)
	assertStatusCode(t, w, http.StatusOK)

	var resp LoginResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.User.ID != NewUserID("u1") {
		t.Errorf("expected user ID u1, got %s", resp.User.ID)
	}
}

func TestHandlers_Login_WrongPassword(t *testing.T) {
	_, mux := setupMux(t)
	registerUser(t, mux)
	w := postJSON(t, mux, "/auth/login", `{"email":"a@b.com","password":"wrong"}`)
	assertStatusCode(t, w, http.StatusUnauthorized)
}

func TestHandlers_Login_InvalidJSON(t *testing.T) {
	_, mux := setupMux(t)
	w := postJSON(t, mux, "/auth/login", "not json")
	assertStatusCode(t, w, http.StatusBadRequest)
}

func TestLoginRequest_MaxPasswordLength(t *testing.T) {
	req := LoginRequest{Email: "test@example.com", Password: strings.Repeat("x", 129)}
	if err := req.Validate(); err == nil {
		t.Fatal("expected validation error for password > 128 chars")
	}
}
