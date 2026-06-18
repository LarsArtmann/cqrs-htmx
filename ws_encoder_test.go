package cqrshtmx_test

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"time"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("WS Encoder", func() {
	Describe("WriteWSMessage", func() {
		It("writes body and headers in HTMX format", func() {
			var buf bytes.Buffer
			msg := cqrshtmx.WSMessage{
				Headers: map[string]string{"HX-Request": "true"},
				Body:    map[string]any{"name": "Alice"},
			}
			err := cqrshtmx.WriteWSMessage(&buf, msg)
			Expect(err).NotTo(HaveOccurred())

			parsed, err := cqrshtmx.ParseWSMessage(buf.Bytes())
			Expect(err).NotTo(HaveOccurred())
			Expect(parsed.Headers["HX-Request"]).To(Equal("true"))
			Expect(parsed.Body["name"]).To(Equal("Alice"))
		})

		It("handles empty headers", func() {
			var buf bytes.Buffer
			msg := cqrshtmx.WSMessage{
				Headers: map[string]string{},
				Body:    map[string]any{"count": float64(42)},
			}
			err := cqrshtmx.WriteWSMessage(&buf, msg)
			Expect(err).NotTo(HaveOccurred())

			parsed, err := cqrshtmx.ParseWSMessage(buf.Bytes())
			Expect(err).NotTo(HaveOccurred())
			Expect(parsed.Body["count"]).To(Equal(float64(42)))
		})
	})

	Describe("WriteWSMessageInto", func() {
		type chatMsg struct {
			Channel string `json:"channel"`
			Text    string `json:"text"`
		}

		It("writes typed body with headers", func() {
			var buf bytes.Buffer
			err := cqrshtmx.WriteWSMessageInto(&buf, chatMsg{
				Channel: "general",
				Text:    "hello",
			}, map[string]string{"HX-Request": "true"})
			Expect(err).NotTo(HaveOccurred())

			parsed, headers, err := cqrshtmx.ParseWSMessageInto[chatMsg](buf.Bytes())
			Expect(err).NotTo(HaveOccurred())
			Expect(parsed.Channel).To(Equal("general"))
			Expect(parsed.Text).To(Equal("hello"))
			Expect(headers["HX-Request"]).To(Equal("true"))
		})

		It("round-trips through parse and write", func() {
			var buf bytes.Buffer
			original := chatMsg{Channel: "dev", Text: "ping"}

			err := cqrshtmx.WriteWSMessageInto(&buf, original, nil)
			Expect(err).NotTo(HaveOccurred())

			parsed, _, err := cqrshtmx.ParseWSMessageInto[chatMsg](buf.Bytes())
			Expect(err).NotTo(HaveOccurred())
			Expect(parsed).To(Equal(original))
		})
	})
})

var _ = Describe("WSBroadcaster", func() {
	Describe("Subscribe and Broadcast", func() {
		It("delivers messages to subscribers", func() {
			b := cqrshtmx.NewWSBroadcaster()
			ch := b.Subscribe()
			defer b.Unsubscribe(ch)

			b.Broadcast("hello")

			Eventually(ch).Should(Receive(Equal("hello")))
		})

		It("delivers to multiple subscribers", func() {
			b := cqrshtmx.NewWSBroadcaster()
			ch1 := b.Subscribe()
			ch2 := b.Subscribe()
			defer b.Unsubscribe(ch1)
			defer b.Unsubscribe(ch2)

			b.Broadcast("ping")

			Eventually(ch1).Should(Receive(Equal("ping")))
			Eventually(ch2).Should(Receive(Equal("ping")))
		})

		It("tracks subscriber count", func() {
			b := cqrshtmx.NewWSBroadcaster()
			Expect(b.SubscriberCount()).To(Equal(0))

			ch := b.Subscribe()
			Expect(b.SubscriberCount()).To(Equal(1))

			b.Unsubscribe(ch)
			Expect(b.SubscriberCount()).To(Equal(0))
		})

		It("closes channel on unsubscribe", func() {
			b := cqrshtmx.NewWSBroadcaster()
			ch := b.Subscribe()
			b.Unsubscribe(ch)

			_, ok := <-ch
			Expect(ok).To(BeFalse())
		})

		It("does not panic on unsubscribe of unknown channel", func() {
			b := cqrshtmx.NewWSBroadcaster()
			ch := make(chan string, 1)

			Expect(func() { b.Unsubscribe(ch) }).NotTo(Panic())
		})

		It("drops messages to slow consumers", func() {
			b := cqrshtmx.NewWSBroadcaster()
			ch := b.Subscribe()
			defer b.Unsubscribe(ch)

			for i := range 65 {
				b.Broadcast(string(rune('a' + i%26)))
			}

			count := 0
			draining := true
			for draining {
				select {
				case <-ch:
					count++
				default:
					draining = false
				}
			}
			Expect(count).To(BeNumerically("<=", 64))
		})

		It("supports concurrent subscribe and unsubscribe", func() {
			b := cqrshtmx.NewWSBroadcaster()
			var wg sync.WaitGroup

			for range 10 {
				wg.Add(1)
				go func() {
					defer wg.Done()
					ch := b.Subscribe()
					time.Sleep(time.Millisecond)
					b.Unsubscribe(ch)
				}()
			}

			wg.Wait()
			Expect(b.SubscriberCount()).To(Equal(0))
		})
	})

	Describe("BroadcastHTML", func() {
		It("wraps HTML with OOB swap before broadcasting", func() {
			b := cqrshtmx.NewWSBroadcaster()
			ch := b.Subscribe()
			defer b.Unsubscribe(ch)

			b.BroadcastHTML("target", "<span>updated</span>")

			var msg string
			Eventually(ch).Should(Receive(&msg))
			Expect(msg).To(ContainSubstring("hx-swap-oob"))
			Expect(msg).To(ContainSubstring(`id="target"`))
			Expect(msg).To(ContainSubstring("updated"))
		})
	})

	Describe("BroadcastOnSuccessWS", func() {
		It("broadcasts on dispatch success", func() {
			b := cqrshtmx.NewWSBroadcaster()
			ch := b.Subscribe()
			defer b.Unsubscribe(ch)

			hook := b.BroadcastOnSuccessWS("<div>done</div>")
			hook(context.Background(), nil, nil)

			Eventually(ch).Should(Receive(Equal("<div>done</div>")))
		})

		It("does not broadcast on error", func() {
			b := cqrshtmx.NewWSBroadcaster()
			ch := b.Subscribe()
			defer b.Unsubscribe(ch)

			hook := b.BroadcastOnSuccessWS("<div>done</div>")
			hook(context.Background(), nil, errors.New("fail"))

			Consistently(ch).ShouldNot(Receive())
		})
	})

	Describe("BroadcastOnErrorWS", func() {
		It("broadcasts structured error on dispatch failure", func() {
			b := cqrshtmx.NewWSBroadcaster()
			ch := b.Subscribe()
			defer b.Unsubscribe(ch)

			hook := b.BroadcastOnErrorWS()
			req := httptest.NewRequest(http.MethodPost, "/api/cmd", nil)
			hook(context.Background(), req, errors.New("validation failed"))

			var msg string
			Eventually(ch).Should(Receive(&msg))
			Expect(msg).To(ContainSubstring("validation failed"))
			Expect(msg).To(ContainSubstring("\"status\""))
		})

		It("does not broadcast on success", func() {
			b := cqrshtmx.NewWSBroadcaster()
			ch := b.Subscribe()
			defer b.Unsubscribe(ch)

			hook := b.BroadcastOnErrorWS()
			hook(context.Background(), nil, nil)

			Consistently(ch).ShouldNot(Receive())
		})
	})
})
