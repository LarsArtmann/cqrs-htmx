package datastar

import (
	"github.com/larsartmann/go-datastar/broadcast"
	"github.com/larsartmann/go-sse"
)

// Broadcaster fans out DataStar patches to all connected SSE clients.
//
// Deprecated: the implementation moved to the go-datastar upstream submodule
// [github.com/larsartmann/go-datastar/broadcast]. Import that module and use
// broadcast.NewBroadcaster, broadcast.NewBroadcasterWithReplay, or
// broadcast.NewBroadcasterFromHub directly; every method (Hub, Broadcast,
// BroadcastMany, BroadcastEvent, SubscriberCount, ServeHTTP, plus the promoted
// go-sse hub methods) exists there unchanged. This type alias remains for
// backward compatibility; removal is bundled with v5. The deprecated Raw
// method did not survive the move — use Hub.
type Broadcaster = broadcast.Broadcaster

// NewBroadcaster creates a DataStar patch broadcaster with default settings
// and no replay support.
//
// Deprecated: use [broadcast.NewBroadcaster]; removal is bundled with v5.
var NewBroadcaster = broadcast.NewBroadcaster

// NewBroadcasterWithBufferSize creates a broadcaster with a custom subscriber
// buffer size and no replay support.
//
// Deprecated: use [broadcast.NewBroadcasterWithBufferSize]; removal is bundled
// with v5.
var NewBroadcasterWithBufferSize = broadcast.NewBroadcasterWithBufferSize

// NewBroadcasterWithReplay creates a broadcaster that retains the last
// capacity events in an in-memory ring buffer for reconnection replay.
//
// Deprecated: use [broadcast.NewBroadcasterWithReplay]; removal is bundled
// with v5.
var NewBroadcasterWithReplay = broadcast.NewBroadcasterWithReplay

// NewBroadcasterFromHub wraps an existing [*sse.Broadcaster] in a
// [*Broadcaster], enabling cross-transport fan-out hub sharing.
//
// Deprecated: use [broadcast.NewBroadcasterFromHub]; removal is bundled
// with v5.
var NewBroadcasterFromHub = broadcast.NewBroadcasterFromHub

// NewBroadcasterFromRaw wraps an existing [*sse.Broadcaster] in a
// [*Broadcaster].
//
// Deprecated: use [broadcast.NewBroadcasterFromHub] — the hub is the canonical
// shareable object, not a "raw" escape hatch. Removal is bundled with v5.
func NewBroadcasterFromRaw(raw *sse.Broadcaster[sse.Event]) *Broadcaster {
	return broadcast.NewBroadcasterFromHub(raw)
}
