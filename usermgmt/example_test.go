package usermgmt_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/larsartmann/cqrs-htmx/usermgmt/v2"
)

func ExampleNewService() {
	_, err := usermgmt.NewService(usermgmt.ServiceConfig{})
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println("service created")
	// Output: service created
}

func ExampleNewAuthHandler() {
	service, _ := usermgmt.NewService(usermgmt.ServiceConfig{})
	secure := true
	handler := usermgmt.NewAuthHandler(service, usermgmt.HandlerConfig{
		Secure: &secure,
	})

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	fmt.Println("routes registered")
	// Output: routes registered
}

func ExampleNewSessionMiddleware() {
	service, _ := usermgmt.NewService(usermgmt.ServiceConfig{})
	middleware := usermgmt.NewSessionMiddleware(service, "session_token")

	handler := middleware(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		user, ok := usermgmt.UserFromContext(r.Context())
		if ok {
			fmt.Println("user:", user.Email)
		} else {
			fmt.Println("no user")
		}
	}))

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	// Output: no user
}

func ExampleService_Register() {
	service, _ := usermgmt.NewService(usermgmt.ServiceConfig{})
	uid := usermgmt.NewUserID("01HXYZ1234567890ABCDEFGHIJKL")

	resp, err := service.Register(context.Background(), usermgmt.RegisterRequest{
		ID:          uid,
		Email:       "alice@example.com",
		Password:    "secure-password-123",
		DisplayName: "Alice",
	})
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println("user created:", resp.User.Email)
	// Output: user created: alice@example.com
}
