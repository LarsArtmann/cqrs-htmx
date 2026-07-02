package cqrshtmx_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/go-cqrs-lite/command/v3"
)

func ExampleRequireMethod() {
	disp := command.NewDispatcher()
	app := cqrshtmx.MustNew(cqrshtmx.Config{Commands: disp})

	handler := app.Command("CreateUser", cqrshtmx.RequireMethod(http.MethodPost))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/users", nil)
	handler.ServeHTTP(w, r)
	fmt.Println(w.Code)
	// Output: 405
}

func ExampleResponse_JSON() {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	cqrshtmx.NewResponse(w, r).JSON(map[string]string{"s": "ok"})
	fmt.Println(w.Header().Get("Content-Type"))
	// Output: application/json; charset=utf-8
}
