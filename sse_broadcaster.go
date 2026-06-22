package cqrshtmx

import (
	"net/http"
)

// Broadcaster distributes SSE events to all subscribed clients.
// It is safe for concurrent use.
//
// Create one at application startup and share it across handlers:
//
//	broadcaster := cqrshtmx.NewBroadcaster()
//
//	// In your SSE endpoint handler:
//	ch := broadcaster.Subscribe()
//	defer broadcaster.Unsubscribe(ch)
//
//	// In your CQRS event handler or AfterDispatch hook:
//	broadcaster.Broadcast(cqrshtmx.SSEEvent{Event: "itemCreated", Data: html})
type Broadcaster struct {
	*fanOut[SSEEvent]
}

// NewBroadcaster creates a new event broadcaster with no subscribers.
func NewBroadcaster() *Broadcaster {
	return &Broadcaster{fanOut: newFanOut[SSEEvent]()}
}

// BroadcastOnSuccess creates an AfterDispatchHook that broadcasts an SSE event
// when a command dispatch succeeds (err == nil). This bridges the CQRS dispatch
// lifecycle with SSE real-time updates.
//
// Use it in Config.AfterDispatch to automatically notify SSE clients after
// successful command dispatch:
//
//	app, _ := cqrshtmx.New(cqrshtmx.Config{
//	    Commands: cmdDispatcher,
//	    AfterDispatch: broadcaster.BroadcastOnSuccess("itemUpdated", ""),
//	})
//
// For dynamic event data based on the request, use BroadcastOnSuccessFunc.
func (b *Broadcaster) BroadcastOnSuccess(eventName, data string) AfterDispatchHook {
	return b.broadcastOnSuccessHook(func(_ *http.Request) SSEEvent {
		return SSEEvent{Event: eventName, Data: data}
	})
}

// BroadcastOnSuccessFunc creates an AfterDispatchHook that generates an SSE event
// dynamically from the request when dispatch succeeds. The eventFunc receives the
// request and returns the SSE event to broadcast.
//
// Use this when the SSE event data depends on the dispatched command:
//
//	app, _ := cqrshtmx.New(cqrshtmx.Config{
//	    Commands: cmdDispatcher,
//	    AfterDispatch: broadcaster.BroadcastOnSuccessFunc(func(r *http.Request) cqrshtmx.SSEEvent {
//	        return cqrshtmx.SSEEvent{
//	            Event: "itemUpdated",
//	            Data:  renderItemsHTML(),
//	        }
//	    }),
//	})
func (b *Broadcaster) BroadcastOnSuccessFunc(eventFunc func(r *http.Request) SSEEvent) AfterDispatchHook {
	return b.broadcastOnSuccessHook(eventFunc)
}

// BroadcastOnError creates an AfterDispatchHook that broadcasts an SSE event
// when a command dispatch fails (err != nil). This is the error counterpart to
// BroadcastOnSuccess — together they ensure SSE clients are always notified,
// whether the command succeeded or failed.
//
// The error is serialized as a StructuredError (RFC 7807 shape) JSON string
// in the SSE event data field. Clients can parse and render it uniformly.
//
// Use it alongside BroadcastOnSuccess for complete real-time feedback:
//
//	app, _ := cqrshtmx.New(cqrshtmx.Config{
//	    Commands: cmdDispatcher,
//	    AfterDispatch: broadcaster.BroadcastOnError("commandError"),
//	})
//
// For dynamic error event generation, use BroadcastOnErrorFunc.
func (b *Broadcaster) BroadcastOnError(eventName string) AfterDispatchHook {
	return b.broadcastOnErrorHook(func(r *http.Request, err error) SSEEvent {
		payload := NewStructuredError(err, r)
		return SSEEvent{Event: eventName, Data: payload.JSON()}
	})
}

// BroadcastOnErrorFunc creates an AfterDispatchHook that generates an SSE error
// event dynamically from the request and error when dispatch fails.
// The errFunc receives both the request and the error, allowing callers to
// customize the event name, data, or retry behavior based on the error type.
//
// Example: suppress certain errors or add retry hints:
//
//	app, _ := cqrshtmx.New(cqrshtmx.Config{
//	    Commands: cmdDispatcher,
//	    AfterDispatch: broadcaster.BroadcastOnErrorFunc(func(r *http.Request, err error) cqrshtmx.SSEEvent {
//	        payload := cqrshtmx.NewStructuredError(err, r)
//	        return cqrshtmx.SSEEvent{
//	            Event: "commandError",
//	            Data:  payload.JSON(),
//	        }
//	    }),
//	})
func (b *Broadcaster) BroadcastOnErrorFunc(errFunc func(r *http.Request, err error) SSEEvent) AfterDispatchHook {
	return b.broadcastOnErrorHook(errFunc)
}
