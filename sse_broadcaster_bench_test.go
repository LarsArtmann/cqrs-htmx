package cqrshtmx_test

import (
	"testing"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/go-sse"
)

// drain launches a goroutine that receives every message on channel until it
// is closed (which happens on Unsubscribe). It simulates a fast SSE client
// whose network write keeps up with the broadcaster. tb.Cleanup blocks
// benchmark teardown until the goroutine has exited, preventing leaks.
func drain[T any](tb testing.TB, channel <-chan T) {
	tb.Helper()

	done := make(chan struct{})

	go func() {
		defer close(done)

		for range channel {
		}
	}()

	tb.Cleanup(func() { <-done })
}

// subscribeN wires count drained subscribers onto broadcaster and registers
// cleanup. Returns the channels so the caller can Unsubscribe them.
func subscribeN(b *testing.B, broadcaster *cqrshtmx.Broadcaster, count int) []<-chan sse.Event {
	b.Helper()

	channels := make([]<-chan sse.Event, 0, count)

	for range count {
		channel := broadcaster.Subscribe()
		channels = append(channels, channel)
	}

	b.Cleanup(func() {
		for _, channel := range channels {
			broadcaster.Unsubscribe(channel)
		}
	})

	return channels
}

// BenchmarkBroadcasterFanOut measures the per-broadcast cost as the subscriber
// count scales. Each subscriber is drained by a goroutine so the benchmark
// exercises the hot path (successful send), not the drop path. This is the
// primary "load testing under high fan-out" benchmark requested in TODO_LIST.
//
// benchstat: ns/op should scale ~linearly with subscriber count; allocs/op
// must stay at 0 (the broadcaster allocates nothing per Broadcast).
func BenchmarkBroadcasterFanOut(b *testing.B) {
	for _, subs := range []int{1, 10, 100, 1000} {
		b.Run("subs="+itoa(subs), func(b *testing.B) {
			broadcaster := cqrshtmx.NewBroadcaster()

			defer broadcaster.Close()

			channels := subscribeN(b, broadcaster, subs)

			for _, channel := range channels {
				drain(b, channel)
			}

			evt := sse.Event{Event: "bench", Data: "payload"}

			b.ReportAllocs()
			b.ResetTimer()

			for range b.N {
				broadcaster.Broadcast(evt)
			}
		})
	}
}

// BenchmarkBroadcasterBroadcastSaturated measures Broadcast cost when every
// subscriber's buffer is full (slow consumers). This exercises the drop path
// (the non-blocking `select { default: }`) and verifies Broadcast never blocks
// under saturation — the correctness guarantee that lets the broadcaster serve
// many clients without head-of-line blocking.
func BenchmarkBroadcasterBroadcastSaturated(b *testing.B) {
	for _, subs := range []int{1, 10, 100, 1000} {
		b.Run("subs="+itoa(subs), func(b *testing.B) {
			broadcaster := cqrshtmx.NewBroadcaster()

			defer broadcaster.Close()

			// Intentionally NOT drained: the 64-deep buffer fills after 64
			// broadcasts, then every send hits the drop path.
			_ = subscribeN(b, broadcaster, subs)

			evt := sse.Event{Event: "bench", Data: "payload"}

			b.ReportAllocs()
			b.ResetTimer()

			for range b.N {
				broadcaster.Broadcast(evt)
			}
		})
	}
}

// BenchmarkSubscribeUnsubscribe measures the churn cost: a client connecting
// and disconnecting repeatedly. This is the real-world load of browsers
// opening/closing SSE tabs. Subscribe stores into the map under a read-lock
// upgrade; Unsubscribe removes + closes the channel. Both are O(1).
func BenchmarkSubscribeUnsubscribe(b *testing.B) {
	broadcaster := cqrshtmx.NewBroadcaster()

	defer broadcaster.Close()

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		channel := broadcaster.Subscribe()

		broadcaster.Unsubscribe(channel)
	}
}

// BenchmarkBroadcastNoSubscribers is the baseline: Broadcast when nobody is
// listening. Establishes the floor cost of the RLock + map iteration (empty).
func BenchmarkBroadcastNoSubscribers(b *testing.B) {
	broadcaster := cqrshtmx.NewBroadcaster()

	defer broadcaster.Close()

	evt := sse.Event{Event: "bench", Data: "payload"}

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		broadcaster.Broadcast(evt)
	}
}

// BenchmarkBroadcasterConcurrentBroadcast measures fan-out under contention:
// multiple goroutines broadcasting simultaneously to a shared pool of
// subscribers. This stresses the RLock path and exposes any lock contention
// scaling issues at high concurrency.
func BenchmarkBroadcasterConcurrentBroadcast(b *testing.B) {
	const subscribers = 100

	broadcaster := cqrshtmx.NewBroadcaster()

	defer broadcaster.Close()

	channels := subscribeN(b, broadcaster, subscribers)

	for _, channel := range channels {
		drain(b, channel)
	}

	evt := sse.Event{Event: "bench", Data: "payload"}

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			broadcaster.Broadcast(evt)
		}
	})
}

// itoa is a tiny allocation-free int->string for benchmark names (avoids
// pulling fmt into a hot benchmark loop). Limited to non-negative values.
func itoa(value int) string {
	if value == 0 {
		return "0"
	}

	var buf [20]byte

	position := len(buf)

	for value > 0 {
		position--
		buf[position] = byte('0' + value%10)
		value /= 10
	}

	return string(buf[position:])
}
