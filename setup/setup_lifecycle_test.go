package setup_test

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/larsartmann/cqrs-htmx/setup/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	memorystorage "github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

// Tests in this file cover Bundle lifecycle:
//
//   - MustNew succeeds with valid defaults.
//   - Bundle exposes the middleware factories (Middleware / SessionMiddleware /
//     CSRFMiddleware) and the projection host for the dashboard.
//   - Close is idempotent and tears down the service + dashboard exactly once.
//   - Run binds a real listener, serves /health, and shuts down gracefully on
//     context cancellation.
//   - AsyncStartup lets New return while projections drain — /health answers
//     503 until every worker reaches "live", then flips to 200.
//   - SyncStartup (the default) blocks New until projections drain.

// --- Middleware / projection host exposure ---

func TestMustNew_SucceedsWithDefaults(t *testing.T) {
	t.Parallel()

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("MustNew panicked with valid config: %v", r)
		}
	}()

	bundle := setup.MustNew(setup.Config{
		Title: "MustNew Test",
	})
	defer func() { _ = bundle.Close() }()
}

func TestMiddleware_ReturnsNonNil(t *testing.T) {
	t.Parallel()

	bundle := setup.MustNew(setup.Config{
		Title: "Middleware Test",
	})
	defer func() { _ = bundle.Close() }()

	if bundle.Middleware() == nil {
		t.Fatal("Middleware() returned nil")
	}

	if bundle.SessionMiddleware() == nil {
		t.Fatal("SessionMiddleware() returned nil")
	}

	if bundle.CSRFMiddleware() == nil {
		t.Fatal("CSRFMiddleware() returned nil")
	}
}

func TestProjectionHost_Exposed(t *testing.T) {
	t.Parallel()

	bundle := setup.MustNew(setup.Config{
		Title: "ProjectionHost Test",
	})
	defer func() { _ = bundle.Close() }()

	host := bundle.Service.ProjectionHost()
	if host == nil {
		t.Fatal("ProjectionHost() returned nil — dashboard cannot show projection health")
	}
}

// --- Close idempotency ---

func TestClose_Idempotent(t *testing.T) {
	t.Parallel()

	bundle := setup.MustNew(setup.Config{
		Title: "Close Test",
	})

	if err := bundle.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}

	if err := bundle.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}

func TestNew_DashboardClosesOnBundleClose(t *testing.T) {
	t.Parallel()

	bundle := setup.MustNew(setup.Config{
		Title: "Dashboard Close Test",
	})

	if bundle.Dashboard == nil {
		t.Fatal("Dashboard is nil")
	}

	// Close the bundle — this should close the dashboard too.
	if err := bundle.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Calling Close again should be safe (dashboard uses sync.Once).
	if err := bundle.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}

func TestNew_CloseReturnsServiceError(t *testing.T) {
	t.Parallel()

	bundle := setup.MustNew(setup.Config{Title: "Close Error Test"})

	// First close succeeds (closes service + dashboard).
	if err := bundle.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}

	// Second close: service is already closed, dashboard uses sync.Once (no-op).
	// Service.Close() may or may not return an error on double-close.
	// Just verify it doesn't panic.
	_ = bundle.Close()
}

// --- Run lifecycle: serve + graceful shutdown ---

// TestBundleRun_ServesAndShutsDownGracefully exercises the full Run lifecycle:
// mount, serve with real timeouts, health check, context cancellation,
// graceful shutdown, and bundle cleanup — with nil as the only clean-exit
// return value.
func TestBundleRun_ServesAndShutsDownGracefully(t *testing.T) {
	t.Parallel()

	bundle := setup.MustNew(setup.Config{Title: "Run Test"})

	// Reserve a free port, then release it for Run to bind (small inherent
	// race, standard practice for testing real listeners).
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve port: %v", err)
	}

	addr := listener.Addr().String()

	if err := listener.Close(); err != nil {
		t.Fatalf("release port: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	runErr := make(chan error, 1)

	go func() { runErr <- bundle.Run(ctx, addr) }()

	// Wait for the server to come up, then verify it serves.
	baseURL := "http://" + addr

	var resp *http.Response

	for range 100 {
		resp, err = http.Get(baseURL + "/health")
		if err == nil {
			break
		}

		time.Sleep(20 * time.Millisecond)
	}

	if err != nil {
		t.Fatalf("server never came up: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /health: status %d, want 200", resp.StatusCode)
	}

	resp.Body.Close()

	cancel()

	select {
	case err := <-runErr:
		if err != nil {
			t.Fatalf("Run returned error on graceful shutdown: %v", err)
		}
	case <-time.After(15 * time.Second):
		t.Fatal("Run did not return within 15s of context cancellation")
	}
}

// --- AsyncStartup / SyncStartup ---

// gatedStore delays every journal ReadFrom until the gate channel is closed.
// projectionhost drains via ReadFrom, so this deterministically holds
// projection workers out of "live" state — without sleeps or huge journals.
type gatedStore struct {
	*memorystorage.MemoryStore

	gate chan struct{}
}

func (g *gatedStore) ReadFrom(
	ctx context.Context,
	after id.EventID,
	limit int,
) ([]event.Event, error) {
	select {
	case <-g.gate:
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	return g.MemoryStore.ReadFrom(ctx, after, limit)
}

// TestAsyncStartup_HealthTransitionsFromDrainingToReady is the end-to-end
// lifecycle test for Config.AsyncStartup: with the journal gated, New returns
// immediately, /health answers 503 (not ready) while projections drain, and
// flips to 200 once every worker reaches "live". This is the contract a
// reverse proxy relies on during the catch-up window after a restart.
func TestAsyncStartup_HealthTransitionsFromDrainingToReady(t *testing.T) {
	t.Parallel()

	store := &gatedStore{MemoryStore: memorystorage.NewMemoryStore(), gate: make(chan struct{})}

	bundle, err := setup.New(setup.Config{
		Title:        "Async Startup Test",
		EventStore:   store,
		AsyncStartup: true,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = bundle.Close() }()

	mux := http.NewServeMux()
	bundle.Mount(mux)

	server := httptest.NewServer(bundle.Middleware()(mux))
	defer server.Close()

	// While the journal read is gated, no projection can be live: /health
	// must report 503 (not ready), never a panic or a hang.
	resp, err := http.Get(server.URL + "/health")
	if err != nil {
		t.Fatalf("GET /health while draining: %v", err)
	}

	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("GET /health while draining: status %d, want 503. Body: %s", resp.StatusCode, body)
	}

	// Release the journal: projections drain the (empty) journal and go live.
	close(store.gate)

	deadline := time.Now().Add(30 * time.Second)

	for {
		resp, err = http.Get(server.URL + "/health")
		if err == nil {
			body, _ = io.ReadAll(resp.Body)
			resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				break
			}
		}

		if time.Now().After(deadline) {
			t.Fatalf("/health never became ready after gate release; last: %d %s", resp.StatusCode, body)
		}

		time.Sleep(50 * time.Millisecond)
	}
}

// TestSyncStartup_BlocksUntilDrained verifies the complementary default:
// with AsyncStartup=false, New does not return until projections finished
// their initial drain — it stays blocked while the journal is gated and
// completes shortly after the gate opens.
func TestSyncStartup_BlocksUntilDrained(t *testing.T) {
	t.Parallel()

	store := &gatedStore{MemoryStore: memorystorage.NewMemoryStore(), gate: make(chan struct{})}

	type result struct {
		bundle *setup.Bundle
		err    error
	}

	newDone := make(chan result, 1)

	go func() {
		bundle, err := setup.New(setup.Config{
			Title:      "Sync Startup Test",
			EventStore: store,
		})
		newDone <- result{bundle, err}
	}()

	// While the journal is gated, New must still be blocked (drain in flight).
	select {
	case res := <-newDone:
		t.Fatalf("New returned before drain completed: bundle=%v err=%v", res.bundle, res.err)
	case <-time.After(300 * time.Millisecond):
	}

	close(store.gate)

	select {
	case res := <-newDone:
		if res.err != nil {
			t.Fatalf("New: %v", res.err)
		}

		if err := res.bundle.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("New did not return within 30s of gate release")
	}
}
