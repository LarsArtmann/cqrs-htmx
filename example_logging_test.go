package cqrshtmx_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v3"
)

func ExampleRequestLogging() {
	mux := http.NewServeMux()

	// Log every request to stdout with the default plain-text formatter.
	logged := cqrshtmx.RequestLogging(nil, func(_ string) {
		fmt.Println("logged")
	})

	mux.Handle("/", logged(
		okHandler(),
	))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/users", nil)
	mux.ServeHTTP(w, r)
	// Output: logged
}

func ExampleJSONLogFormatter() {
	mux := http.NewServeMux()

	logged := cqrshtmx.RequestLogging(cqrshtmx.JSONLogFormatter, func(line string) {
		fmt.Println(line)
	})

	mux.Handle("/", logged(
		createdHandler(),
	))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/items", nil)
	mux.ServeHTTP(w, r)
}
