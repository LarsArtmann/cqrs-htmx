package usermgmt

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

func newRateLimitedHandler(t *testing.T) (*Service, *AuthHandler, *http.ServeMux) {
	t.Helper()
	svc := newTestService(t)
	h := NewAuthHandler(svc, HandlerConfig{
		Secure: new(bool),
		RegistrationRateLimit: RegistrationRateLimitConfig{
			Enabled:     true,
			MaxRequests: 2,
			Window:      time.Minute,
		},
	})
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	return svc, h, mux
}

func TestHandleRegister_RateLimitAllows(t *testing.T) {
	_, _, mux := newRateLimitedHandler(t)

	for i := range 2 {
		body := fmt.Sprintf(`{"id":"rl%d","email":"rl%d@test.com"}`, i, i)
		w := postJSON(t, mux, "/auth/register", body)
		if w.Code != http.StatusCreated {
			t.Errorf("request %d: status = %d, want %d", i, w.Code, http.StatusCreated)
		}
	}
}

func TestHandleRegister_RateLimitBlocks(t *testing.T) {
	_, _, mux := newRateLimitedHandler(t)

	for i := range 2 {
		body := fmt.Sprintf(`{"id":"rlb%d","email":"rlb%d@test.com"}`, i, i)
		w := postJSON(t, mux, "/auth/register", body)
		if w.Code != http.StatusCreated {
			t.Fatalf("request %d: status = %d, want %d", i, w.Code, http.StatusCreated)
		}
	}

	body := `{"id":"rlb3","email":"rlb3@test.com"}`
	w := postJSON(t, mux, "/auth/register", body)
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("third request: status = %d, want %d", w.Code, http.StatusTooManyRequests)
	}
}

func TestHandleRegister_RateLimitNoOpWhenDisabled(t *testing.T) {
	svc := newTestService(t)
	h := NewAuthHandler(svc, HandlerConfig{Secure: new(bool)})
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	for i := range 10 {
		body := fmt.Sprintf(`{"id":"nrl%d","email":"nrl%d@test.com"}`, i, i)
		w := postJSON(t, mux, "/auth/register", body)
		if w.Code != http.StatusCreated {
			t.Errorf("request %d: status = %d, want %d", i, w.Code, http.StatusCreated)
		}
	}
}

func TestRegistrationRateLimiter_WindowReset(t *testing.T) {
	rl := newRegistrationRateLimiter(2, 50*time.Millisecond)

	if !rl.allow("ip1") {
		t.Error("first request should be allowed")
	}
	if !rl.allow("ip1") {
		t.Error("second request should be allowed")
	}
	if rl.allow("ip1") {
		t.Error("third request should be blocked")
	}

	time.Sleep(60 * time.Millisecond)

	if !rl.allow("ip1") {
		t.Error("request after window reset should be allowed")
	}
}

func TestRegistrationRateLimiter_DifferentIPs(t *testing.T) {
	rl := newRegistrationRateLimiter(1, time.Minute)

	if !rl.allow("ip1") {
		t.Error("first ip1 request should be allowed")
	}
	if rl.allow("ip1") {
		t.Error("second ip1 request should be blocked")
	}
	if !rl.allow("ip2") {
		t.Error("first ip2 request should be allowed")
	}
}

func TestRegistrationRateLimiter_RapidRequests(t *testing.T) {
	svc := newTestService(t)
	h := NewAuthHandler(svc, HandlerConfig{
		Secure: new(bool),
		RegistrationRateLimit: RegistrationRateLimitConfig{
			Enabled:     true,
			MaxRequests: 1,
			Window:      time.Hour,
		},
	})
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	body1 := `{"id":"rr1","email":"rr1@test.com"}`
	w1 := postJSON(t, mux, "/auth/register", body1)
	if w1.Code != http.StatusCreated {
		t.Fatalf("first request: status = %d", w1.Code)
	}

	body2 := `{"id":"rr2","email":"rr2@test.com"}`
	w2 := postJSON(t, mux, "/auth/register", body2)
	if w2.Code != http.StatusTooManyRequests {
		t.Errorf("second request: status = %d, want %d", w2.Code, http.StatusTooManyRequests)
	}
	if !strings.Contains(w2.Body.String(), "too many") {
		t.Errorf("error message should mention rate limit, got: %s", w2.Body.String())
	}
}
