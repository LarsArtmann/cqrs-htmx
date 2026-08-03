package cqrshtmx_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/go-sse"
)

func ExampleWriteSSEEvent() {
	w := httptest.NewRecorder()
	w.Header().Set("Content-Type", "text/event-stream")

	err := sse.WriteEvent(w, sse.Event{
		Event: "todoCreated",
		Data:  "<li>Buy milk</li>",
		ID:    sse.NewEventID("evt-1"),
	})
	if err != nil {
		panic(err)
	}

	fmt.Println(w.Body.String())
	// Output: event: todoCreated
	// data: <li>Buy milk</li>
	// id: evt-1
	//
}

func ExampleSSEStream() {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/events", nil)

	stream := sse.NewStream(w, r)
	defer func() { _ = stream.Close() }()

	_ = stream.Send(sse.Event{Event: "update", Data: "<div>new</div>"})
	_ = stream.SendData("update", "<div>newer</div>")

	fmt.Println(w.Header().Get("Content-Type"))
	// Output: text/event-stream
}

func ExampleBroadcaster() {
	b := cqrshtmx.NewBroadcaster()

	ch := b.Subscribe()
	defer b.Unsubscribe(ch)

	b.Broadcast(sse.Event{Event: "itemCreated", Data: "<li>item</li>"})

	evt := <-ch
	fmt.Println(evt.Event, evt.Data)
	// Output: itemCreated <li>item</li>
}

func ExampleBroadcaster_BroadcastOnError() {
	b := cqrshtmx.NewBroadcaster()

	ch := b.Subscribe()
	defer b.Unsubscribe(ch)

	hook := b.BroadcastOnError("commandError")

	r := httptest.NewRequest(http.MethodPost, "/api/cmd", nil)
	hook(context.Background(), r, errors.New("validation failed"))

	evt := <-ch
	fmt.Println(evt.Event)
	// Output: commandError
}

func ExampleStructuredError() {
	err := cqrshtmx.ErrValidationFailed
	se := cqrshtmx.NewStructuredError(err, nil)

	fmt.Println(se.Status, se.Type)
	// Output: 400 rejection
}

func ExampleWSBroadcaster() {
	b := cqrshtmx.NewWSBroadcaster()

	ch := b.Subscribe()
	defer b.Unsubscribe(ch)

	b.Broadcast("<div hx-swap-oob='true'>Updated</div>")

	msg := <-ch
	fmt.Println(msg)
	// Output: <div hx-swap-oob='true'>Updated</div>
}

func ExampleWriteWSMessage() {
	msg := cqrshtmx.WSMessage{
		Headers: map[string]string{"HX-Request": "true"},
		Body:    map[string]any{"action": "update"},
	}

	r := httptest.NewRecorder()
	_ = cqrshtmx.WriteWSMessage(r, msg)

	parsed, _ := cqrshtmx.ParseWSMessage(r.Body.Bytes())
	fmt.Println(parsed.Headers["HX-Request"], parsed.Body["action"])
	// Output: true update
}
