package cqrshtmx_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
)

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

func ExampleHTMXScriptHandler() {
	mux := http.NewServeMux()
	mux.Handle("/static/htmx.js", cqrshtmx.HTMXScriptHandler())
	fmt.Println("mounted")
	// Output: mounted
}

func ExampleHTMXScriptHandlerWith() {
	customJS := []byte("// my custom htmx build")
	mux := http.NewServeMux()
	mux.Handle("/static/htmx.js", cqrshtmx.HTMXScriptHandlerWith(customJS, "4.0.0"))
	fmt.Println("mounted")
	// Output: mounted
}

func ExampleHTMXCDNScriptTag() {
	fmt.Println(cqrshtmx.HTMXCDNScriptTag(""))
	fmt.Println(cqrshtmx.HTMXCDNScriptTag("4.0.0"))
	// Output: <script src="https://unpkg.com/htmx.org@2.0.10"></script>
	// <script src="https://unpkg.com/htmx.org@4.0.0"></script>
}

func ExampleOOBHTML() {
	fmt.Println(cqrshtmx.OOBHTML("counter", "<span>3</span>"))
	fmt.Println(cqrshtmx.OOBHTML("list", "<li>item</li>", cqrshtmx.SwapBeforeEnd))
	// Output: <div id="counter" hx-swap-oob="true"><span>3</span></div>
	// <div id="list" hx-swap-oob="beforeend"><li>item</li></div>
}
