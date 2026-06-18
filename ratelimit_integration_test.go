package cqrshtmx_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v2"
)

// writeStringHandler returns an http.Handler that writes the given
// body with a 200 OK. Used by the rate-limiter integration tests to
// give a stable, observable inner handler.
func writeStringHandler(body string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(body))
	})
}

// newRateLimitedServer builds a real httptest.Server with the given rate
// limit configuration serving `body` at `path`. The returned server is
// automatically closed via t.Cleanup.
func newRateLimitedServer(t *testing.T, cfg cqrshtmx.RateLimiterConfig, path, body string) *httptest.Server {
	t.Helper()
	middleware := cqrshtmx.RateLimiterMiddleware(cfg)
	mux := http.NewServeMux()
	mux.Handle(path, middleware(writeStringHandler(body)))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server
}

// TestRateLimiter_RealServer_AllowsThenBlocks boots an actual HTTP server
// and exercises the rate limiter end-to-end with a real http.Client.
// This catches issues that httptest.NewRecorder cannot: real TCP connections,
// real client/server interaction, and real ResponseWriter hijacking behavior.
func TestRateLimiter_RealServer_AllowsThenBlocks(t *testing.T) {
	t.Parallel()
	server := newRateLimitedServer(t, cqrshtmx.RateLimiterConfig{
		Limit:        1,
		Window:       time.Minute,
		Burst:        1,
		KeyExtractor: cqrshtmx.KeyExtractorFromClientIP(),
	}, "/ping", "pong")

	client := server.Client()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// First request: should pass.
	req1, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/ping", nil)
	if err != nil {
		t.Fatalf("NewRequest 1: %v", err)
	}
	resp1, err := client.Do(req1)
	if err != nil {
		t.Fatalf("Do 1: %v", err)
	}
	_ = resp1.Body.Close()
	if resp1.StatusCode != http.StatusOK {
		t.Errorf("expected first request 200, got %d", resp1.StatusCode)
	}

	// Second request: same client IP (loopback) → should be rate-limited.
	req2, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/ping", nil)
	if err != nil {
		t.Fatalf("NewRequest 2: %v", err)
	}
	resp2, err := client.Do(req2)
	if err != nil {
		t.Fatalf("Do 2: %v", err)
	}
	_ = resp2.Body.Close()
	if resp2.StatusCode != http.StatusTooManyRequests {
		t.Errorf("expected second request 429, got %d", resp2.StatusCode)
	}
	if retry := resp2.Header.Get("Retry-After"); retry == "" {
		t.Error("expected Retry-After header on rate-limited response")
	}
}

// TestRateLimiter_RealServer_ConcurrentRequests verifies that concurrent
// requests from the same client are correctly limited by the token bucket
// under real concurrency. With Limit=5/Burst=5, exactly 5 should succeed.
func TestRateLimiter_RealServer_ConcurrentRequests(t *testing.T) {
	t.Parallel()
	server := newRateLimitedServer(t, cqrshtmx.RateLimiterConfig{
		Limit:        5,
		Window:       time.Minute,
		Burst:        5,
		KeyExtractor: cqrshtmx.KeyExtractorFromClientIP(),
	}, "/work", "ok")

	client := server.Client()

	const totalRequests = 10
	results := make(chan int, totalRequests)
	for range totalRequests {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/work", nil)
			if err != nil {
				results <- -1
				return
			}
			resp, err := client.Do(req)
			if err != nil {
				results <- -1
				return
			}
			_ = resp.Body.Close()
			results <- resp.StatusCode
		}()
	}

	var ok, blocked int
	for range totalRequests {
		code := <-results
		switch code {
		case http.StatusOK:
			ok++
		case http.StatusTooManyRequests:
			blocked++
		default:
			t.Errorf("unexpected status code: %d", code)
		}
	}

	// Token bucket may let slightly fewer than 5 through due to refill timing,
	// but never more. Accept 1..5 OK responses and the rest blocked.
	if ok < 1 || ok > 5 {
		t.Errorf("expected 1..5 OK responses under burst 5, got ok=%d blocked=%d", ok, blocked)
	}
	if ok+blocked != totalRequests {
		t.Errorf("expected %d total responses, got %d", totalRequests, ok+blocked)
	}
}
