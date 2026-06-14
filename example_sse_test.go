package cqrshtmx_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v2"
)

func ExampleWriteSSEEvent() {
	w := httptest.NewRecorder()
	w.Header().Set("Content-Type", "text/event-stream")

	err := cqrshtmx.WriteSSEEvent(w, cqrshtmx.SSEEvent{
		Event: "todoCreated",
		Data:  "<li>Buy milk</li>",
		ID:    "evt-1",
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

	stream := cqrshtmx.NewSSEStream(w, r)
	defer stream.Close()

	_ = stream.Send(cqrshtmx.SSEEvent{Event: "update", Data: "<div>new</div>"})
	_ = stream.SendHTML("update", "<div>newer</div>")

	fmt.Println(w.Header().Get("Content-Type"))
	// Output: text/event-stream
}

func ExampleBroadcaster() {
	b := cqrshtmx.NewBroadcaster()
	ch := b.Subscribe()
	defer b.Unsubscribe(ch)

	b.Broadcast(cqrshtmx.SSEEvent{Event: "itemCreated", Data: "<li>item</li>"})

	evt := <-ch
	fmt.Println(evt.Event, evt.Data)
	// Output: itemCreated <li>item</li>
}
