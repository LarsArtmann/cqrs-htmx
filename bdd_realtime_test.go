package cqrshtmx_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"time"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v3"
	"github.com/larsartmann/go-cqrs-lite/command/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// sseClient simulates one browser connected to a live-updating SSE endpoint.
// It subscribes to the broadcaster and forwards each received event onto its
// stream, stopping after stopAfter events so the forwarding goroutine always
// terminates. This mirrors the consumer handler pattern from NewSSEStream's
// docs, letting specs assert on the exact bytes the browser would receive.
type sseClient struct {
	recorder *httptest.ResponseRecorder
	stream   *cqrshtmx.SSEStream
	done     chan struct{}
}

func newSSEClient(b *cqrshtmx.Broadcaster, stopAfter int) *sseClient {
	rec := httptest.NewRecorder()
	// httptest.NewRequest provides its own context; the forwarding loop below
	// always exits via the data path (every client receives stopAfter events),
	// matching the fan-out integration test convention and avoiding a leaked
	// cancellable context.
	r := httptest.NewRequest(http.MethodGet, "/events", nil)
	stream := cqrshtmx.NewSSEStream(rec, r)
	ch := b.Subscribe()

	c := &sseClient{recorder: rec, stream: stream, done: make(chan struct{})}
	go func() {
		defer close(c.done)
		defer b.Unsubscribe(ch)
		defer stream.Close()
		received := 0
		for evt := range ch {
			if err := stream.Send(evt); err != nil {
				return
			}
			received++
			if received >= stopAfter {
				return
			}
		}
	}()
	return c
}

// lockedRecorder is a concurrency-safe http.ResponseWriter for SSE specs.
// httptest.ResponseRecorder embeds a *bytes.Buffer that is NOT safe for
// concurrent use: Heartbeat writes from its own goroutine while Gomega's
// Eventually polls the body from the spec goroutine. lockedRecorder guards
// the body with a mutex and exposes a locked accessor.
type lockedRecorder struct {
	*httptest.ResponseRecorder
	mu  sync.Mutex
	buf bytes.Buffer
}

func newLockedRecorder() *lockedRecorder {
	return &lockedRecorder{ResponseRecorder: httptest.NewRecorder()}
}

func (l *lockedRecorder) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.buf.Write(p)
}

func (l *lockedRecorder) Flush() {}

// body returns the written body so far, safe to call from another goroutine.
func (l *lockedRecorder) body() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.buf.String()
}

var _ = Describe("BDD: Realtime (SSE & WebSocket) Consumer Scenarios", func() {
	Describe(
		"As a consumer, I want to push live updates to every connected browser",
		func() {
			var broadcaster *cqrshtmx.Broadcaster

			BeforeEach(func() {
				broadcaster = cqrshtmx.NewBroadcaster()
			})

			It("responds to a connecting browser with the SSE protocol headers", func() {
				rec := httptest.NewRecorder()
				r := httptest.NewRequest(http.MethodGet, "/events", nil)
				stream := cqrshtmx.NewSSEStream(rec, r)
				defer stream.Close()

				Expect(rec.Code).To(Equal(http.StatusOK))
				Expect(rec.Header().Get("Content-Type")).To(Equal("text/event-stream"))
				Expect(rec.Header().Get("Cache-Control")).To(Equal("no-cache"))
				Expect(rec.Header().Get("Connection")).To(Equal("keep-alive"))
			})

			It("delivers a broadcast event to every connected client as valid SSE frames", func() {
				alice := newSSEClient(broadcaster, 1)
				bob := newSSEClient(broadcaster, 1)

				broadcaster.Broadcast(cqrshtmx.SSEEvent{
					Event: "itemUpdated",
					Data:  "<div id='items'>restocked</div>",
				})

				Eventually(alice.done).Should(BeClosed())
				Eventually(bob.done).Should(BeClosed())

				for _, body := range []string{alice.recorder.Body.String(), bob.recorder.Body.String()} {
					Expect(body).To(ContainSubstring("event: itemUpdated"))
					Expect(body).To(ContainSubstring("data: <div id='items'>restocked</div>"))
					Expect(body).To(HaveSuffix("\n\n"))
				}
			})

			It("splits a multi-line HTML payload across data lines so the browser renders each", func() {
				client := newSSEClient(broadcaster, 1)

				broadcaster.Broadcast(cqrshtmx.SSEEvent{
					Event: "listUpdated",
					Data:  "<li>Apple</li>\n<li>Banana</li>",
				})

				Eventually(client.done).Should(BeClosed())
				body := client.recorder.Body.String()
				Expect(body).To(ContainSubstring("data: <li>Apple</li>\ndata: <li>Banana</li>"))
			})

			It("sends an HTML fragment as a named event the browser can swap on", func() {
				rec := httptest.NewRecorder()
				r := httptest.NewRequest(http.MethodGet, "/events", nil)
				stream := cqrshtmx.NewSSEStream(rec, r)
				defer stream.Close()

				Expect(stream.SendHTML("toast", "<div class='toast'>Saved</div>")).
					To(Succeed())

				body := rec.Body.String()
				Expect(body).To(ContainSubstring("event: toast"))
				Expect(body).To(ContainSubstring("data: <div class='toast'>Saved</div>"))
			})
		},
	)

	Describe(
		"As a consumer, I want a reconnecting client to receive only the updates it missed",
		func() {
			It("replays events newer than the browser's Last-Event-ID", func() {
				rec := httptest.NewRecorder()
				r := httptest.NewRequest(http.MethodGet, "/events", nil)
				r.Header.Set("Last-Event-ID", "2")
				stream := cqrshtmx.NewSSEStream(rec, r)
				defer stream.Close()

				Expect(stream.LastEventID()).To(Equal(cqrshtmx.NewSSEEventID("2")))

				store := &memoryEventStore{events: []cqrshtmx.SSEEvent{
					{Event: eventItem, Data: "first", ID: cqrshtmx.NewSSEEventID("1")},
					{Event: eventItem, Data: "second", ID: cqrshtmx.NewSSEEventID("2")},
					{Event: eventItem, Data: "third", ID: cqrshtmx.NewSSEEventID("3")},
					{Event: eventItem, Data: "fourth", ID: cqrshtmx.NewSSEEventID("4")},
				}}

				n, err := cqrshtmx.ReplayEvents(stream, store, stream.LastEventID())
				Expect(err).NotTo(HaveOccurred())
				Expect(n).To(Equal(2))

				body := rec.Body.String()
				Expect(body).To(ContainSubstring("id: 3"))
				Expect(body).To(ContainSubstring("data: third"))
				Expect(body).To(ContainSubstring("id: 4"))
				Expect(body).To(ContainSubstring("data: fourth"))
				Expect(body).NotTo(ContainSubstring("data: first"))
				Expect(body).NotTo(ContainSubstring("data: second"))
			})

			It("treats a connection with no Last-Event-ID as a first-time visitor", func() {
				rec := httptest.NewRecorder()
				r := httptest.NewRequest(http.MethodGet, "/events", nil)
				stream := cqrshtmx.NewSSEStream(rec, r)
				defer stream.Close()

				Expect(stream.LastEventID()).To(BeZero())
				Expect(stream.LastEventID().IsZero()).To(BeTrue())
			})

			It("records an event id and retry hint so the browser can resume and back off", func() {
				rec := httptest.NewRecorder()
				r := httptest.NewRequest(http.MethodGet, "/events", nil)
				stream := cqrshtmx.NewSSEStream(rec, r)
				defer stream.Close()

				Expect(stream.Send(cqrshtmx.SSEEvent{
					Event: eventUpdate, Data: "ping", ID: cqrshtmx.NewSSEEventID("42"), Retry: 3000,
				})).To(Succeed())

				body := rec.Body.String()
				Expect(body).To(ContainSubstring("id: 42"))
				Expect(body).To(ContainSubstring("retry: 3000"))
			})

			It("rejects a Last-Event-ID that would corrupt the SSE wire format", func() {
				_, err := cqrshtmx.ParseSSEEventID("bad\nid")
				Expect(err).To(HaveOccurred())
			})
		},
	)

	Describe("As a consumer, I want to keep idle SSE connections alive behind proxies", func() {
		It("emits comment-frame pings that browsers ignore but proxies see", func() {
			rec := newLockedRecorder()
			ctx, cancel := context.WithCancel(context.Background())
			r := httptest.NewRequestWithContext(ctx, http.MethodGet, "/events", nil)
			stream := cqrshtmx.NewSSEStream(rec, r)
			defer stream.Close()

			done := make(chan struct{})
			go func() {
				stream.Heartbeat(ctx, time.Millisecond)
				close(done)
			}()

			Eventually(rec.body).Should(ContainSubstring(": keepalive"))

			cancel()
			Eventually(done).Should(BeClosed())
		})

		It("runs cleanup callbacks when the client disconnects", func() {
			ctx, cancel := context.WithCancel(context.Background())
			r := httptest.NewRequestWithContext(ctx, http.MethodGet, "/events", nil)
			stream := cqrshtmx.NewSSEStream(httptest.NewRecorder(), r)

			cleanedUp := false
			stream.OnDisconnect(func() { cleanedUp = true })

			Expect(cleanedUp).To(BeFalse())
			cancel()
			stream.Close()
			Expect(cleanedUp).To(BeTrue())
		})
	})

	Describe(
		"As a consumer, I want a successful command to automatically notify every live client",
		func() {
			var (
				broadcaster *cqrshtmx.Broadcaster
				subscriber  <-chan cqrshtmx.SSEEvent
			)

			BeforeEach(func() {
				broadcaster = cqrshtmx.NewBroadcaster()
				subscriber = broadcaster.Subscribe()
			})

			AfterEach(func() { broadcaster.Unsubscribe(subscriber) })

			It("broadcasts an SSE event to subscribers when a command succeeds", func() {
				disp := command.NewDispatcher()
				_ = disp.Register("CreateUser", noOpCommandHandler)
				app, err := cqrshtmx.New(cqrshtmx.Config{
					Commands:      disp,
					AfterDispatch: broadcaster.BroadcastOnSuccess("itemCreated", "<div>new</div>"),
				})
				Expect(err).NotTo(HaveOccurred())

				w := serve(
					app.Command("CreateUser", decodeBDDCreateUserJSON()),
					newPostRequest("/items", "{}"),
				)
				Expect(w.code()).To(Equal(http.StatusNoContent))

				Expect(subscriber).To(Receive(Equal(cqrshtmx.SSEEvent{
					Event: "itemCreated", Data: "<div>new</div>",
				})))
			})

			It("does not notify clients when the command fails", func() {
				disp := command.NewDispatcher()
				_ = disp.Register("CreateUser", rejectionHandler("item.invalid", "rejected"))
				app, err := cqrshtmx.New(cqrshtmx.Config{
					Commands:      disp,
					AfterDispatch: broadcaster.BroadcastOnSuccess("itemCreated", "<div>new</div>"),
				})
				Expect(err).NotTo(HaveOccurred())

				w := serve(
					app.Command("CreateUser", decodeBDDCreateUserJSON()),
					newPostRequest("/items", "{}"),
				)
				Expect(w.code()).To(Equal(http.StatusBadRequest))

				Consistently(subscriber, 100*time.Millisecond).
					ShouldNot(Receive())
			})
		},
	)

	Describe(
		"As a consumer, I want to parse HTMX WebSocket form submissions into typed Go structs",
		func() {
			It("separates the form body from the HTMX HEADERS blob", func() {
				raw := []byte(`{
					"message": "hello",
					"HEADERS": {"HX-Request": "true", "HX-Trigger": "send-btn"}
				}`)

				msg, err := cqrshtmx.ParseWSMessage(raw)
				Expect(err).NotTo(HaveOccurred())

				Expect(msg.StringBody("message")).To(Equal("hello"))
				Expect(msg.Headers["HX-Request"]).To(Equal("true"))
				Expect(msg.Headers["HX-Trigger"]).To(Equal("send-btn"))
				Expect(msg.Body).NotTo(HaveKey("HEADERS"))
			})

			It("deserializes body fields into a typed struct while keeping headers", func() {
				type chatMessage struct {
					Room    string `json:"room"`
					Message string `json:"message"`
				}
				raw := []byte(`{
					"room": "general",
					"message": "ping",
					"HEADERS": {"HX-Request": "true"}
				}`)

				msg, headers, err := cqrshtmx.ParseWSMessageInto[chatMessage](raw)
				Expect(err).NotTo(HaveOccurred())

				Expect(msg.Room).To(Equal("general"))
				Expect(msg.Message).To(Equal("ping"))
				Expect(headers["HX-Request"]).To(Equal("true"))
			})

			It("wraps a fragment for out-of-band swap so the browser updates a target element", func() {
				html := cqrshtmx.WSOOBHTML("notifications", "<span>3 unread</span>")

				Expect(html).To(ContainSubstring(`id="notifications"`))
				Expect(html).To(ContainSubstring(`hx-swap-oob`))
				Expect(html).To(ContainSubstring("<span>3 unread</span>"))
			})
		},
	)
})
