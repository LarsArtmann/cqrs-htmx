package cqrshtmx_test

import (
	cqrshtmx "github.com/larsartmann/cqrs-htmx/v3"
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

		DescribeTable(
			"returns empty string for non-string StringBody fields",
			func(payload, field string) {
				msg, _ := cqrshtmx.ParseWSMessage([]byte(payload))
				Expect(msg.StringBody(field)).To(BeEmpty())
			},
			Entry("missing field", `{}`, "missing"),
			Entry("non-string field", `{"count":42}`, "count"),
		)
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

	Describe("ParseWSMessageInto", func() {
		type chatMsg struct {
			Room    string `json:"room"`
			Message string `json:"chat_message"`
		}

		It("parses into typed struct with headers", func() {
			data := []byte(`{
				"chat_message": "hello",
				"room": "general",
				"HEADERS": {"HX-Request": "true", "HX-Trigger": "send-btn"}
			}`)

			msg, headers, err := cqrshtmx.ParseWSMessageInto[chatMsg](data)
			Expect(err).NotTo(HaveOccurred())
			Expect(msg.Message).To(Equal("hello"))
			Expect(msg.Room).To(Equal("general"))
			Expect(headers).To(HaveLen(2))
			Expect(headers["HX-Request"]).To(Equal("true"))
			Expect(headers["HX-Trigger"]).To(Equal("send-btn"))
		})

		It("parses without HEADERS", func() {
			data := []byte(`{"chat_message": "hi", "room": "dev"}`)

			msg, headers, err := cqrshtmx.ParseWSMessageInto[chatMsg](data)
			Expect(err).NotTo(HaveOccurred())
			Expect(msg.Message).To(Equal("hi"))
			Expect(msg.Room).To(Equal("dev"))
			Expect(headers).To(BeEmpty())
		})

		It("returns error for invalid JSON", func() {
			_, _, err := cqrshtmx.ParseWSMessageInto[chatMsg]([]byte(`not json`))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("parse ws message into"))
		})

		It("handles non-string header values", func() {
			data := []byte(`{"chat_message":"hi","room":"dev","HEADERS":{"HX-Request":true,"num":42}}`)

			msg, headers, err := cqrshtmx.ParseWSMessageInto[chatMsg](data)
			Expect(err).NotTo(HaveOccurred())
			Expect(msg.Message).To(Equal("hi"))
			Expect(headers).To(BeEmpty())
		})

		It("preserves numeric body fields in typed struct", func() {
			type countMsg struct {
				Name  string `json:"name"`
				Count int    `json:"count"`
			}

			data := []byte(`{"name":"test","count":42}`)

			msg, _, err := cqrshtmx.ParseWSMessageInto[countMsg](data)
			Expect(err).NotTo(HaveOccurred())
			Expect(msg.Name).To(Equal("test"))
			Expect(msg.Count).To(Equal(42))
		})

		It("returns error when body cannot be unmarshaled into target type", func() {
			type strictMsg struct {
				Count int `json:"count"`
			}

			// "count" is a string but the target expects an int.
			data := []byte(`{"count":"not-a-number"}`)

			_, _, err := cqrshtmx.ParseWSMessageInto[strictMsg](data)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("parse ws message into"))
		})

		It("handles non-string typed body field", func() {
			type flexibleMsg struct {
				Count int `json:"count"`
			}

			data := []byte(`{"count":42,"HEADERS":{"HX-Request":"true"}}`)

			msg, headers, err := cqrshtmx.ParseWSMessageInto[flexibleMsg](data)
			Expect(err).NotTo(HaveOccurred())
			Expect(msg.Count).To(Equal(42))
			Expect(headers).To(HaveLen(1))
		})
	})

	Describe("parseWSHeaders (via ParseWSMessageInto)", func() {
		It("handles HEADERS with non-string non-numeric values gracefully", func() {
			data := []byte(`{"HEADERS":{"foo":null,"bar":true,"baz":"x"}}`)

			_, headers, err := cqrshtmx.ParseWSMessageInto[map[string]any](data)
			Expect(err).NotTo(HaveOccurred())
			// Only the string-valued header survives; non-strings are dropped.
			Expect(headers).To(HaveKeyWithValue("baz", "x"))
			Expect(headers).NotTo(HaveKey("foo"))
			Expect(headers).NotTo(HaveKey("bar"))
		})
	})
})
