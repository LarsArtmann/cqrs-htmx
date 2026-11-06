// RunWithAppkit lifecycle-hook forwarding tests (A22/B93): Config's three
// Appkit*Hooks fields must reach the appkit service, run in the documented
// lifecycle order (start -> drain -> shutdown), and the shutdown hooks must
// fire BEFORE the bundle's own Close tears down the SSE hub — the
// telemetry-flush-while-stores-live contract the config docs promise.
package setup

import (
	"context"
	"net/http"
	"slices"
	"sync"
	"testing"
	"time"

	appkit "github.com/larsartmann/go-appkit"
)

// hookRecorder collects lifecycle-hook firings behind a mutex so the race
// detector can observe hooks running on appkit's goroutines while the test
// goroutine reads the sequence.
type hookRecorder struct {
	mu     sync.Mutex
	events []string
}

// hook returns an appkit.Hook that records event when it fires.
func (r *hookRecorder) hook(event string) appkit.Hook {
	return func(context.Context) error {
		r.mu.Lock()
		defer r.mu.Unlock()

		r.events = append(r.events, event)

		return nil
	}
}

func (r *hookRecorder) snapshot() []string {
	r.mu.Lock()
	defer r.mu.Unlock()

	return append([]string(nil), r.events...)
}

func TestRunWithAppkit_HookForwardingAndLifecycleOrder(t *testing.T) {
	rec := &hookRecorder{}

	bundle := MustNew(Config{
		Title: "appkit-hooks-adoption",
		// SSEPath wires Bundle.Broadcaster so the test can observe Close's
		// hub teardown: hub Close closes subscriber channels, and the shutdown
		// hook must fire while the hub is still open.
		SSEPath: "/events",
		AppkitStartHooks: []appkit.Hook{
			rec.hook("start:1"),
			rec.hook("start:2"),
		},
		AppkitDrainHooks:    []appkit.Hook{rec.hook("drain")},
		AppkitShutdownHooks: []appkit.Hook{rec.hook("shutdown")},
	})

	hubSub := bundle.Broadcaster.Subscribe()
	defer func() { _ = bundle.Close() }()

	var hubClosedBeforeShutdownHook bool
	bundle.config.AppkitShutdownHooks = append(bundle.config.AppkitShutdownHooks, func(context.Context) error {
		// Non-blocking probe: a closed hub closes its subscriber channels, so
		// receiving !ok here would mean Bundle.Close ran before the shutdown
		// hooks — the ordering the config docs promise against.
		select {
		case _, ok := <-hubSub:
			hubClosedBeforeShutdownHook = !ok
		default:
		}

		return nil
	})

	addr := freeLocalAddr(t)
	realStop := serveInBackground(t, func(ctx context.Context, addr string, h http.Handler) error {
		return bundle.runWithAppkit(ctx, addr, h, testDrainDelay, "")
	}, addr, nil)
	// Once-guarded so the explicit stop() below (needed before the post-shutdown
	// asserts) and the leak-safety defer cannot both consume serveInBackground's
	// one-shot done channel.
	var stopOnce sync.Once
	stop := func() { stopOnce.Do(realStop) }
	defer stop()

	// appkit serves /health/live before any bundle route; a 200 response
	// proves serving began, which by appkit's contract means both start hooks
	// already ran (post-listen, pre-serve).
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get("http://" + addr + "/health/live")
	if err != nil {
		t.Fatalf("health/live request: %v", err)
	}
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("health/live status = %d, want 200", resp.StatusCode)
	}

	if got := rec.snapshot(); len(got) != 2 || got[0] != "start:1" || got[1] != "start:2" {
		t.Fatalf("start hooks after first response = %v, want [start:1 start:2] (post-listen, pre-serve)", got)
	}

	stop()

	want := []string{"start:1", "start:2", "drain", "shutdown"}
	if got := rec.snapshot(); !slices.Equal(got, want) {
		t.Fatalf("lifecycle hook order = %v, want %v", got, want)
	}

	if hubClosedBeforeShutdownHook {
		t.Fatal("SSE hub was closed before the shutdown hooks ran; shutdown hooks must precede Bundle.Close")
	}
}
