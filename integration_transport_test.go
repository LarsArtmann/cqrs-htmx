package cqrshtmx_test

import (
	"context"
	"encoding/json/v2"
	"net/http"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
	errorfamily "github.com/larsartmann/go-error-family"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Transport Parity Integration", func() {
	Describe("SSE error flow: HTTP dispatch failure → BroadcastOnError → subscriber", func() {
		It("broadcasts a StructuredError SSE event when an HTTP command fails", func() {
			broadcaster := cqrshtmx.NewBroadcaster()

			ch := broadcaster.Subscribe()
			defer broadcaster.Unsubscribe(ch)

			disp := command.NewDispatcher()
			_ = disp.Register("CreateUser", erroringCommandHandlerWith(
				errorfamily.NewConflict("email_taken", "email already exists"),
			))
			app, err := cqrshtmx.New(cqrshtmx.Config{
				Commands:      disp,
				AfterDispatch: broadcaster.BroadcastOnError("commandError"),
			})
			Expect(err).NotTo(HaveOccurred())

			handler := app.Command("CreateUser", decodeCreateUserJSON())
			serve(handler, newPostRequest("/users", `{"email":"dup@test.com","name":"Dup"}`))

			var event sse.Event
			Eventually(ch).Should(Receive(&event))
			Expect(event.Event).To(Equal("commandError"))

			var payload cqrshtmx.StructuredError
			Expect(json.Unmarshal([]byte(event.Data), &payload)).To(Succeed())
			Expect(payload.Detail).To(ContainSubstring("email already exists"))
			Expect(payload.Status).To(BeNumerically(">", 0))
		})

		It("broadcasts both success and error events from the same broadcaster", func() {
			broadcaster := cqrshtmx.NewBroadcaster()

			ch := broadcaster.Subscribe()
			defer broadcaster.Unsubscribe(ch)

			successCount := 0
			errorCount := 0

			disp := command.NewDispatcher()
			_ = disp.Register("CreateUser", noOpCommandHandler)

			app := cqrshtmx.MustNew(cqrshtmx.Config{
				Commands: disp,
				AfterDispatch: func(_ context.Context, r *http.Request, err error) {
					if err == nil {
						successCount++

						broadcaster.Broadcast(sse.Event{Event: "ok", Data: "success"})
					} else {
						errorCount++
						payload := cqrshtmx.NewStructuredError(err, r)
						broadcaster.Broadcast(sse.Event{
							Event: "error",
							Data:  payload.JSON(),
						})
					}
				},
			})

			handler := app.Command("CreateUser", decodeCreateUserJSON())
			serve(handler, newPostRequest("/users", `{}`))

			Eventually(ch).Should(Receive())
			Expect(successCount).To(Equal(1))
			Expect(errorCount).To(Equal(0))
		})
	})

	Describe("WS dispatch round-trip: DispatchWSCommand → hooks → WSBroadcaster", func() {
		It("broadcasts a WS message on successful command dispatch via hook", func() {
			wsBroadcaster := cqrshtmx.NewWSBroadcaster()

			ch := wsBroadcaster.Subscribe()
			defer wsBroadcaster.Unsubscribe(ch)

			disp := command.NewDispatcher()
			_ = disp.Register("CreateUser", noOpCommandHandler)

			app, err := cqrshtmx.New(cqrshtmx.Config{
				Commands:      disp,
				AfterDispatch: wsBroadcaster.BroadcastOnSuccessWS("<div hx-swap-oob='true'>updated</div>"),
			})
			Expect(err).NotTo(HaveOccurred())

			dispatchErr := app.DispatchWSCommand(
				nil,
				"CreateUser",
				wsNoOpCreateUserDecoder(),
				[]byte(`{"email":"a@b.com"}`),
			)
			Expect(dispatchErr).NotTo(HaveOccurred())

			Eventually(ch).Should(Receive(ContainSubstring("hx-swap-oob")))
		})

		It("broadcasts a WS StructuredError on failed command dispatch", func() {
			wsBroadcaster := cqrshtmx.NewWSBroadcaster()

			ch := wsBroadcaster.Subscribe()
			defer wsBroadcaster.Unsubscribe(ch)

			disp := command.NewDispatcher()
			rateLimited := cqrshtmx.WithHTTPStatus(
				errorfamily.NewRejection("rate_limited", "rate limited"), http.StatusTooManyRequests,
			)
			_ = disp.Register("CreateUser", erroringCommandHandlerWith(rateLimited))
			app, err := cqrshtmx.New(cqrshtmx.Config{
				Commands:      disp,
				AfterDispatch: wsBroadcaster.BroadcastOnErrorWS(),
			})
			Expect(err).NotTo(HaveOccurred())

			dispatchErr := app.DispatchWSCommand(nil, "CreateUser", wsNoOpCreateUserDecoder(), []byte(`{}`))
			Expect(dispatchErr).To(HaveOccurred())

			var msg string
			Eventually(ch).Should(Receive(&msg))

			var payload cqrshtmx.StructuredError
			Expect(json.Unmarshal([]byte(msg), &payload)).To(Succeed())
			Expect(payload.Detail).To(ContainSubstring("rate limited"))
		})

		It("dispatches a WS query and returns typed result", func() {
			disp := query.NewDispatcher()
			_ = disp.Register("GetUser", func(_ context.Context, _ query.Query) (any, error) {
				return map[string]any{"id": "123", "name": "Alice"}, nil
			})

			app := cqrshtmx.MustNew(cqrshtmx.Config{Queries: disp})

			decoder := cqrshtmx.DecodeWSJSONQuery(func(_ testGetUserQuery) (query.Query, error) {
				return &testGetUserQuery{}, nil
			})

			result, err := app.DispatchWSQuery(nil, "GetUser", decoder, []byte(`{}`))
			Expect(err).NotTo(HaveOccurred())

			resultMap, ok := result.(map[string]any)
			Expect(ok).To(BeTrue())
			Expect(resultMap["name"]).To(Equal("Alice"))
		})

		It("handles nil request gracefully without panicking", func() {
			disp := command.NewDispatcher()
			_ = disp.Register("CreateUser", noOpCommandHandler)
			app := cqrshtmx.MustNew(cqrshtmx.Config{Commands: disp})

			Expect(func() {
				_ = app.DispatchWSCommand(nil, "CreateUser", wsNoOpCreateUserDecoder(), []byte(`{}`))
			}).NotTo(Panic())
		})
	})

	Describe("WS message encode/decode round-trip with dispatch", func() {
		It("parses HTMX WS message, dispatches command, encodes response", func() {
			var capturedEmail string

			disp := command.NewDispatcher()
			_ = disp.Register("CreateUser", func(_ context.Context, cmd command.Command) error {
				typed, ok := cmd.(*testCreateUserCmd)
				if ok {
					capturedEmail = typed.email
				}

				return nil
			})

			app := cqrshtmx.MustNew(cqrshtmx.Config{Commands: disp})

			htmxMsg := `{"HEADERS":{"HX-Request":"true"},"email":"roundtrip@test.com","name":"RT"}`
			msg, _, err := cqrshtmx.ParseWSMessageInto[testCreateUserRequest]([]byte(htmxMsg))
			Expect(err).NotTo(HaveOccurred())
			Expect(msg.Email).To(Equal("roundtrip@test.com"))

			cmdJSON, _ := json.Marshal(msg)
			err = app.DispatchWSCommand(nil, "CreateUser", wsCreateUserDecoder(), cmdJSON)
			Expect(err).NotTo(HaveOccurred())
			Expect(capturedEmail).To(Equal("roundtrip@test.com"))
		})
	})
})
