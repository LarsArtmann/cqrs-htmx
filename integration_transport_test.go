package cqrshtmx_test

import (
	"context"
	"encoding/json/v2"
	"net/http"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/larsartmann/go-sse"
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
})
