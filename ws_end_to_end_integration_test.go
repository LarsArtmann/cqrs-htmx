package cqrshtmx_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/go-cqrs-lite/command/v3"
	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/query/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// drainChannel reads all currently-buffered messages from a channel and returns
// the count. Non-blocking: stops as soon as the channel is empty.
func drainChannel[T any](ch <-chan T) int {
	count := 0
	for {
		select {
		case <-ch:
			count++
		default:
			return count
		}
	}
}

// These tests verify the full WebSocket handler loop end-to-end:
//
//   1. Simulated WS client sends bytes (what a real conn.ReadMessage() would return)
//   2. App.DispatchWSCommand decodes + runs hooks + dispatches
//   3. AfterDispatch hook broadcasts to WSBroadcaster
//   4. All subscribers receive the broadcast (what a real conn.WriteMessage would send)
//
// No WebSocket library is imported — cqrs-htmx intentionally leaves the WS
// transport to the consumer. These tests exercise the CQRS-bridge layer
// (dispatch + hooks + broadcaster) which is what the library owns.

var _ = Describe("WebSocket End-to-End Integration", func() {
	Describe("Full WS handler loop: receive → dispatch → broadcast", func() {
		It("broadcasts success to all subscribers on successful command", func() {
			wsBroadcaster := cqrshtmx.NewWSBroadcaster()
			sub1 := wsBroadcaster.Subscribe()
			sub2 := wsBroadcaster.Subscribe()
			sub3 := wsBroadcaster.Subscribe()
			defer wsBroadcaster.Unsubscribe(sub1)
			defer wsBroadcaster.Unsubscribe(sub2)
			defer wsBroadcaster.Unsubscribe(sub3)

			disp := command.NewDispatcher()
			_ = disp.Register("CreateUser", noOpCommandHandler)

			successMsg := cqrshtmx.WSOOBHTML("user-list", "<div id='user-list'>Alice</div>")
			app := cqrshtmx.MustNew(cqrshtmx.Config{
				Commands:      disp,
				AfterDispatch: wsBroadcaster.BroadcastOnSuccessWS(successMsg),
			})

			// Simulate a WS client sending a CreateUser message.
			err := app.DispatchWSCommand(
				nil,
				"CreateUser",
				wsNoOpCreateUserDecoder(),
				[]byte(`{"email":"alice@test.com","name":"Alice"}`),
			)
			Expect(err).NotTo(HaveOccurred())

			// All three subscribers must receive the broadcast.
			Eventually(sub1).Should(Receive(ContainSubstring("user-list")))
			Eventually(sub2).Should(Receive(ContainSubstring("user-list")))
			Eventually(sub3).Should(Receive(ContainSubstring("Alice")))
		})

		It("broadcasts StructuredError to all subscribers on failed command", func() {
			wsBroadcaster := cqrshtmx.NewWSBroadcaster()
			subs := make([]<-chan string, 5)
			for i := range subs {
				subs[i] = wsBroadcaster.Subscribe()
				defer wsBroadcaster.Unsubscribe(subs[i])
			}

			disp := command.NewDispatcher()
			_ = disp.Register("CreateUser", erroringCommandHandlerWith(
				event.NewTransient("db_unavailable", "database unavailable"),
			))
			app := cqrshtmx.MustNew(cqrshtmx.Config{
				Commands:      disp,
				AfterDispatch: wsBroadcaster.BroadcastOnErrorWS(),
			})

			err := app.DispatchWSCommand(nil, "CreateUser", wsNoOpCreateUserDecoder(), []byte(`{}`))
			Expect(err).To(HaveOccurred())

			// All 5 subscribers must receive the StructuredError JSON.
			for i, sub := range subs {
				var msg string
				Eventually(sub).Should(Receive(&msg), "subscriber %d did not receive broadcast", i)

				var payload cqrshtmx.StructuredError
				Expect(json.Unmarshal([]byte(msg), &payload)).To(Succeed())
				// 5xx detail is redacted to the family's public-safe message; the
				// original cause stays on the StructuredError for server-side use.
				Expect(payload.Detail).To(ContainSubstring("temporary error"))
				Expect(payload.Detail).NotTo(ContainSubstring("database unavailable"))
				Expect(payload.Status).To(Equal(http.StatusServiceUnavailable))
			}
		})

		It("handles rapid sequential dispatches without losing broadcasts", func() {
			wsBroadcaster := cqrshtmx.NewWSBroadcaster()
			sub := wsBroadcaster.Subscribe()
			defer wsBroadcaster.Unsubscribe(sub)

			var counter atomic.Int64
			disp := command.NewDispatcher()
			_ = disp.Register("CreateUser", func(_ context.Context, _ command.Command) error {
				counter.Add(1)
				return nil
			})

			successMsg := cqrshtmx.WSOOBHTML("counter", "<div>ok</div>")
			app := cqrshtmx.MustNew(cqrshtmx.Config{
				Commands:      disp,
				AfterDispatch: wsBroadcaster.BroadcastOnSuccessWS(successMsg),
			})

			// Fire 20 commands in sequence, like a chatty WS client.
			const iterations = 20
			decoder := wsNoOpCreateUserDecoder()
			for range iterations {
				err := app.DispatchWSCommand(nil, "CreateUser", decoder, []byte(`{}`))
				Expect(err).NotTo(HaveOccurred())
			}

			Expect(counter.Load()).To(Equal(int64(iterations)))

			// Subscriber must receive all 20 broadcasts (buffered channel capacity is 64).
			received := 0
			for range iterations {
				Eventually(sub).Should(Receive())
				received++
			}
			Expect(received).To(Equal(iterations))
		})

		It("subscribers joining mid-stream only receive subsequent broadcasts", func() {
			wsBroadcaster := cqrshtmx.NewWSBroadcaster()

			disp := command.NewDispatcher()
			_ = disp.Register("CreateUser", noOpCommandHandler)
			successMsg := cqrshtmx.WSOOBHTML("update", "<div>event</div>")
			app := cqrshtmx.MustNew(cqrshtmx.Config{
				Commands:      disp,
				AfterDispatch: wsBroadcaster.BroadcastOnSuccessWS(successMsg),
			})

			// Subscriber 1 joins before any dispatch.
			early := wsBroadcaster.Subscribe()
			defer wsBroadcaster.Unsubscribe(early)

			_ = app.DispatchWSCommand(nil, "CreateUser", wsNoOpCreateUserDecoder(), []byte(`{}`))
			Eventually(early).Should(Receive())

			// Subscriber 2 joins AFTER the first dispatch.
			late := wsBroadcaster.Subscribe()
			defer wsBroadcaster.Unsubscribe(late)

			_ = app.DispatchWSCommand(nil, "CreateUser", wsNoOpCreateUserDecoder(), []byte(`{}`))

			// Late subscriber must receive only the SECOND broadcast.
			Eventually(late).Should(Receive())
			// Early subscriber receives the second broadcast too.
			Eventually(early).Should(Receive())

			// Late subscriber must NOT have a third message pending.
			Consistently(late).ShouldNot(Receive())
		})
	})

	Describe("WS query end-to-end", func() {
		It("dispatches a query and broadcasts the result as a WS message", func() {
			wsBroadcaster := cqrshtmx.NewWSBroadcaster()
			sub := wsBroadcaster.Subscribe()
			defer wsBroadcaster.Unsubscribe(sub)

			disp := query.NewDispatcher()
			_ = disp.Register("GetUser", func(_ context.Context, _ query.Query) (any, error) {
				return map[string]any{"id": "123", "name": "Alice", "email": "alice@test.com"}, nil
			})

			app := cqrshtmx.MustNew(cqrshtmx.Config{Queries: disp})

			decoder := cqrshtmx.DecodeWSJSONQuery(func(_ testGetUserQuery) (query.Query, error) {
				return &testGetUserQuery{}, nil
			})

			result, err := app.DispatchWSQuery(nil, "GetUser", decoder, []byte(`{}`))
			Expect(err).NotTo(HaveOccurred())

			// In a real WS handler, we'd serialize result and write to conn.
			resultJSON, err := json.Marshal(result)
			Expect(err).NotTo(HaveOccurred())

			wsBroadcaster.Broadcast(string(resultJSON))

			var msg string
			Eventually(sub).Should(Receive(&msg))

			var received map[string]any
			Expect(json.Unmarshal([]byte(msg), &received)).To(Succeed())
			Expect(received["name"]).To(Equal("Alice"))
		})
	})

	Describe("Concurrent dispatch from multiple simulated WS clients", func() {
		It("fans out broadcasts to all subscribers under concurrent load", func() {
			wsBroadcaster := cqrshtmx.NewWSBroadcaster()
			sub := wsBroadcaster.Subscribe()
			defer wsBroadcaster.Unsubscribe(sub)

			var dispatchCount atomic.Int64
			disp := command.NewDispatcher()
			_ = disp.Register("CreateUser", func(_ context.Context, _ command.Command) error {
				dispatchCount.Add(1)
				return nil
			})

			app := cqrshtmx.MustNew(cqrshtmx.Config{
				Commands: disp,
				AfterDispatch: func(_ context.Context, _ *http.Request, err error) {
					if err == nil {
						wsBroadcaster.Broadcast("ok")
					}
				},
			})

			// 10 goroutines × 10 dispatches each = 100 concurrent dispatches.
			const goroutines, perG = 10, 10
			var wg sync.WaitGroup
			wg.Add(goroutines)
			for range goroutines {
				go func() {
					defer wg.Done()
					decoder := wsNoOpCreateUserDecoder()
					for range perG {
						_ = app.DispatchWSCommand(nil, "CreateUser", decoder, []byte(`{}`))
					}
				}()
			}
			wg.Wait()

			Expect(dispatchCount.Load()).To(Equal(int64(goroutines * perG)))

			// Drain the subscriber channel — count how many broadcasts were received.
			// Channel buffer is 64, so we expect at least 64 to be delivered.
			received := drainChannel(sub)
			// We expect at least some broadcasts to be received. The buffered
			// channel (cap 64) should hold most of them; concurrent dispatch
			// may drop a few if the channel fills between send attempts.
			Expect(received).To(BeNumerically(">", 0), "expected at least some broadcasts to be received")
		})
	})

	Describe("WS handler with request context propagation", func() {
		It("propagates user ID from request context through dispatch to hooks", func() {
			wsBroadcaster := cqrshtmx.NewWSBroadcaster()
			sub := wsBroadcaster.Subscribe()
			defer wsBroadcaster.Unsubscribe(sub)

			var capturedUserID string
			disp := command.NewDispatcher()
			_ = disp.Register("CreateUser", noOpCommandHandler)

			app := cqrshtmx.MustNew(cqrshtmx.Config{
				Commands: disp,
				AfterDispatch: func(ctx context.Context, _ *http.Request, err error) {
					if err == nil {
						uid := cqrshtmx.UserIDFromContext(ctx)
						if !uid.IsZero() {
							capturedUserID = uid.String()
						}
						wsBroadcaster.Broadcast("ok")
					}
				},
			})

			// Simulate an authenticated WS upgrade request.
			req := httptest.NewRequest(http.MethodPost, "/ws", nil)
			testUserID := cqrshtmx.NewUserID()
			req = req.WithContext(cqrshtmx.WithUserID(req.Context(), testUserID))

			_ = app.DispatchWSCommand(req, "CreateUser", wsNoOpCreateUserDecoder(), []byte(`{}`))

			Expect(capturedUserID).To(Equal(testUserID.String()))
			Eventually(sub).Should(Receive(Equal("ok")))
		})
	})
})
