package cqrshtmx_test

import (
	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/go-codec"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type payloadUser struct {
	Name string `json:"name"`
}

var _ = Describe("Payload decoding", func() {
	newEvent := func(payload []byte) event.Event {
		evt, err := event.NewEvent("UserCreated", id.NewStreamID(), "User", 1, payload)
		Expect(err).NotTo(HaveOccurred())

		return evt
	}

	It("DecodePayloadAuto decodes a JSON payload without a codec argument", func() {
		evt := newEvent([]byte(`{"name":"Alice"}`))

		user, err := cqrshtmx.DecodePayloadAuto[payloadUser](evt)
		Expect(err).NotTo(HaveOccurred())
		Expect(user.Name).To(Equal("Alice"))
	})

	It("DecodePayload decodes with an explicit JSON codec", func() {
		evt := newEvent([]byte(`{"name":"Bob"}`))

		user, err := cqrshtmx.DecodePayload[payloadUser](evt, codec.JSONCodec{})
		Expect(err).NotTo(HaveOccurred())
		Expect(user.Name).To(Equal("Bob"))
	})

	It("DecodePayloads decodes a batch of events", func() {
		events := []event.Event{
			newEvent([]byte(`{"name":"A"}`)),
			newEvent([]byte(`{"name":"B"}`)),
		}

		users, err := cqrshtmx.DecodePayloads[payloadUser](events, codec.JSONCodec{})
		Expect(err).NotTo(HaveOccurred())
		Expect(users).To(HaveLen(2))
		Expect(users[1].Name).To(Equal("B"))
	})

	It("DecodePayloads fails on the first undecodable payload", func() {
		events := []event.Event{
			newEvent([]byte(`{"name":"A"}`)),
			newEvent([]byte(`{not-json`)),
		}

		_, err := cqrshtmx.DecodePayloads[payloadUser](events, codec.JSONCodec{})
		Expect(err).To(HaveOccurred())
	})
})
