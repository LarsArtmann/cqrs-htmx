package usermgmt

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWebAuthn_FinishRegistration_UserNotFound(t *testing.T) {
	svc := newWebAuthnTestService(t)
	r := httptest.NewRequest(http.MethodPost, "/finish", nil)

	if err := svc.FinishRegistration(context.Background(), NewUserID("ghost"), r, "key"); err == nil {
		t.Fatal("expected error for non-existent user")
	}
}

func TestWebAuthn_FinishRegistration_SessionNotFound(t *testing.T) {
	svc := newWebAuthnTestService(t)
	reg := registerTestUser(t, svc, "wa-snf", "wasnf@test.com")
	r := httptest.NewRequest(http.MethodPost, "/finish", nil)

	// No BeginRegistration was called, so the session store has no challenge.
	if err := svc.FinishRegistration(context.Background(), reg.User.ID, r, "key"); err == nil {
		t.Fatal("expected session-not-found error")
	}
}

func TestWebAuthn_FinishLogin_SessionNotFound(t *testing.T) {
	svc := newWebAuthnTestService(t)
	reg := registerTestUser(t, svc, "wa-flsnf", "waflsnf@test.com")
	r := httptest.NewRequest(http.MethodPost, "/finish", nil)

	// No BeginLogin was called, so no assertion session exists.
	if _, err := svc.FinishLogin(context.Background(), reg.User.ID, r); err == nil {
		t.Fatal("expected session-not-found error")
	}
}

func TestWebAuthn_FinishLogin_UserNotFound(t *testing.T) {
	svc := newWebAuthnTestService(t)
	r := httptest.NewRequest(http.MethodPost, "/finish", nil)

	if _, err := svc.FinishLogin(context.Background(), NewUserID("ghost"), r); err == nil {
		t.Fatal("expected user-not-found error")
	}
}
