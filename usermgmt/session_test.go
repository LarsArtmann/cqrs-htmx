package usermgmt

import (
	"testing"
	"time"
)

func TestSession(t *testing.T) {
	session, err := NewSession(NewUserID("user-1"), time.Hour)
	if err != nil {
		t.Fatalf("NewSession failed: %v", err)
	}
	if session.UserID != NewUserID("user-1") {
		t.Errorf("expected UserID user-1, got %s", session.UserID)
	}
	if session.IsExpired() {
		t.Error("expected session not to be expired")
	}
	if !session.Valid(session.Token) {
		t.Error("expected session to be valid with correct token")
	}
	if session.Valid("wrong-token") {
		t.Error("expected session to be invalid with wrong token")
	}
	if !session.TokenMatches(session.Token) {
		t.Error("expected TokenMatches to return true for correct token")
	}
	if session.TokenMatches("wrong-token") {
		t.Error("expected TokenMatches to return false for wrong token")
	}
}
