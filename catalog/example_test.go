package cataloghtmx_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	cataloghtmx "github.com/larsartmann/cqrs-htmx/catalog/v2"
	"github.com/larsartmann/go-cqrs-lite/catalog/v3"
)

type RegisterUserCmd struct {
	Email       string `json:"email"        doc:"User email address"`
	DisplayName string `json:"display_name" doc:"Display name"`
}

type UserRegisteredEvent struct {
	UserID string `json:"user_id" doc:"The new user ID"`
	Email  string `json:"email"   doc:"User email"`
}

type GetUserQuery struct {
	ID string `json:"id" doc:"User ID to look up"`
}

// ExampleNew demonstrates building a catalog and serving it as OpenAPI.
func ExampleNew() {
	b := cataloghtmx.New("User Service", "1.0.0")

	cataloghtmx.Command[RegisterUserCmd](
		b, "register-user",
		cataloghtmx.WithOperation("POST", "/api/users"),
	)

	cataloghtmx.Event[UserRegisteredEvent](b, "user.registered", catalog.Sends)

	cataloghtmx.Query[GetUserQuery](
		b, "get-user",
		cataloghtmx.WithOperation("GET", "/api/users/{id}"),
	)

	cat := b.Build()

	// Serve as OpenAPI JSON
	mux := http.NewServeMux()
	mux.Handle("/docs/openapi.json", cataloghtmx.OpenAPIHandler(cat))
	mux.Handle("/docs/asyncapi.json", cataloghtmx.AsyncAPIHandler(cat))
	mux.Handle("/docs/diagram.d2", cataloghtmx.D2Handler(cat))

	_ = mux // Wire into your server

	fmt.Println("Services:", len(cat.Services))
	fmt.Println("Commands:", len(cat.Services[0].Commands))
	fmt.Println("Events:", len(cat.Services[0].Events))
	fmt.Println("Queries:", len(cat.Services[0].Queries))

	// Output:
	// Services: 1
	// Commands: 1
	// Events: 1
	// Queries: 1
}

// ExampleOpenAPIHandler demonstrates serving OpenAPI documentation.
func ExampleOpenAPIHandler() {
	b := cataloghtmx.New("My API", "2.0.0")
	cataloghtmx.Command[RegisterUserCmd](b, "register-user")
	cat := b.Build()

	handler := cataloghtmx.OpenAPIHandler(cat)

	req := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	fmt.Println("Status:", w.Code)
	fmt.Println("Content-Type:", w.Header().Get("Content-Type"))

	// Output:
	// Status: 200
	// Content-Type: application/json; charset=utf-8
}
