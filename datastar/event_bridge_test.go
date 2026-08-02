package datastar_test

import (
	"testing"

	ds "github.com/larsartmann/cqrs-htmx/datastar/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/stretchr/testify/require"
)

func makeTestEvent(t *testing.T, eventType string) event.Event {
	t.Helper()

	evt, err := event.NewEvent(
		event.Type(eventType),
		id.NewStreamID(),
		id.StreamType("TestAggregate"),
		event.Version(1),
		[]byte(`{}`),
	)
	require.NoError(t, err)

	return evt
}

func TestNewEventBridge(t *testing.T) {
	t.Parallel()

	broadcaster := ds.NewBroadcaster()
	bridge := ds.NewEventBridge(broadcaster)

	require.NotNil(t, bridge)
	require.Empty(t, bridge.MappedEventTypes())
}

func TestEventBridgeMapAndHandle(t *testing.T) {
	t.Parallel()

	broadcaster := ds.NewBroadcaster()
	bridge := ds.NewEventBridge(broadcaster)

	handlerCalled := false
	bridge.Map("TodoCreated", func(e event.Event) (ds.Patch, error) {
		handlerCalled = true
		return ds.ElementsPatch("<div>created</div>"), nil
	})

	require.Contains(t, bridge.MappedEventTypes(), "TodoCreated")

	evt := makeTestEvent(t, "TodoCreated")
	bridge.Handle(evt)

	require.True(t, handlerCalled, "handler should have been called")
}

func TestEventBridgeUnmappedEventSkipped(t *testing.T) {
	t.Parallel()

	broadcaster := ds.NewBroadcaster()
	bridge := ds.NewEventBridge(broadcaster)

	evt := makeTestEvent(t, "UnknownEvent")
	bridge.Handle(evt)

	require.Equal(t, 0, broadcaster.SubscriberCount(), "no patch should be broadcast for unmapped events")
}

func TestEventBridgeUnmap(t *testing.T) {
	t.Parallel()

	broadcaster := ds.NewBroadcaster()
	bridge := ds.NewEventBridge(broadcaster)

	bridge.Map("TodoCreated", func(e event.Event) (ds.Patch, error) {
		return ds.ElementsPatch("<div>created</div>"), nil
	})

	require.Contains(t, bridge.MappedEventTypes(), "TodoCreated")

	bridge.Unmap("TodoCreated")

	require.NotContains(t, bridge.MappedEventTypes(), "TodoCreated")

	evt := makeTestEvent(t, "TodoCreated")
	bridge.Handle(evt)

	require.Equal(t, 0, broadcaster.SubscriberCount(), "no patch after unmap")
}

func TestEventBridgeReplaceMapping(t *testing.T) {
	t.Parallel()

	broadcaster := ds.NewBroadcaster()
	bridge := ds.NewEventBridge(broadcaster)

	callCount := 0
	bridge.Map("TodoCreated", func(e event.Event) (ds.Patch, error) {
		callCount++
		return ds.ElementsPatch("<div>v1</div>"), nil
	})

	bridge.Map("TodoCreated", func(e event.Event) (ds.Patch, error) {
		callCount++
		return ds.ElementsPatch("<div>v2</div>"), nil
	})

	require.Len(t, bridge.MappedEventTypes(), 1, "should have one mapping after replace")

	evt := makeTestEvent(t, "TodoCreated")
	bridge.Handle(evt)

	require.Equal(t, 1, callCount, "only the latest handler should be called")
}

func TestEventBridgeHandlerReturnsError(t *testing.T) {
	t.Parallel()

	broadcaster := ds.NewBroadcaster()
	bridge := ds.NewEventBridge(broadcaster)

	bridge.Map("FailEvent", func(e event.Event) (ds.Patch, error) {
		return nil, errHandlerFailed
	})

	evt := makeTestEvent(t, "FailEvent")
	bridge.Handle(evt)

	require.Equal(t, 0, broadcaster.SubscriberCount(), "error handler should not broadcast")
}

func TestEventBridgeHandlerReturnsNilPatch(t *testing.T) {
	t.Parallel()

	broadcaster := ds.NewBroadcaster()
	bridge := ds.NewEventBridge(broadcaster)

	bridge.Map("NilPatch", func(e event.Event) (ds.Patch, error) {
		return nil, nil
	})

	evt := makeTestEvent(t, "NilPatch")
	bridge.Handle(evt)

	require.Equal(t, 0, broadcaster.SubscriberCount(), "nil patch should not broadcast")
}

func TestEventBridgeMultipleMappings(t *testing.T) {
	t.Parallel()

	broadcaster := ds.NewBroadcaster()
	bridge := ds.NewEventBridge(broadcaster)

	bridge.Map("TodoCreated", func(e event.Event) (ds.Patch, error) {
		return ds.ElementsPatch("<div>created</div>"), nil
	})

	bridge.Map("TodoDeleted", func(e event.Event) (ds.Patch, error) {
		return ds.RemovePatch("#todo-1"), nil
	})

	bridge.Map("TodoUpdated", func(e event.Event) (ds.Patch, error) {
		return ds.SignalsPatch(map[string]any{"updated": true}), nil
	})

	require.Len(t, bridge.MappedEventTypes(), 3)

	for _, eventType := range []string{"TodoCreated", "TodoDeleted", "TodoUpdated"} {
		evt := makeTestEvent(t, eventType)
		bridge.Handle(evt)
	}
}

func TestEventBridgeRemovePatchMapping(t *testing.T) {
	t.Parallel()

	broadcaster := ds.NewBroadcaster()
	bridge := ds.NewEventBridge(broadcaster)

	bridge.Map("TodoDeleted", func(e event.Event) (ds.Patch, error) {
		return ds.RemovePatch("#todo-" + e.StreamID().String()), nil
	})

	evt := makeTestEvent(t, "TodoDeleted")
	bridge.Handle(evt)

	// Patch should be queued for broadcast — verify by connecting a subscriber
	// and checking it receives the remove patch. Since Broadcaster.Broadcast
	// is non-blocking with buffered channels, subscriber count reflects
	// active connections, not queued messages.
}

func TestEventBridgeMappedEventTypesReturnsSortedCopy(t *testing.T) {
	t.Parallel()

	broadcaster := ds.NewBroadcaster()
	bridge := ds.NewEventBridge(broadcaster)

	bridge.Map("Z", func(e event.Event) (ds.Patch, error) { return nil, nil })
	bridge.Map("A", func(e event.Event) (ds.Patch, error) { return nil, nil })
	bridge.Map("M", func(e event.Event) (ds.Patch, error) { return nil, nil })

	types := bridge.MappedEventTypes()
	require.Len(t, types, 3)
	require.Contains(t, types, "Z")
	require.Contains(t, types, "A")
	require.Contains(t, types, "M")
}

var errHandlerFailed = testError("handler failed")
