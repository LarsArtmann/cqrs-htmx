package cqrshtmx_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"fmt"
	"sync"


	cqrshtmx "github.com/larsartmann/cqrs-htmx/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("SSE", func() {
	Describe("WriteSSEEvent", func() {
		It("writes a named event with data", func() {
			var buf bytes.Buffer
			err := cqrshtmx.WriteSSEEvent(&buf, cqrshtmx.SSEEvent{
				Event: "todoCreated",
				Data:  "<div>Buy milk</div>",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(buf.String()).To(Equal(
				"event: todoCreated\ndata: <div>Buy milk</div>\n\n",
			))
		})

		It("writes an unnamed message event", func() {
			var buf bytes.Buffer
			err := cqrshtmx.WriteSSEEvent(&buf, cqrshtmx.SSEEvent{
				Data: "<div>content</div>",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(buf.String()).To(Equal("data: <div>content</div>\n\n"))
		})

		It("writes multi-line data", func() {
			var buf bytes.Buffer
			err := cqrshtmx.WriteSSEEvent(&buf, cqrshtmx.SSEEvent{
				Event: "update",
				Data:  "line1\nline2\nline3",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(buf.String()).To(Equal(
				"event: update\ndata: line1\ndata: line2\ndata: line3\n\n",
			))
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
			var buf bytes.Buffer
			err := cqrshtmx.WriteSSEEvent(&buf, cqrshtmx.SSEEvent{
				Event: "empty",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(buf.String()).To(Equal("event: empty\ndata: \n\n"))
		})

		It("handles CRLF line endings in data", func() {
			var buf bytes.Buffer
			err := cqrshtmx.WriteSSEEvent(&buf, cqrshtmx.SSEEvent{
				Data: "line1\r\nline2",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(buf.String()).To(Equal("data: line1\ndata: line2\n\n"))
		})

		It("returns error on write failure", func() {
			err := cqrshtmx.WriteSSEEvent(&errorWriter{}, cqrshtmx.SSEEvent{
				Event: "fail",
			})
			Expect(err).To(HaveOccurred())
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
				Event: "update",
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

			event := cqrshtmx.SSEEvent{Event: "update", Data: "<div>new</div>"}
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
				wg.Add(1)
				go func() {
					defer wg.Done()
					defer b.Unsubscribe(ch)
					<-ch
				}()
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

			b.Broadcast(cqrshtmx.SSEEvent{Event: "todoCreated", Data: "<li>Buy milk</li>"})

			Eventually(done).Should(BeClosed())

			body := w.Body.String()
			Expect(body).To(Equal("event: todoCreated\ndata: <li>Buy milk</li>\n\n"))
		})
	})
})

// errorWriter always returns an error on Write.
type errorWriter struct{}

func (e *errorWriter) Write(_ []byte) (int, error) {
	return 0, fmt.Errorf("write error")
}