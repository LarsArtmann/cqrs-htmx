package cqrshtmx_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	cqrshtmx "github.com/larsartmann/cqrs-htmx"
	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/query"
)

func ExampleNew() {
	disp := command.NewDispatcher()

	app, err := cqrshtmx.New(cqrshtmx.Config{
		Commands: disp,
	})
	if err != nil {
		panic(err)
	}

	_ = app
	fmt.Println("app created")
	// Output: app created
}

func ExampleApp_Command() {
	disp := command.NewDispatcher()
	app, _ := cqrshtmx.New(cqrshtmx.Config{Commands: disp})

	handler := app.Command("CreateUser",
		cqrshtmx.DecodeJSON(func(_ struct {
			Email string `json:"email"`
		},
		) (command.Command, error) {
			return nil, nil
		}),
		cqrshtmx.Trigger("userCreated"),
		cqrshtmx.PushURL("/users"),
	)

	_ = handler
	fmt.Println("command handler created")
	// Output: command handler created
}

func ExampleApp_Query() {
	disp := query.NewDispatcher()
	app, _ := cqrshtmx.New(cqrshtmx.Config{Queries: disp})

	handler := app.Query("GetUser",
		cqrshtmx.DecodeJSONQuery(func(_ struct{}) (query.Query, error) {
			return nil, nil
		}),
		cqrshtmx.Render(func(_ http.ResponseWriter, _ *http.Request, _ any) error {
			return nil
		}),
	)

	_ = handler
	fmt.Println("query handler created")
	// Output: query handler created
}

func ExampleNewResponse() {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/items", nil)
	r.Header.Set("HX-Request", "true")

	cqrshtmx.NewResponse(w, r).
		Trigger("itemCreated").
		PushURL("/items/42").
		Retarget("#item-list").
		Reswap(cqrshtmx.SwapOuterHTML).
		NotifySuccess("Item created").
		Apply()

	fmt.Println(w.Header().Get("HX-Trigger"))
	fmt.Println(w.Header().Get("HX-Push-Url"))
	fmt.Println(w.Header().Get("HX-Retarget"))
	fmt.Println(w.Header().Get("HX-Reswap"))
	// Output: itemCreated,showMessage
	// /items/42
	// #item-list
	// outerHTML
}

func ExampleSwapStrategy() {
	strategies := []cqrshtmx.SwapStrategy{
		cqrshtmx.SwapInnerHTML,
		cqrshtmx.SwapOuterHTML,
		cqrshtmx.SwapBeforeBegin,
		cqrshtmx.SwapAfterEnd,
		cqrshtmx.SwapNone,
	}
	for _, s := range strategies {
		fmt.Println(string(s))
	}
	// Output: innerHTML
	// outerHTML
	// beforebegin
	// afterend
	// none
}

func ExampleHTMXMiddleware() {
	mux := http.NewServeMux()

	mux.Handle("/", cqrshtmx.HTMXMiddleware(
		http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			if cqrshtmx.IsHTMXRequest(r) {
				fmt.Println("HTMX request")
			}
			if cqrshtmx.RenderPartial(r) {
				fmt.Println("render partial")
			}
		}),
	))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("HX-Request", "true")
	mux.ServeHTTP(w, r)
	// Output: HTMX request
	// render partial
}

func ExampleRequestLogging() {
	mux := http.NewServeMux()

	// Log every request to stdout with the default plain-text formatter.
	logged := cqrshtmx.RequestLogging(nil, func(_ string) {
		fmt.Println("logged")
	})

	mux.Handle("/", logged(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
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
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusCreated)
		}),
	))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/items", nil)
	mux.ServeHTTP(w, r)
}

func ExampleRateLimiterMiddleware() {
	mux := http.NewServeMux()

	// Allow 10 requests per minute per IP address.
	limited := cqrshtmx.RateLimiterMiddleware(cqrshtmx.RateLimiterConfig{
		Limit:        10,
		Window:       time.Minute,
		KeyExtractor: cqrshtmx.KeyExtractorFromRemoteAddr(),
	})

	mux.Handle("/", limited(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	mux.ServeHTTP(w, r)
	fmt.Println(w.Code)
	// Output: 200
}
