package integration_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
)

const testRemoteAddr1 = "1.2.3.4:1234"

// TestRateLimiterContract verifies the root RateLimiter API that usermgmt
// depends on. This is a contract test: if root changes the RateLimiter or
// RateLimiterConfig API, this test breaks, alerting us to update usermgmt.
func TestRateLimiterContract(t *testing.T) {
	t.Run("Check allows requests within limit and blocks beyond", func(t *testing.T) {
		limiter := cqrshtmx.NewRateLimiter(cqrshtmx.RateLimiterConfig{
			Limit:        2,
			Window:       1 * time.Second,
			KeyExtractor: cqrshtmx.KeyExtractorFromRemoteAddr(),
		})

		r1 := httptest.NewRequest(http.MethodPost, "/register", nil)
		r1.RemoteAddr = testRemoteAddr1
		r2 := httptest.NewRequest(http.MethodPost, "/register", nil)
		r2.RemoteAddr = testRemoteAddr1
		r3 := httptest.NewRequest(http.MethodPost, "/register", nil)
		r3.RemoteAddr = testRemoteAddr1

		if ok, _ := limiter.Check(r1); !ok {
			t.Error("first request should pass")
		}
		if ok, _ := limiter.Check(r2); !ok {
			t.Error("second request should pass")
		}
		if ok, _ := limiter.Check(r3); ok {
			t.Error("third request should be rate limited")
		}
	})

	t.Run("Check uses different keys for different IPs", func(t *testing.T) {
		limiter := cqrshtmx.NewRateLimiter(cqrshtmx.RateLimiterConfig{
			Limit:        1,
			Window:       1 * time.Second,
			KeyExtractor: cqrshtmx.KeyExtractorFromRemoteAddr(),
		})

		r1 := httptest.NewRequest(http.MethodPost, "/register", nil)
		r1.RemoteAddr = testRemoteAddr1
		r2 := httptest.NewRequest(http.MethodPost, "/register", nil)
		r2.RemoteAddr = "5.6.7.8:5678"

		if ok, _ := limiter.Check(r1); !ok {
			t.Error("first IP should pass")
		}
		if ok, _ := limiter.Check(r2); !ok {
			t.Error("second IP should pass")
		}
		r3 := httptest.NewRequest(http.MethodPost, "/register", nil)
		r3.RemoteAddr = testRemoteAddr1
		if ok, _ := limiter.Check(r3); ok {
			t.Error("repeated first IP should be rate limited")
		}
	})
}

// TestRateLimitConfigTypeCompatibility verifies that the RateLimiterConfig
// fields usermgmt relies on are stable.
func TestRateLimitConfigTypeCompatibility(t *testing.T) {
	var _ *cqrshtmx.RateLimiter

	cfg := cqrshtmx.RateLimiterConfig{
		Limit:        5,
		Window:       1 * time.Second,
		KeyExtractor: cqrshtmx.KeyExtractorFromRemoteAddr(),
	}
	if cfg.Limit != 5 {
		t.Errorf("Limit field: got %d, want 5", cfg.Limit)
	}
	if cfg.KeyExtractor == nil {
		t.Error("KeyExtractor should be settable")
	}
}
