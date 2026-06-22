package usermgmt

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func newRateLimitedHandler(t *testing.T) (*Service, *AuthHandler, *http.ServeMux) {
	t.Helper()
	svc := newTestService(t)
	h := NewAuthHandler(svc, HandlerConfig{
		Secure: new(bool),
		RegistrationRateLimit: RateLimitConfig{
			Enabled:     true,
			MaxRequests: 2,
			Window:      time.Minute,
		},
	})
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	return svc, h, mux
}

func postRegister(t *testing.T, mux *http.ServeMux, prefix string, i int) *httptest.ResponseRecorder {
	t.Helper()
	uid := NewUserID(fmt.Sprintf("%s%d", prefix, i)).Get().String()
	body := fmt.Sprintf(`{"id":%q,"email":"%s%d@test.com"}`, uid, prefix, i)
	return postJSON(t, mux, "/auth/register", body)
}

// expectCreatedRegistrations posts `n` registrations with the given id/email
// prefix and asserts each returns StatusCreated.
func expectCreatedRegistrations(t *testing.T, mux *http.ServeMux, prefix string, n int) {
	t.Helper()
	for i := range n {
		w := postRegister(t, mux, prefix, i)
		if w.Code != http.StatusCreated {
			t.Errorf("request %d: status = %d, want %d", i, w.Code, http.StatusCreated)
		}
	}
}

func TestHandleRegister_RateLimitAllows(t *testing.T) {
	_, _, mux := newRateLimitedHandler(t)
	expectCreatedRegistrations(t, mux, "rl", 2)
}

func TestHandleRegister_RateLimitBlocks(t *testing.T) {
	_, _, mux := newRateLimitedHandler(t)

	for i := range 2 {
		w := postRegister(t, mux, "rlb", i)
		if w.Code != http.StatusCreated {
			t.Fatalf("request %d: status = %d, want %d", i, w.Code, http.StatusCreated)
		}
	}

	body := fmt.Sprintf(`{"id":%q,"email":"rlb3@test.com"}`, NewUserID("rlb3").Get().String())
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

	expectCreatedRegistrations(t, mux, "nrl", 10)
}

func TestRegistrationRateLimiter_WindowReset(t *testing.T) {
	rl := newPerIPRateLimiter(2, 50*time.Millisecond)

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
	rl := newPerIPRateLimiter(1, time.Minute)

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
		RegistrationRateLimit: RateLimitConfig{
			Enabled:     true,
			MaxRequests: 1,
			Window:      time.Hour,
		},
	})
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	body1 := fmt.Sprintf(`{"id":%q,"email":"rr1@test.com"}`, NewUserID("rr1").Get().String())
	w1 := postJSON(t, mux, "/auth/register", body1)
	if w1.Code != http.StatusCreated {
		t.Fatalf("first request: status = %d", w1.Code)
	}

	body2 := fmt.Sprintf(`{"id":%q,"email":"rr2@test.com"}`, NewUserID("rr2").Get().String())
	w2 := postJSON(t, mux, "/auth/register", body2)
	if w2.Code != http.StatusTooManyRequests {
		t.Errorf("second request: status = %d, want %d", w2.Code, http.StatusTooManyRequests)
	}
	if !strings.Contains(w2.Body.String(), "too many") {
		t.Errorf("error message should mention rate limit, got: %s", w2.Body.String())
	}
}
