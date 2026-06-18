package cqrshtmx_test

import (
	"net/http"
	"net/http/httptest"
	"sync"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("SSE Broadcaster and Integration", func() {
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

		It("never panics when unsubscribe races with broadcast", func() {
			// Regression test: Broadcast() must not send on a channel that a
			// concurrent Unsubscribe() has closed. Before the fix, Broadcast
			// snapshotted subscribers under RLock, released it, then iterated —
			// so a close() between release and send panicked. We hammer both
			// paths concurrently across many goroutines and cycles; any
			// send-on-closed-channel would crash the test process.
			b := cqrshtmx.NewBroadcaster()

			stop := make(chan struct{})
			var wg sync.WaitGroup

			// Many concurrent broadcasters maximize the chance of hitting the
			// snapshot-release → send window while a channel is being closed.
			for range 8 {
				wg.Go(func() {
					for {
						select {
						case <-stop:
							return
						default:
							b.Broadcast(cqrshtmx.SSEEvent{Event: "hammer", Data: "x"})
						}
					}
				})
			}

			wg.Go(func() {
				for range 4000 {
					ch := b.Subscribe()
					b.Unsubscribe(ch)
				}
				close(stop)
			})

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
})
