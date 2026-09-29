package setup_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/larsartmann/cqrs-htmx/setup/v4"
)

// PapDashboard feedback item 2 (2026-09-29): consumer-owned middleware chain.
// Config.ExtraMiddleware composes inside the built-in security stack;
// Config.DisableSecurityMiddleware hands the whole outer chain to the
// consumer. The pins below hold the defined position and the zero-value
// no-op contract.

// TestMiddleware_ExtraMiddlewareRunsInsideSecurityStack asserts the defined
// position: extras apply (their effects reach the response) while the
// built-in security layer still wraps them (its headers also reach the
// response) — one chain, both layers visible.
func TestMiddleware_ExtraMiddlewareRunsInsideSecurityStack(t *testing.T) {
	t.Parallel()

	bundle := setup.MustNew(setup.Config{
		Title: "Extra MW",
		ExtraMiddleware: []func(http.Handler) http.Handler{
			func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("X-Extra-Layer", "applied")
					next.ServeHTTP(w, r)
				})
			},
		},
	})
	defer func() { _ = bundle.Close() }()

	mux := http.NewServeMux()
	server := httptest.NewServer(bundle.Handler(mux))
	defer server.Close()

	resp, err := server.Client().Get(server.URL + "/health")
	if err != nil {
		t.Fatalf("GET /health: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, resp.Body)

	if got := resp.Header.Get("X-Extra-Layer"); got != "applied" {
		t.Errorf("ExtraMiddleware must apply to every response, got X-Extra-Layer=%q", got)
	}

	if got := resp.Header.Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf(
			"built-in security layer must still wrap the extras, got X-Content-Type-Options=%q",
			got,
		)
	}
}

// TestMiddleware_ExtraMiddlewareOrderPinsListedOrder drives two recording
// extras and asserts they run in listed order, first entry outermost — the
// same convention cqrshtmx.Chain documents.
func TestMiddleware_ExtraMiddlewareOrderPinsListedOrder(t *testing.T) {
	t.Parallel()

	var (
		mu      sync.Mutex
		entered []string
	)

	record := func(name string) func(http.Handler) http.Handler {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				entered = append(entered, name)
				mu.Unlock()

				next.ServeHTTP(w, r)
			})
		}
	}

	bundle := setup.MustNew(setup.Config{
		Title: "Order",
		ExtraMiddleware: []func(http.Handler) http.Handler{
			record("first"),
			record("second"),
		},
	})
	defer func() { _ = bundle.Close() }()

	mux := http.NewServeMux()
	server := httptest.NewServer(bundle.Handler(mux))
	defer server.Close()

	resp, err := server.Client().Get(server.URL + "/health")
	if err != nil {
		t.Fatalf("GET /health: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, resp.Body)

	if strings.Join(entered, ",") != "first,second" {
		t.Errorf("extras must run in listed order (first outermost), got %v", entered)
	}
}

// TestMiddleware_DisableSecurityMiddlewareRemovesBuiltInLayer pins the
// ownership escape hatch: with the flag set, the built-in security headers
// disappear while the extras (and routing) keep working.
func TestMiddleware_DisableSecurityMiddlewareRemovesBuiltInLayer(t *testing.T) {
	t.Parallel()

	bundle := setup.MustNew(setup.Config{
		Title:                     "Own Chain",
		DisableSecurityMiddleware: true,
		ExtraMiddleware: []func(http.Handler) http.Handler{
			func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("X-Owned-Chain", "yes")
					next.ServeHTTP(w, r)
				})
			},
		},
	})
	defer func() { _ = bundle.Close() }()

	mux := http.NewServeMux()
	server := httptest.NewServer(bundle.Handler(mux))
	defer server.Close()

	resp, err := server.Client().Get(server.URL + "/health")
	if err != nil {
		t.Fatalf("GET /health: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("/health must still answer 200, got %d", resp.StatusCode)
	}

	if got := resp.Header.Get("X-Content-Type-Options"); got != "" {
		t.Errorf("DisableSecurityMiddleware must remove built-in security headers, got %q", got)
	}

	if got := resp.Header.Get("X-Owned-Chain"); got != "yes" {
		t.Errorf("extras must survive the disable flag, got X-Owned-Chain=%q", got)
	}
}

// TestMiddleware_NoExtrasMatchesLegacyChain pins the zero value: without
// extras, the chain still carries the built-in security layer — the seam is
// invisible unless configured.
func TestMiddleware_NoExtrasMatchesLegacyChain(t *testing.T) {
	t.Parallel()

	bundle := setup.MustNew(setup.Config{Title: "Legacy"})
	defer func() { _ = bundle.Close() }()

	mux := http.NewServeMux()
	server := httptest.NewServer(bundle.Handler(mux))
	defer server.Close()

	resp, err := server.Client().Get(server.URL + "/health")
	if err != nil {
		t.Fatalf("GET /health: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, resp.Body)

	if got := resp.Header.Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("default chain must keep security headers, got %q", got)
	}
}
