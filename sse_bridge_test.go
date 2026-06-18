package cqrshtmx_test

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// itemUpdatedEventFunc returns a BroadcastOnSuccessFunc payload function
// that emits an "itemUpdated" SSE event whose Data wraps the request path.
// Shared by the example in example_app_test.go and the test below.
func itemUpdatedEventFunc(r *http.Request) cqrshtmx.SSEEvent {
	return cqrshtmx.SSEEvent{
		Event: "itemUpdated",
		Data:  "<div>" + r.URL.Path + "</div>",
	}
}

var _ = Describe("SSE Broadcaster AfterDispatchHook Bridge", func() {
	Describe("Broadcaster AfterDispatchHook bridge", func() {
		It("broadcasts event on successful dispatch", func() {
			b := cqrshtmx.NewBroadcaster()
			ch := b.Subscribe()
			defer b.Unsubscribe(ch)

			hook := b.BroadcastOnSuccess("itemCreated", "<li>New item</li>")
			hook(context.Background(), httptest.NewRequest(http.MethodPost, "/items", nil), nil)

			Eventually(ch).Should(Receive(Equal(cqrshtmx.SSEEvent{
				Event: "itemCreated",
				Data:  "<li>New item</li>",
			})))
		})

		It("does not broadcast on dispatch error", func() {
			b := cqrshtmx.NewBroadcaster()
			ch := b.Subscribe()
			defer b.Unsubscribe(ch)

			hook := b.BroadcastOnSuccess("itemCreated", "data")
			hook(
				context.Background(),
				httptest.NewRequest(http.MethodPost, "/items", nil),
				errors.New("dispatch failed"),
			)

			Consistently(ch).ShouldNot(Receive())
		})

		It("broadcasts dynamic event via BroadcastOnSuccessFunc", func() {
			b := cqrshtmx.NewBroadcaster()
			ch := b.Subscribe()
			defer b.Unsubscribe(ch)

			hook := b.BroadcastOnSuccessFunc(itemUpdatedEventFunc)

			r := httptest.NewRequest(http.MethodPost, "/items/42", nil)
			hook(context.Background(), r, nil)

			Eventually(ch).Should(Receive(Equal(cqrshtmx.SSEEvent{
				Event: "itemUpdated",
				Data:  "<div>/items/42</div>",
			})))
		})

		It("does not broadcast dynamic event on dispatch error", func() {
			b := cqrshtmx.NewBroadcaster()
			ch := b.Subscribe()
			defer b.Unsubscribe(ch)

			hook := b.BroadcastOnSuccessFunc(func(_ *http.Request) cqrshtmx.SSEEvent {
				return cqrshtmx.SSEEvent{Event: "shouldNotFire", Data: "oops"}
			})

			hook(context.Background(), nil, errors.New("fail"))
			Consistently(ch).ShouldNot(Receive())
		})
	})
})

// errorWriter always returns an error on Write.
type errorWriter struct{}

func (e *errorWriter) Write(_ []byte) (int, error) {
	return 0, errors.New("write error")
}

// writeAndExpect writes an SSE event to a buffer and asserts the exact output.
func writeAndExpect(event cqrshtmx.SSEEvent, want string) {
	var buf bytes.Buffer
	err := cqrshtmx.WriteSSEEvent(&buf, event)
	Expect(err).NotTo(HaveOccurred())
	Expect(buf.String()).To(Equal(want))
}

// errorResponseWriter wraps errorWriter as an http.ResponseWriter for SSE stream tests.
type errorResponseWriter struct {
	http.ResponseWriter
	writer *errorWriter
}

func (e *errorResponseWriter) Write(p []byte) (int, error) {
	return e.writer.Write(p)
}

func (e *errorResponseWriter) Flush() {}

type memoryEventStore struct {
	events []cqrshtmx.SSEEvent
}

func (m *memoryEventStore) EventsAfter(lastID string) []cqrshtmx.SSEEvent {
	if lastID == "" {
		return m.events
	}
	for i, evt := range m.events {
		if evt.ID == lastID {
			if i+1 < len(m.events) {
				return m.events[i+1:]
			}
			return nil
		}
	}
	return nil
}
