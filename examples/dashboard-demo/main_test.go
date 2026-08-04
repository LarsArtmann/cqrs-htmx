package main

import (
	"net/http"
	"testing"
)

func TestRouteRegistrationDoesNotPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("route registration panicked: %v", r)
		}
	}()

	mux := http.NewServeMux()

	// Reproduce the exact patterns from main(). A method-specific "GET /"
	// would conflict with the "/dashboard/" subtree and panic on Go 1.22+.
	// "GET /{$}" (anchored) is safe.
	mux.Handle("/dashboard/", http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	mux.HandleFunc("GET /{$}", func(http.ResponseWriter, *http.Request) {})
}
