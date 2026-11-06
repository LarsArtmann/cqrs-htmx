package cqrshtmx

import (
	"sync/atomic"

	"github.com/larsartmann/go-sse"
)

// DropStats counts SSE events dropped because a subscriber's buffer was
// full. Slow-consumer drops are otherwise silent: the broadcast contract is
// non-blocking by design, so the only honest way to know a client fell
// behind is to count it.
//
// The counter is wired through go-sse's OnDrop callback and follows its
// documented contract exactly: the callback runs on the broadcasting
// goroutine inside the fan-out read lock and must be fast and non-blocking —
// a single atomic increment. It fires once per FULL SUBSCRIBER per
// broadcast, not once per unique message: one Broadcast to N stuck
// subscribers counts N drops of the same event. Pair with idempotent
// consumers + reconnect replay (Last-Event-ID) for guaranteed eventual
// delivery.
//
// Attach at construction:
//
//	drops := cqrshtmx.NewDropStats()
//	broadcaster := cqrshtmx.NewBroadcaster(drops.Option())
//
// or onto an existing broadcaster:
//
//	drops.Attach(broadcaster)
type DropStats struct {
	total atomic.Int64
}

// NewDropStats returns a zeroed drop counter, ready to wire.
func NewDropStats() *DropStats {
	return &DropStats{total: atomic.Int64{}}
}

// Option returns the construction-time wiring for [NewBroadcaster]:
// pass it as an option and every drop increments the counter.
func (s *DropStats) Option() sse.Option[sse.Event] {
	return sse.WithOnDrop[sse.Event](func(sse.Event) { s.total.Add(1) })
}

// Attach wires the counter onto an existing broadcaster via the promoted
// OnDrop setter. Attaching to a broadcaster that already carries a callback
// replaces it — wire ONE DropStats per broadcaster.
func (s *DropStats) Attach(b *Broadcaster) {
	b.OnDrop(func(sse.Event) { s.total.Add(1) })
}

// Total returns the cumulative per-subscriber drop count since attach.
func (s *DropStats) Total() int64 {
	return s.total.Load()
}

// SSEHubSnapshot pairs a hub's lifecycle health with the drop count — one
// pollable struct for dashboards and admin surfaces.
type SSEHubSnapshot struct {
	Health sse.BroadcasterHealth
	Drops  int64
}

// Snapshot reads the broadcaster's health plus this counter's total. Safe to
// call from any goroutine; b must be non-nil.
func (s *DropStats) Snapshot(b *Broadcaster) SSEHubSnapshot {
	return SSEHubSnapshot{
		Health: b.Hub().Health(),
		Drops:  s.Total(),
	}
}
