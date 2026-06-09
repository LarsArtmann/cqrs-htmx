package cqrshtmx_test

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	eventTodoCreated = "todoCreated"
	eventUpdate      = "update"
	eventItem        = "item"
	dataFirst        = "first"
)

var _ = Describe("SSE", func() {
	Describe("WriteSSEEvent", func() {
		It("writes a named event with data", func() {
			writeAndExpect(cqrshtmx.SSEEvent{
				Event: eventTodoCreated,
				Data:  "<div>Buy milk</div>",
			}, "event: "+eventTodoCreated+"\ndata: <div>Buy milk</div>\n\n")
		})

		It("writes an unnamed message event", func() {
			writeAndExpect(cqrshtmx.SSEEvent{
				Data: "<div>content</div>",
			}, "data: <div>content</div>\n\n")
		})

		It("writes multi-line data", func() {
			writeAndExpect(cqrshtmx.SSEEvent{
				Event: eventUpdate,
				Data:  "line1\nline2\nline3",
			}, "event: update\ndata: line1\ndata: line2\ndata: line3\n\n")
		})

		It("writes an event ID", func() {
			var buf bytes.Buffer
			err := cqrshtmx.WriteSSEEvent(&buf, cqrshtmx.SSEEvent{
				Data: "payload",
				ID:   "42",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(buf.String()).To(ContainSubstring("id: 42\n"))
		})

		It("writes a retry interval", func() {
			var buf bytes.Buffer
			err := cqrshtmx.WriteSSEEvent(&buf, cqrshtmx.SSEEvent{
				Data:  "payload",
				Retry: 5000,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(buf.String()).To(ContainSubstring("retry: 5000\n"))
		})

		It("writes a complete event with all fields", func() {
			var buf bytes.Buffer
			err := cqrshtmx.WriteSSEEvent(&buf, cqrshtmx.SSEEvent{
				Event: "fullEvent",
				Data:  "multi\nline\ndata",
				ID:    "evt-123",
				Retry: 3000,
			})
			Expect(err).NotTo(HaveOccurred())

			output := buf.String()
			Expect(output).To(HavePrefix("event: fullEvent\n"))
			Expect(output).To(ContainSubstring("data: multi\ndata: line\ndata: data\n"))
			Expect(output).To(ContainSubstring("id: evt-123\n"))
			Expect(output).To(ContainSubstring("retry: 3000\n"))
			Expect(output).To(HaveSuffix("\n\n"))
		})

		It("handles empty data", func() {
			writeAndExpect(cqrshtmx.SSEEvent{
				Event: "empty",
			}, "event: empty\ndata: \n\n")
		})

		It("handles CRLF line endings in data", func() {
			writeAndExpect(cqrshtmx.SSEEvent{
				Data: "line1\r\nline2",
			}, "data: line1\ndata: line2\n\n")
		})

		It("returns error on write failure", func() {
			err := cqrshtmx.WriteSSEEvent(&errorWriter{}, cqrshtmx.SSEEvent{
				Event: "fail",
			})
			Expect(err).To(HaveOccurred())
		})

		It("handles trailing newline in data", func() {
			writeAndExpect(cqrshtmx.SSEEvent{
				Event: "trailing",
				Data:  "line1\n",
			}, "event: trailing\ndata: line1\n\n")
		})

		It("handles multiple consecutive newlines", func() {
			writeAndExpect(cqrshtmx.SSEEvent{
				Data: "a\n\nb",
			}, "data: a\ndata: \ndata: b\n\n")
		})

		It("handles CRLF-only string", func() {
			writeAndExpect(cqrshtmx.SSEEvent{
				Data: "\r\n",
			}, "data: \n\n")
		})

		It("handles multiple CRLF lines with trailing CRLF", func() {
			writeAndExpect(cqrshtmx.SSEEvent{
				Data: "line1\r\nline2\r\n",
			}, "data: line1\ndata: line2\n\n")
		})
	})

	Describe("SSEStream", func() {
		It("sets correct SSE headers", func() {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/events", nil)

			stream := cqrshtmx.NewSSEStream(w, r)
			defer stream.Close()

			Expect(w.Header().Get("Content-Type")).To(Equal("text/event-stream"))
			Expect(w.Header().Get("Cache-Control")).To(Equal("no-cache"))
			Expect(w.Header().Get("Connection")).To(Equal("keep-alive"))
			Expect(w.Code).To(Equal(http.StatusOK))
		})

		It("sends events to the response writer", func() {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/events", nil)

			stream := cqrshtmx.NewSSEStream(w, r)
			defer stream.Close()

			err := stream.Send(cqrshtmx.SSEEvent{
				Event: eventUpdate,
				Data:  "<div>new content</div>",
			})
			Expect(err).NotTo(HaveOccurred())

			body := w.Body.String()
			Expect(body).To(Equal("event: update\ndata: <div>new content</div>\n\n"))
		})

		It("sends multiple events sequentially", func() {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/events", nil)

			stream := cqrshtmx.NewSSEStream(w, r)
			defer stream.Close()

			Expect(stream.Send(cqrshtmx.SSEEvent{Event: "e1", Data: "d1"})).To(Succeed())
			Expect(stream.Send(cqrshtmx.SSEEvent{Event: "e2", Data: "d2"})).To(Succeed())

			body := w.Body.String()
			Expect(body).To(ContainSubstring("event: e1\ndata: d1\n\n"))
			Expect(body).To(ContainSubstring("event: e2\ndata: d2\n\n"))
		})

		It("sends HTML via SendHTML convenience method", func() {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/events", nil)

			stream := cqrshtmx.NewSSEStream(w, r)
			defer stream.Close()

			err := stream.SendHTML("todoUpdated", "<ul><li>Buy milk</li></ul>")
			Expect(err).NotTo(HaveOccurred())

			body := w.Body.String()
			Expect(body).To(Equal(
				"event: todoUpdated\ndata: <ul><li>Buy milk</li></ul>\n\n",
			))
		})

		It("exposes the request context", func() {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/events", nil).WithContext(ctx)

			stream := cqrshtmx.NewSSEStream(w, r)
			defer stream.Close()

			Expect(stream.Context().Done()).NotTo(BeNil())
		})

		It("detects context cancellation", func() {
			ctx, cancel := context.WithCancel(context.Background())

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/events", nil).WithContext(ctx)

			stream := cqrshtmx.NewSSEStream(w, r)
			defer stream.Close()

			cancel()

			Eventually(stream.Context().Done()).Should(BeClosed())
		})
	})

	Describe("Broadcaster", func() {
		It("starts with zero subscribers", func() {
			b := cqrshtmx.NewBroadcaster()
			Expect(b.SubscriberCount()).To(Equal(0))
		})

		It("adds subscribers via Subscribe", func() {
			b := cqrshtmx.NewBroadcaster()
			ch := b.Subscribe()
			defer b.Unsubscribe(ch)

			Expect(b.SubscriberCount()).To(Equal(1))
		})

		It("tracks multiple subscribers", func() {
			b := cqrshtmx.NewBroadcaster()
			ch1 := b.Subscribe()
			ch2 := b.Subscribe()
			ch3 := b.Subscribe()

			Expect(b.SubscriberCount()).To(Equal(3))

			b.Unsubscribe(ch1)
			Expect(b.SubscriberCount()).To(Equal(2))

			b.Unsubscribe(ch2)
			b.Unsubscribe(ch3)
			Expect(b.SubscriberCount()).To(Equal(0))
		})

		It("broadcasts events to all subscribers", func() {
			b := cqrshtmx.NewBroadcaster()
			ch1 := b.Subscribe()
			ch2 := b.Subscribe()
			defer b.Unsubscribe(ch1)
			defer b.Unsubscribe(ch2)

			event := cqrshtmx.SSEEvent{Event: eventUpdate, Data: "<div>new</div>"}
			b.Broadcast(event)

			Expect(<-ch1).To(Equal(event))
			Expect(<-ch2).To(Equal(event))
		})

		It("closes the channel on unsubscribe", func() {
			b := cqrshtmx.NewBroadcaster()
			ch := b.Subscribe()
			b.Unsubscribe(ch)

			_, ok := <-ch
			Expect(ok).To(BeFalse())
		})

		It("handles unsubscribe of unknown channel gracefully", func() {
			b := cqrshtmx.NewBroadcaster()
			ch := make(chan cqrshtmx.SSEEvent)

			Expect(func() { b.Unsubscribe(ch) }).NotTo(Panic())
			Expect(b.SubscriberCount()).To(Equal(0))
		})

		It("drops events for slow subscribers", func() {
			b := cqrshtmx.NewBroadcaster()
			ch := b.Subscribe()
			defer b.Unsubscribe(ch)

			// The buffer is 64; send 65 events to overflow
			for i := range 65 {
				b.Broadcast(cqrshtmx.SSEEvent{Event: "e", Data: string(rune(i))})
			}

			// Should have received 64 (buffer size), lost 1
			received := 0
			for {
				select {
				case <-ch:
					received++
				default:
					goto done
				}
			}
		done:
			Expect(received).To(Equal(64))
		})

		It("is safe for concurrent use", func() {
			b := cqrshtmx.NewBroadcaster()

			var wg sync.WaitGroup
			const goroutines = 10
			channels := make([]<-chan cqrshtmx.SSEEvent, goroutines)

			for i := range goroutines {
				ch := b.Subscribe()
				channels[i] = ch
				wg.Go(func() {
					defer b.Unsubscribe(ch)
					<-ch
				})
			}

			b.Broadcast(cqrshtmx.SSEEvent{Event: "test", Data: "concurrent"})

			Eventually(func() int { return b.SubscriberCount() }).Should(BeNumerically("<=", goroutines))
			wg.Wait()
		})

		It("delivers events in order to a subscriber", func() {
			b := cqrshtmx.NewBroadcaster()
			ch := b.Subscribe()
			defer b.Unsubscribe(ch)

			for i := range 5 {
				b.Broadcast(cqrshtmx.SSEEvent{Event: "seq", Data: string(rune('0' + i))})
			}

			for i := range 5 {
				evt := <-ch
				Expect(evt.Data).To(Equal(string(rune('0' + i))))
			}
		})
	})

	Describe("Integration: Broadcaster + SSEStream", func() {
		It("fans out events to SSE streams", func() {
			b := cqrshtmx.NewBroadcaster()

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/events", nil)
			stream := cqrshtmx.NewSSEStream(w, r)

			ch := b.Subscribe()

			done := make(chan struct{})
			go func() {
				defer close(done)
				defer b.Unsubscribe(ch)
				defer stream.Close()

				for {
					select {
					case <-stream.Context().Done():
						return
					case evt := <-ch:
						if err := stream.Send(evt); err != nil {
							return
						}
						return
					}
				}
			}()

			b.Broadcast(cqrshtmx.SSEEvent{Event: eventTodoCreated, Data: "<li>Buy milk</li>"})

			Eventually(done).Should(BeClosed())

			body := w.Body.String()
			Expect(body).To(Equal("event: todoCreated\ndata: <li>Buy milk</li>\n\n"))
		})
	})

	Describe("SSE Reconnection", func() {
		It("extracts Last-Event-ID from request", func() {
			r := httptest.NewRequest(http.MethodGet, "/events", nil)
			r.Header.Set("Last-Event-ID", "evt-42")
			Expect(cqrshtmx.LastEventIDFromRequest(r)).To(Equal("evt-42"))
		})

		It("returns empty string when Last-Event-ID is not set", func() {
			r := httptest.NewRequest(http.MethodGet, "/events", nil)
			Expect(cqrshtmx.LastEventIDFromRequest(r)).To(BeEmpty())
		})

		It("exposes LastEventID on SSEStream", func() {
			r := httptest.NewRequest(http.MethodGet, "/events", nil)
			r.Header.Set("Last-Event-ID", "evt-99")
			w := httptest.NewRecorder()
			stream := cqrshtmx.NewSSEStream(w, r)
			defer stream.Close()

			Expect(stream.LastEventID()).To(Equal("evt-99"))
		})

		It("replays events from store", func() {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/events", nil)
			r.Header.Set("Last-Event-ID", "2")
			stream := cqrshtmx.NewSSEStream(w, r)
			defer stream.Close()

			store := &memoryEventStore{
				events: []cqrshtmx.SSEEvent{
					{Event: eventItem, Data: dataFirst, ID: "1"},
					{Event: eventItem, Data: "second", ID: "2"},
					{Event: eventItem, Data: "third", ID: "3"},
					{Event: eventItem, Data: "fourth", ID: "4"},
				},
			}

			n, err := cqrshtmx.ReplayEvents(stream, store, "2")
			Expect(err).NotTo(HaveOccurred())
			Expect(n).To(Equal(2))

			body := w.Body.String()
			Expect(body).To(ContainSubstring("id: 3"))
			Expect(body).To(ContainSubstring("data: third"))
			Expect(body).To(ContainSubstring("id: 4"))
			Expect(body).To(ContainSubstring("data: fourth"))
			Expect(body).NotTo(ContainSubstring("data: first"))
			Expect(body).NotTo(ContainSubstring("data: second"))
		})

		It("replays zero events when no events after lastID", func() {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/events", nil)
			stream := cqrshtmx.NewSSEStream(w, r)
			defer stream.Close()

			store := &memoryEventStore{
				events: []cqrshtmx.SSEEvent{
					{Event: eventItem, Data: dataFirst, ID: "1"},
				},
			}

			n, err := cqrshtmx.ReplayEvents(stream, store, "1")
			Expect(err).NotTo(HaveOccurred())
			Expect(n).To(Equal(0))
		})

		It("returns error on write failure during replay", func() {
			ew := &errorWriter{}
			w := &errorResponseWriter{ResponseWriter: httptest.NewRecorder(), writer: ew}
			r := httptest.NewRequest(http.MethodGet, "/events", nil)
			stream := cqrshtmx.NewSSEStream(w, r)
			defer stream.Close()

			store := &memoryEventStore{
				events: []cqrshtmx.SSEEvent{
					{Event: eventItem, Data: dataFirst, ID: "1"},
				},
			}

			_, err := cqrshtmx.ReplayEvents(stream, store, "")
			Expect(err).To(HaveOccurred())
		})
	})

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

			hook := b.BroadcastOnSuccessFunc(func(r *http.Request) cqrshtmx.SSEEvent {
				return cqrshtmx.SSEEvent{
					Event: "itemUpdated",
					Data:  "<div>" + r.URL.Path + "</div>",
				}
			})

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
