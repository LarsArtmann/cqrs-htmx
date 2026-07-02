package cqrshtmx_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"time"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("SSE Stream Concurrency", func() {
	// This test guards the critical fix for the Send()+Heartbeat() data race
	// on http.ResponseWriter (sse_stream.go). http.ResponseWriter is not safe
	// for concurrent use; the stream serializes both writers on a mutex.
	// Run under `go test -race` to catch regressions.
	It("does not race when Send and Heartbeat write concurrently", func() {
		for range 50 {
			ctx, cancel := context.WithCancel(context.Background())
			r := httptest.NewRequest(http.MethodGet, "/events", nil).WithContext(ctx)
			w := httptest.NewRecorder()

			stream := cqrshtmx.NewSSEStream(w, r)
			Expect(stream.Context()).To(Equal(ctx))

			var wg sync.WaitGroup
			wg.Add(2)

			go func() {
				defer wg.Done()
				stream.Heartbeat(ctx, time.Microsecond)
			}()
			go func() {
				defer wg.Done()
				defer cancel()
				for range 20 {
					_ = stream.Send(cqrshtmx.SSEEvent{Event: "ping", Data: "x"})
				}
			}()

			wg.Wait()
			stream.Close()
		}
	})
})
