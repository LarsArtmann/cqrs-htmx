package cqrshtmx_test

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v2"
	"github.com/larsartmann/go-cqrs-lite/command/v3"
)

// BenchmarkBroadcasterBroadcastStress validates the snapshot-based Broadcast
// optimization by measuring fan-out latency across subscriber counts.
func BenchmarkBroadcasterBroadcastStress(b *testing.B) {
	for _, n := range []int{1, 10, 100, 1000} {
		b.Run("subs_"+strconv.Itoa(n), func(b *testing.B) {
			bc := cqrshtmx.NewBroadcaster()
			for range n {
				ch := bc.Subscribe()
				go func() {
					for range ch {
					}
				}()
			}
			evt := cqrshtmx.SSEEvent{Event: "test", Data: "hello world from broadcaster"}
			b.ResetTimer()
			for b.Loop() {
				bc.Broadcast(evt)
			}
			b.StopTimer()
		})
	}
}

// BenchmarkBroadcasterConcurrentSubscribe validates that Subscribe/Unsubscribe
// can proceed concurrently with Broadcast (the T04 optimization).
func BenchmarkBroadcasterConcurrentSubscribe(b *testing.B) {
	bc := cqrshtmx.NewBroadcaster()

	for range 100 {
		ch := bc.Subscribe()
		go func() {
			for range ch {
			}
		}()
	}

	ctx := b.Context()
	go func() {
		evt := cqrshtmx.SSEEvent{Event: "tick", Data: "data"}
		for {
			select {
			case <-ctx.Done():
				return
			default:
				bc.Broadcast(evt)
			}
		}
	}()

	b.ResetTimer()
	for b.Loop() {
		ch := bc.Subscribe()
		bc.Unsubscribe(ch)
	}
}

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

var _ = sync.WaitGroup{}

// BenchmarkWSBroadcasterBroadcastStress validates WS fan-out across subscriber counts.
func BenchmarkWSBroadcasterBroadcastStress(b *testing.B) {
	for _, n := range []int{1, 10, 100, 1000} {
		b.Run("subs_"+strconv.Itoa(n), func(b *testing.B) {
			bc := cqrshtmx.NewWSBroadcaster()
			for range n {
				ch := bc.Subscribe()
				go func() {
					for range ch {
					}
				}()
			}
			msg := "<div hx-swap-oob='true'>hello world from ws broadcaster</div>"
			b.ResetTimer()
			for b.Loop() {
				bc.Broadcast(msg)
			}
			b.StopTimer()
		})
	}
}

// BenchmarkWSBroadcasterConcurrentSubscribe validates concurrent subscribe/unsubscribe.
func BenchmarkWSBroadcasterConcurrentSubscribe(b *testing.B) {
	bc := cqrshtmx.NewWSBroadcaster()

	for range 100 {
		ch := bc.Subscribe()
		go func() {
			for range ch {
			}
		}()
	}

	ctx := b.Context()
	go func() {
		msg := "<div>tick</div>"
		for {
			select {
			case <-ctx.Done():
				return
			default:
				bc.Broadcast(msg)
			}
		}
	}()

	b.ResetTimer()
	for b.Loop() {
		ch := bc.Subscribe()
		bc.Unsubscribe(ch)
	}
}

// BenchmarkSSEHeartbeatWrite measures the cost of writing a single SSE
// heartbeat comment frame — the same write SSEStream.Heartbeat performs on each tick.
func BenchmarkSSEHeartbeatWrite(b *testing.B) {
	w := httptest.NewRecorder()
	frame := []byte(": keepalive\n\n")
	b.ResetTimer()
	for b.Loop() {
		_, _ = w.Write(frame)
	}
}
