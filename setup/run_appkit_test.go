// cqrs-lint:ignore(E009) spike test file (ADR-001 P3 / M18): verifies the appkit server swap
package setup

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"testing"
	"time"

	appkit "github.com/larsartmann/go-appkit"
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

// spikeTestDrainDelay replaces the 2-second appkit production drain with a
// value small enough that the spike test suite finishes in ~3 seconds instead
// of ~6 — production behavior (drain → readiness flip → stop) is still
// exercised end-to-end, just on a faster clock.
const spikeTestDrainDelay = 50 * time.Millisecond

// spikeBenchDrainDelay keeps the per-iteration shutdown cost near-zero in the
// adoption benchmark. The 2-second production drain would force the benchmark
// framework to spawn one goroutine, run Shutdown, wait DrainDelay, and then
// proceed to the next b.N iteration's server bring-up — b.N scales by elapsed
// time, so an in-timer 2s drain poisons every measurement. By making the
// drain near-instant, both the baseline and the appkit cases differ only by
// the request-path work each stack actually performs.
const spikeBenchDrainDelay = 10 * time.Millisecond

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
	stop := serveInBackground(t, func(ctx context.Context, addr string, h http.Handler) error {
		return bundle.runWithAppkit(ctx, addr, h, spikeTestDrainDelay, "")
	}, addr, sse)
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
	stop := serveInBackground(t, func(ctx context.Context, addr string, h http.Handler) error {
		return bundle.runWithAppkit(ctx, addr, h, spikeTestDrainDelay, "")
	}, addr, nil)
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
	stop := serveInBackground(t, func(ctx context.Context, addr string, h http.Handler) error {
		return bundle.runWithAppkit(ctx, addr, h, spikeTestDrainDelay, "")
	}, addr, hello)
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
// RunWithAppkit (appkit.Service: extra middleware + readiness probe). Used
// to decide whether the ADR-001 appkit adoption costs any per-request work
// worth caring about (comparison-report finding 7).
//
// The 10ms DrainDelay (vs the 2s production default) keeps per-iteration
// shutdown cost near zero in both cases; the delta we measure is the actual
// request-path work each stack does, not drain phase.
//
// Per-request INFO logs from appkit's Logging middleware dominate the
// measurement when logging stays at INFO — finding 7 showed 16178 vs 45049
// ns/op with logging on, ~28µs/request of which ~22µs is the formatted line.
// The appkit sub-benchmark passes LogLevelError to suppress those lines so
// the comparison isolates stack overhead from observability overhead;
// logging posture is a deliberate decision, not a benchmark artifact. As
// belt-and-suspenders for any third-party dependency that talks to
// slog.Default(), we also pin a discard handler for the duration of the
// sub-benchmark and restore it on cleanup so the surrounding suite keeps
// its normal logs.
//
// Invocation:
//
//	# 5× runs, benchstat-friendly (alloc + throughput metrics):
//	go test -run xxx -bench '^BenchmarkSpikeBaselineVsAppkit$' \
//	    -benchtime=2s -benchmem -count=5 -timeout=120s \
//	    ./setup | tee /tmp/before.txt
//	# edit source, repeat into /tmp/after.txt
//	benchstat /tmp/before.txt /tmp/after.txt
func BenchmarkSpikeBaselineVsAppkit(b *testing.B) {
	ping := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	cases := []struct {
		name string
		run  func(*Bundle, context.Context, string, http.Handler, time.Duration, appkit.LogLevel) error
	}{
		{
			name: "baseline-httputil",
			run: func(b *Bundle, ctx context.Context, addr string, h http.Handler, _ time.Duration, _ appkit.LogLevel) error {
				return b.RunHandler(ctx, addr, h)
			},
		},
		{
			name: "appkit-service",
			run: func(b *Bundle, ctx context.Context, addr string, h http.Handler, drainDelay time.Duration, logLevel appkit.LogLevel) error {
				return b.runWithAppkit(ctx, addr, h, drainDelay, logLevel)
			},
		},
	}

	for _, bc := range cases {
		b.Run(bc.name, func(b *testing.B) {
			// Pin slog to discard before serving so anything that talks to
			// slog.Default() (appkit's logger uses its own, but third-party
			// dependencies in the chain might not) stays silent.
			prevLogger := slog.Default()
			slog.SetDefault(slog.New(slog.DiscardHandler))

			b.Cleanup(func() { slog.SetDefault(prevLogger) })

			bundle := MustNew(Config{Title: "bench-" + bc.name})

			addr := freeLocalAddr(b)
			stop := serveInBackground(b, func(ctx context.Context, addr string, h http.Handler) error {
				return bc.run(bundle, ctx, addr, h, spikeBenchDrainDelay, appkit.LogLevelError)
			}, addr, ping)

			// The run funcs close the bundle on every exit path; stop() waits
			// for the (short) drain after the measured loop finishes. b.Cleanup
			// runs after b's timer has been stopped, so the 10ms drain never
			// counts toward ns/op.
			b.Cleanup(func() {
				b.StopTimer()
				stop()
			})

			client := &http.Client{Timeout: 5 * time.Second}

			req, err := http.NewRequestWithContext(
				context.Background(), http.MethodGet, "http://"+addr+"/", nil)
			if err != nil {
				b.Fatalf("build request: %v", err)
			}

			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				resp, err := client.Do(req)
				if err != nil {
					b.Fatalf("request: %v", err)
				}

				_, _ = io.Copy(io.Discard, resp.Body)
				_ = resp.Body.Close()
			}

			// benchstat-friendly throughput column (req/s) alongside the
			// default ns/op and (with -benchmem) B/op + allocs/op.
			b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "req/s")
		})
	}
}
