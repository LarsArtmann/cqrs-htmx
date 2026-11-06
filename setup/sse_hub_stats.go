package setup

// sseHubStats adapts the bundle's shared SSE hub and drop counter onto
// adminui.SSEHubStatsSource. Reads are live (per render), so construction
// order does not matter: attachAdmin runs BEFORE attachSSE wires the hub,
// and a hub-less bundle (no SSEPath/DataStarPath) reads as zeros — the
// adapter is only wired when one of those paths is set.
type sseHubStats struct {
	bundle *Bundle
}

// SubscriberCount reports the live SSE connections on the shared hub.
func (s sseHubStats) SubscriberCount() int {
	if s.bundle.Broadcaster == nil {
		return 0
	}

	return s.bundle.Broadcaster.Health().SubscriberCount
}

// DroppedEvents reports the lifetime slow-consumer drop count since start.
func (s sseHubStats) DroppedEvents() int64 {
	if s.bundle.SSEDrops == nil {
		return 0
	}

	return s.bundle.SSEDrops.Total()
}
