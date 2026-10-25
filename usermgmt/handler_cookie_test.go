package usermgmt

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestAuthHandler_SessionCookieMaxAge pins S37c: the session cookie's
// Max-Age defaults to the default session TTL (24h) and an explicit
// HandlerConfig.SessionMaxAge wins. A cookie that outlives the session
// would send expired tokens on every request; one that dies early would
// log users out mid-session.
func TestAuthHandler_SessionCookieMaxAge(t *testing.T) {
	svc := setupTestService(t)
	t.Cleanup(svc.Stop)

	defaultHandler := NewAuthHandler(svc)
	rec := httptest.NewRecorder()
	defaultHandler.setSessionCookie(rec, "token-1")

	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected 1 session cookie, got %d", len(cookies))
	}
	if want := int(defaultSessionTTL.Seconds()); cookies[0].MaxAge != want {
		t.Errorf("default Max-Age = %d, want %d (defaultSessionTTL)", cookies[0].MaxAge, want)
	}

	customHandler := NewAuthHandler(svc, HandlerConfig{SessionMaxAge: 3600})
	rec = httptest.NewRecorder()
	customHandler.setSessionCookie(rec, "token-2")

	cookies = rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected 1 session cookie, got %d", len(cookies))
	}
	if cookies[0].MaxAge != 3600 {
		t.Errorf("explicit SessionMaxAge ignored: Max-Age = %d, want 3600", cookies[0].MaxAge)
	}
}
