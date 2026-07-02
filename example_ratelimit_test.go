package cqrshtmx_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
)

func ExampleRateLimiterMiddleware() {
	mux := http.NewServeMux()

	// Allow 10 requests per minute per IP address.
	limited := cqrshtmx.RateLimiterMiddleware(cqrshtmx.RateLimiterConfig{
		Limit:        10,
		Window:       time.Minute,
		KeyExtractor: cqrshtmx.KeyExtractorFromRemoteAddr(),
	})

	mux.Handle("/", limited(
		okHandler(),
	))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	mux.ServeHTTP(w, r)
	fmt.Println(w.Code)
	// Output: 200
}
