package cqrshtmx_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v2"
)

func ExampleRecommendedHSTS() {
	handler := cqrshtmx.SecurityHeadersMiddlewareWithConfig(cqrshtmx.SecurityHeadersConfig{
		StrictTransportSecurity: cqrshtmx.RecommendedHSTS,
	})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	handler.ServeHTTP(w, r)
	fmt.Println(w.Header().Get("Strict-Transport-Security") != "")
	// Output: true
}
