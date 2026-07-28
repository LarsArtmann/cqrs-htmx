package cqrshtmx_test

import (
	"net/http"
	"net/http/httptest"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("SSE Reconnection", func() {
	Describe("SSE Reconnection", func() {
		It("extracts Last-Event-ID from request", func() {
			r := httptest.NewRequest(http.MethodGet, "/events", nil)
			r.Header.Set("Last-Event-ID", "evt-42")
			Expect(cqrshtmx.LastEventIDFromRequest(r)).To(Equal(cqrshtmx.NewSSEEventID("evt-42")))
		})

		It("returns empty string when Last-Event-ID is not set", func() {
			r := httptest.NewRequest(http.MethodGet, "/events", nil)
			Expect(cqrshtmx.LastEventIDFromRequest(r)).To(BeZero())
		})

		It("exposes LastEventID on SSEStream", func() {
			r := httptest.NewRequest(http.MethodGet, "/events", nil)
			r.Header.Set("Last-Event-ID", "evt-99")

			w := httptest.NewRecorder()

			stream := cqrshtmx.NewSSEStream(w, r)
			defer func() { _ = stream.Close() }()

			Expect(stream.LastEventID()).To(Equal(cqrshtmx.NewSSEEventID("evt-99")))
		})

		It("replays events from store", func() {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/events", nil)
			r.Header.Set("Last-Event-ID", "2")

			stream := cqrshtmx.NewSSEStream(w, r)
			defer func() { _ = stream.Close() }()

			store := &memoryEventStore{
				events: []cqrshtmx.SSEEvent{
					{Event: eventItem, Data: dataFirst, ID: cqrshtmx.NewSSEEventID("1")},
					{Event: eventItem, Data: "second", ID: cqrshtmx.NewSSEEventID("2")},
					{Event: eventItem, Data: "third", ID: cqrshtmx.NewSSEEventID("3")},
					{Event: eventItem, Data: "fourth", ID: cqrshtmx.NewSSEEventID("4")},
				},
			}

			n, err := cqrshtmx.ReplayEvents(stream, store, cqrshtmx.NewSSEEventID("2"))
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
			defer func() { _ = stream.Close() }()

			store := &memoryEventStore{
				events: []cqrshtmx.SSEEvent{
					{Event: eventItem, Data: dataFirst, ID: cqrshtmx.NewSSEEventID("1")},
				},
			}

			n, err := cqrshtmx.ReplayEvents(stream, store, cqrshtmx.NewSSEEventID("1"))
			Expect(err).NotTo(HaveOccurred())
			Expect(n).To(Equal(0))
		})

		It("returns error on write failure during replay", func() {
			ew := &errorWriter{}
			w := &errorResponseWriter{ResponseWriter: httptest.NewRecorder(), writer: ew}
			r := httptest.NewRequest(http.MethodGet, "/events", nil)

			stream := cqrshtmx.NewSSEStream(w, r)
			defer func() { _ = stream.Close() }()

			store := &memoryEventStore{
				events: []cqrshtmx.SSEEvent{
					{Event: eventItem, Data: dataFirst, ID: cqrshtmx.NewSSEEventID("1")},
				},
			}

			_, err := cqrshtmx.ReplayEvents(stream, store, cqrshtmx.NewSSEEventID(""))
			Expect(err).To(HaveOccurred())
		})
	})
})
