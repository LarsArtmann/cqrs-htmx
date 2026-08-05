package cqrshtmx_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/go-cqrs-lite/command/v4"
)

// NOTE: SSE broadcaster load benchmarks live in sse_broadcaster_bench_test.go
// (BenchmarkBroadcasterFanOut, BenchmarkBroadcasterBroadcastSaturated,
// BenchmarkSubscribeUnsubscribe, BenchmarkBroadcasterConcurrentBroadcast,
// BenchmarkBroadcastNoSubscribers). That file uses leak-safe drain helpers
// (b.Cleanup waits for drain goroutines) and reports allocs; the earlier
// versions that lived here leaked goroutines because they never Unsubscribed.

// BenchmarkServerCommandDispatch measures dispatch over a real net/http server
// instead of httptest.NewRecorder.
func BenchmarkServerCommandDispatch(b *testing.B) {
	disp := command.NewDispatcher()
	_ = disp.Register("CreateUser", noOpCommandHandler)

	app, _ := cqrshtmx.New(cqrshtmx.Config{Commands: disp})
	handler := app.Command("CreateUser", decodeCreateUserJSON())

	server := httptest.NewServer(handler)
	defer server.Close()

	client := &http.Client{Timeout: 5 * time.Second}
	body := strings.NewReader(`{}`)

	b.ResetTimer()

	for b.Loop() {
		body.Reset(`{}`)

		resp, err := client.Post(server.URL, "application/json", body)
		if err != nil {
			b.Fatal(err)
		}

		_ = resp.Body.Close()
	}
}

// BenchmarkServerHealthHandler measures the health endpoint over real HTTP.
func BenchmarkServerHealthHandler(b *testing.B) {
	app := cqrshtmx.MustNew(cqrshtmx.Config{Commands: command.NewDispatcher()})

	server := httptest.NewServer(app.HealthHandler())
	defer server.Close()

	client := &http.Client{Timeout: 5 * time.Second}

	b.ResetTimer()

	for b.Loop() {
		resp, err := client.Get(server.URL)
		if err != nil {
			b.Fatal(err)
		}

		_ = resp.Body.Close()
	}
}

// BenchmarkSSEHeartbeatWrite measures the cost of writing a single SSE
// heartbeat comment frame — the same write SSEStream.Heartbeat performs on each tick.
func BenchmarkSSEHeartbeatWrite(b *testing.B) {
	w := httptest.NewRecorder()
	frame := []byte(": heartbeat\n\n")

	b.ResetTimer()

	for b.Loop() {
		_, _ = w.Write(frame)
	}
}
