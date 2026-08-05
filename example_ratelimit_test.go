package cqrshtmx_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/larsartmann/httputil"
)

func ExampleRateLimiterMiddleware() {
	mux := http.NewServeMux()

	// Allow 10 requests per minute per IP address.
	limited := httputil.KeyedRateLimiterMiddleware(httputil.KeyedRateLimiterConfig{
		Limit:        10,
		Window:       time.Minute,
		KeyExtractor: httputil.KeyExtractorFromRemoteAddr(),
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
