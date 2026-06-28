package usermgmt

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
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

// postWithRemoteAddr sends a POST request with a specific RemoteAddr,
// which is what the rate limiter keys on.
func postWithRemoteAddr(mux *http.ServeMux, path, body, remoteAddr string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.RemoteAddr = remoteAddr
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	return w
}

// rlRegBody builds a registration JSON body with a unique user ID.
func rlRegBody(email string) string {
	uid := NewUserID("rl-" + strconv.FormatInt(time.Now().UnixNano(), 10)).Get().String()
	return fmt.Sprintf(`{"id":%q,"email":%q}`, uid, email)
}

func TestHandleRegister_RateLimitAllows(t *testing.T) {
	_, _, mux := newRateLimitedHandler(t)
	expectCreatedRegistrations(t, mux, "rl", 2)
}

func TestRegistrationRateLimiter_Disabled(t *testing.T) {
	svc := newTestService(t)
	h := NewAuthHandler(svc, HandlerConfig{
		Secure: new(bool),
	})
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	// No rate limit configured — all requests should succeed.
	expectCreatedRegistrations(t, mux, "nrl", 10)
}

func TestRegistrationRateLimiter_DifferentIPs(t *testing.T) {
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

	// First request from ip1 — should succeed.
	body1 := rlRegBody("ip1rat@test.com")
	w := postWithRemoteAddr(mux, "/auth/register", body1, "1.2.3.4:1234")
	if w.Code != http.StatusCreated {
		t.Errorf("first ip1 request: expected %d, got %d", http.StatusCreated, w.Code)
	}

	// Second request from ip1 — should be rate-limited.
	body2 := rlRegBody("ip1rat2@test.com")
	w = postWithRemoteAddr(mux, "/auth/register", body2, "1.2.3.4:1234")
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("second ip1 request: expected %d, got %d", http.StatusTooManyRequests, w.Code)
	}

	// First request from ip2 — should succeed (different IP).
	body3 := rlRegBody("ip2rat@test.com")
	w = postWithRemoteAddr(mux, "/auth/register", body3, "5.6.7.8:5678")
	if w.Code != http.StatusCreated {
		t.Errorf("first ip2 request: expected %d, got %d", http.StatusCreated, w.Code)
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

	// First request should succeed.
	body1 := rlRegBody("rapid1@test.com")
	w := postWithRemoteAddr(mux, "/auth/register", body1, "10.0.0.1:1234")
	if w.Code != http.StatusCreated {
		t.Fatalf("first request: expected %d, got %d", http.StatusCreated, w.Code)
	}

	// Second rapid request should be blocked.
	body2 := rlRegBody("rapid2@test.com")
	w = postWithRemoteAddr(mux, "/auth/register", body2, "10.0.0.1:1234")
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("second request: expected %d, got %d", http.StatusTooManyRequests, w.Code)
	}
}
