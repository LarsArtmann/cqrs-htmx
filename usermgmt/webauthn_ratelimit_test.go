package usermgmt

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestHandlers_WebAuthnRateLimit(t *testing.T) {
	svc := newWebAuthnTestService(t)
	reg := registerTestUser(t, svc, "wa-rl", "warl@test.com")
	sess, err := svc.createSession(context.Background(), reg.User.ID)
	if err != nil {
		t.Fatalf("Create session: %v", err)
	}

	h := NewAuthHandler(svc, HandlerConfig{
		Secure:            new(bool),
		WebAuthnRateLimit: RateLimitConfig{Enabled: true, MaxRequests: 1, Window: time.Minute},
	})
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	wrapped := NewSessionMiddleware(svc, "session_token")(mux)

	// First begin-registration request is allowed.
	body := `{"user_id":"` + reg.User.ID.Get().String() + `"}`
	w1 := authenticatedRequest(t, wrapped, http.MethodPost, "/auth/webauthn/register/begin", sess.Token, body)
	if w1.Code == http.StatusTooManyRequests {
		t.Fatalf("first request should not be rate-limited, got %d", w1.Code)
	}

	// Second request is blocked by the per-IP limiter.
	w2 := authenticatedRequest(t, wrapped, http.MethodPost, "/auth/webauthn/register/begin", sess.Token, body)
	assertStatusCode(t, w2, http.StatusTooManyRequests)
}
