package cqrshtmx_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
)

func ExampleRecommendedHSTS() {
	handler := cqrshtmx.SecurityHeadersMiddlewareWithConfig(cqrshtmx.SecurityHeadersConfig{
		StrictTransportSecurity: cqrshtmx.RecommendedHSTS,
	})(okHandler())

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	handler.ServeHTTP(w, r)
	fmt.Println(w.Header().Get("Strict-Transport-Security") != "")
	// Output: true
}
