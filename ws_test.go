package cqrshtmx_test

import (
	cqrshtmx "github.com/larsartmann/cqrs-htmx/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("WebSocket Protocol Helpers", func() {
	Describe("ParseWSMessage", func() {
		It("parses a simple form submission", func() {
			msg, err := cqrshtmx.ParseWSMessage([]byte(`{"message":"hello","HEADERS":{"HX-Request":"true"}}`))
			Expect(err).NotTo(HaveOccurred())
			Expect(msg.Body).To(HaveKeyWithValue("message", "hello"))
			Expect(msg.Headers).To(HaveKeyWithValue("HX-Request", "true"))
		})

		It("separates HEADERS from body fields", func() {
			msg, err := cqrshtmx.ParseWSMessage([]byte(`{
				"chat_message": "hi there",
				"room": "general",
				"HEADERS": {
					"HX-Request": "true",
					"HX-Trigger": "send-btn",
					"HX-Target": "messages"
				}
			}`))
			Expect(err).NotTo(HaveOccurred())
			Expect(msg.Body).To(HaveLen(2))
			Expect(msg.Body["chat_message"]).To(Equal("hi there"))
			Expect(msg.Body["room"]).To(Equal("general"))
			Expect(msg.Headers).To(HaveLen(3))
			Expect(msg.Headers["HX-Request"]).To(Equal("true"))
			Expect(msg.Headers["HX-Trigger"]).To(Equal("send-btn"))
		})

		It("handles messages without HEADERS", func() {
			msg, err := cqrshtmx.ParseWSMessage([]byte(`{"field1":"value1"}`))
			Expect(err).NotTo(HaveOccurred())
			Expect(msg.Body).To(HaveKeyWithValue("field1", "value1"))
			Expect(msg.Headers).To(BeEmpty())
		})

		It("handles empty JSON object", func() {
			msg, err := cqrshtmx.ParseWSMessage([]byte(`{}`))
			Expect(err).NotTo(HaveOccurred())
			Expect(msg.Body).To(BeEmpty())
			Expect(msg.Headers).To(BeEmpty())
		})

		It("returns error for invalid JSON", func() {
			_, err := cqrshtmx.ParseWSMessage([]byte(`not json`))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("parse ws message"))
		})

		It("handles non-string header values gracefully", func() {
			msg, err := cqrshtmx.ParseWSMessage([]byte(`{"HEADERS":{"HX-Request":true,"num":42}}`))
			Expect(err).NotTo(HaveOccurred())
			Expect(msg.Headers).To(BeEmpty())
		})

		It("handles numeric body values", func() {
			msg, err := cqrshtmx.ParseWSMessage([]byte(`{"count":42}`))
			Expect(err).NotTo(HaveOccurred())
			Expect(msg.Body["count"]).To(Equal(float64(42)))
		})
	})

	Describe("WSMessage.StringBody", func() {
		It("returns string body field", func() {
			msg, _ := cqrshtmx.ParseWSMessage([]byte(`{"name":"Alice"}`))
			Expect(msg.StringBody("name")).To(Equal("Alice"))
		})

		It("returns empty string for missing field", func() {
			msg, _ := cqrshtmx.ParseWSMessage([]byte(`{}`))
			Expect(msg.StringBody("missing")).To(BeEmpty())
		})

		It("returns empty string for non-string field", func() {
			msg, _ := cqrshtmx.ParseWSMessage([]byte(`{"count":42}`))
			Expect(msg.StringBody("count")).To(BeEmpty())
		})
	})

	Describe("WSOOBHTML", func() {
		It("wraps HTML with default OOB swap", func() {
			result := cqrshtmx.WSOOBHTML("todos", "<ul><li>Buy milk</li></ul>")
			Expect(result).To(Equal(
				`<div id="todos" hx-swap-oob="true"><ul><li>Buy milk</li></ul></div>`,
			))
		})

		It("uses custom swap strategy", func() {
			result := cqrshtmx.WSOOBHTML("notifications", "New message",
				cqrshtmx.SwapBeforeEnd)
			Expect(result).To(ContainSubstring(`hx-swap-oob="beforeend"`))
		})

		It("passes through HTML that already has hx-swap-oob", func() {
			html := `<div id="items" hx-swap-oob="morphdom">...</div>`
			result := cqrshtmx.WSOOBHTML("items", html)
			Expect(result).To(Equal(html))
		})

		It("handles empty content", func() {
			result := cqrshtmx.WSOOBHTML("target", "")
			Expect(result).To(Equal(`<div id="target" hx-swap-oob="true"></div>`))
		})
	})
})
