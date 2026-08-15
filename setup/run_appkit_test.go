// cqrs-lint:ignore(E009) spike test file (ADR-001 P3 / M18): verifies the appkit server swap
package setup

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"testing"
	"time"
)

// freeLocalAddr reserves an OS-assigned port, releases it, and returns the
// address. Small TOCTOU window, standard for spike tests.
func freeLocalAddr(tb testing.TB) string {
	tb.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		tb.Fatalf("reserve port: %v", err)
	}

	addr := ln.Addr().String()
	_ = ln.Close()

	return addr
}

// waitServe polls until the address accepts TCP connections.
func waitServe(tb testing.TB, addr string) {
	tb.Helper()

	deadline := time.Now().Add(10 * time.Second)

	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 500*time.Millisecond)
		if err == nil {
			_ = conn.Close()

			return
		}

		time.Sleep(25 * time.Millisecond)
	}

	tb.Fatalf("server at %s never started listening", addr)
}

// serveInBackground runs fn (RunHandler or RunWithAppkit) and returns a done
// channel plus a stop function that cancels the context and asserts clean exit.
func serveInBackground(
	tb testing.TB,
	run func(context.Context, string, http.Handler) error,
	addr string,
	h http.Handler,
) (stop func()) {
	tb.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)

	go func() { done <- run(ctx, addr, h) }()

	waitServe(tb, addr)

	return func() {
		cancel()

		select {
		case err := <-done:
			if err != nil {
				tb.Fatalf("server returned error: %v", err)
			}
		case <-time.After(20 * time.Second):
			tb.Fatal("server did not shut down")
		}
	}
}

// TestRunWithAppkit_SSEHeaderFlushThroughFullStack (M18.3): an SSE-style
// handler — headers written and flushed immediately, first event 400ms later —
// must have its flush survive BOTH middleware layers (appkit's generic stack
// outside, the bundle's chain inside). The client must observe headers well
// before the first event is due, and then the event itself, intact.
func TestRunWithAppkit_SSEHeaderFlushThroughFullStack(t *testing.T) {
	bundle := MustNew(Config{Title: "sse-flush-spike"})
	defer func() { _ = bundle.Close() }()

	const eventDelay = 400 * time.Millisecond

	sse := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.WriteHeader(http.StatusOK)

		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}

		time.Sleep(eventDelay)
		_, _ = fmt.Fprint(w, "event: ping\ndata: {}\n\n")

		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	})

	addr := freeLocalAddr(t)
	stop := serveInBackground(t, bundle.RunWithAppkit, addr, sse)
	defer stop()

	client := &http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequestWithContext(
		context.Background(), http.MethodGet, "http://"+addr+"/", nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}

	start := time.Now()

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("get: %v", err)
	}

	headerElapsed := time.Since(start)

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("content-type = %q, want text/event-stream", ct)
	}

	if headerElapsed >= eventDelay {
		t.Fatalf("headers after %v, expected flushed well before first event at %v",
			headerElapsed, eventDelay)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	if string(body) != "event: ping\ndata: {}\n\n" {
		t.Fatalf("body = %q, want the SSE event intact", body)
	}
}

// TestRunWithAppkit_ReadinessAndCleanShutdown (M18.2): /health/ready is served
// by appkit with the projection-aware check wired in, and a cancelled context
// drains and closes the bundle cleanly.
func TestRunWithAppkit_ReadinessAndCleanShutdown(t *testing.T) {
	bundle := MustNew(Config{Title: "readiness-spike"})
	defer func() { _ = bundle.Close() }()

	addr := freeLocalAddr(t)
	stop := serveInBackground(t, bundle.RunWithAppkit, addr, nil)
	defer stop()

	client := &http.Client{Timeout: 2 * time.Second}
	ctx := context.Background()

	deadline := time.Now().Add(10 * time.Second)

	for {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+addr+"/health/ready", nil)
		if err != nil {
			t.Fatalf("build request: %v", err)
		}

		resp, err := client.Do(req)

		if err == nil {
			sc := resp.StatusCode
			_ = resp.Body.Close()

			if sc == http.StatusOK {
				break
			}
		}

		if time.Now().After(deadline) {
			t.Fatal("readiness never became 200 (projections never live?)")
		}

		time.Sleep(50 * time.Millisecond)
	}
}

// TestRunWithAppkit_ResponseParity: an ordinary handler's response passes
// through the appkit stack unchanged.
func TestRunWithAppkit_ResponseParity(t *testing.T) {
	bundle := MustNew(Config{Title: "parity-spike"})
	defer func() { _ = bundle.Close() }()

	hello := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)

		_, _ = io.WriteString(w, "parity-ok")
	})

	addr := freeLocalAddr(t)
	stop := serveInBackground(t, bundle.RunWithAppkit, addr, hello)
	defer stop()

	resp, err := http.Get("http://" + addr + "/hello")
	if err != nil {
		t.Fatalf("get: %v", err)
	}

	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	if resp.StatusCode != http.StatusOK || string(body) != "parity-ok" {
		t.Fatalf("status = %d body = %q, want 200 parity-ok", resp.StatusCode, body)
	}
}

// BenchmarkSpikeBaselineVsAppkit (M18.4): smoke comparison of request-path
// overhead — the same handler behind RunHandler (httputil.Server) vs
// RunWithAppkit (appkit.Service: extra middleware + readiness probe).
// The 2s drain delay only affects appkit shutdown (outside the timer).
//
//	go test -bench BenchmarkSpikeBaselineVsAppkit -benchtime 2s -run xxx
func BenchmarkSpikeBaselineVsAppkit(b *testing.B) {
	ping := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	cases := []struct {
		name string
		run  func(*Bundle, context.Context, string, http.Handler) error
	}{
		{"baseline-httputil", (*Bundle).RunHandler},
		{"appkit-service", (*Bundle).RunWithAppkit},
	}

	for _, bc := range cases {
		b.Run(bc.name, func(b *testing.B) {
			bundle := MustNew(Config{Title: "bench-" + bc.name})

			addr := freeLocalAddr(b)
			stop := serveInBackground(b, func(ctx context.Context, addr string, h http.Handler) error {
				return bc.run(bundle, ctx, addr, h)
			}, addr, ping)

			// The run funcs close the bundle on every exit path; stop() waits
			// for that drain after the measured loop finishes. The deferred
			// form stops the timer first so the 2s drain never counts toward
			// ns/op (it would poison b.N scaling at one iteration).
			defer func() {
				b.StopTimer()
				stop()
			}()

			client := &http.Client{Timeout: 5 * time.Second}

			req, err := http.NewRequestWithContext(
				context.Background(), http.MethodGet, "http://"+addr+"/", nil)
			if err != nil {
				b.Fatalf("build request: %v", err)
			}

			b.ResetTimer()

			for i := range b.N {
				resp, err := client.Do(req)
				if err != nil {
					b.Fatalf("request %d: %v", i, err)
				}

				_, _ = io.Copy(io.Discard, resp.Body)
				_ = resp.Body.Close()
			}
		})
	}
}
