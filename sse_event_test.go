package cqrshtmx_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"time"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("SSE Event Writing and Streaming", func() {
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

		It("sends heartbeat comment frames at the given interval", func() {
			ctx, cancel := context.WithCancel(context.Background())

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/events", nil).WithContext(ctx)

			stream := cqrshtmx.NewSSEStream(w, r)
			done := make(chan struct{})
			go func() {
				defer close(done)
				stream.Heartbeat(ctx, 10*time.Millisecond)
			}()

			time.Sleep(50 * time.Millisecond)
			cancel()
			Eventually(done).Should(BeClosed())

			Expect(w.Body.String()).To(ContainSubstring(": keepalive\n\n"))
		})

		It("stops heartbeat when context is cancelled", func() {
			ctx, cancel := context.WithCancel(context.Background())

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/events", nil).WithContext(ctx)

			stream := cqrshtmx.NewSSEStream(w, r)
			done := make(chan struct{})
			go func() {
				defer close(done)
				stream.Heartbeat(ctx, 10*time.Millisecond)
			}()

			cancel()
			Eventually(done).Should(BeClosed())
		})

		It("fires OnDisconnect callbacks when Close is called", func() {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/events", nil)

			stream := cqrshtmx.NewSSEStream(w, r)

			called := false
			stream.OnDisconnect(func() {
				called = true
			})

			stream.Close()
			Expect(called).To(BeTrue())
		})

		It("fires multiple OnDisconnect callbacks in registration order", func() {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/events", nil)

			stream := cqrshtmx.NewSSEStream(w, r)

			order := []int{}
			stream.OnDisconnect(func() { order = append(order, 1) })
			stream.OnDisconnect(func() { order = append(order, 2) })
			stream.OnDisconnect(func() { order = append(order, 3) })

			stream.Close()
			Expect(order).To(Equal([]int{1, 2, 3}))
		})
	})

	Describe("SSEEventID", func() {
		DescribeTable(
			"ParseSSEEventID accepts valid IDs",
			func(input string) {
				id, err := cqrshtmx.ParseSSEEventID(input)
				Expect(err).NotTo(HaveOccurred())
				Expect(id).To(Equal(cqrshtmx.SSEEventID(input)))
			},
			Entry("empty (initial connection)", ""),
			Entry("numeric", "42"),
			Entry("prefixed", "evt-99"),
			Entry("uuid", "550e8400-e29b-41d4-a716-446655440000"),
			Entry("ulid", "01H8XGJWBWBAQ4TPJRA2STZ9G9"),
		)

		DescribeTable(
			"ParseSSEEventID rejects IDs with newlines",
			func(input string) {
				_, err := cqrshtmx.ParseSSEEventID(input)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("forbidden character"))
			},
			Entry("newline", "evt\n42"),
			Entry("carriage return", "evt\r42"),
			Entry("CRLF", "evt\r\n42"),
			Entry("only newline", "\n"),
		)

		It("NewSSEEventID constructs without validation", func() {
			Expect(cqrshtmx.NewSSEEventID("any-value")).To(Equal(cqrshtmx.SSEEventID("any-value")))
		})

		It("IsZero reports emptiness", func() {
			Expect(cqrshtmx.SSEEventID("").IsZero()).To(BeTrue())
			Expect(cqrshtmx.SSEEventID("x").IsZero()).To(BeFalse())
		})

		It("String returns the underlying value", func() {
			Expect(cqrshtmx.SSEEventID("evt-1").String()).To(Equal("evt-1"))
		})
	})
})
