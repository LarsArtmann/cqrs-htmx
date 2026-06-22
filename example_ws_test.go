package cqrshtmx_test

import (
	"fmt"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v3"
)

func ExampleParseWSMessage() {
	data := []byte(`{"message":"hello","room":"general","HEADERS":{"HX-Request":"true"}}`)

	msg, err := cqrshtmx.ParseWSMessage(data)
	if err != nil {
		panic(err)
	}

	fmt.Println(msg.StringBody("message"), msg.StringBody("room"), msg.Headers["HX-Request"])
	// Output: hello general true
}

func ExampleWSOOBHTML() {
	html := cqrshtmx.WSOOBHTML("todos", "<ul><li>Buy milk</li></ul>")
	fmt.Println(html)
	// Output: <div id="todos" hx-swap-oob="true"><ul><li>Buy milk</li></ul></div>
}

func ExampleWSOOBHTML_swapStrategy() {
	html := cqrshtmx.WSOOBHTML("notifications", "New message", cqrshtmx.SwapBeforeEnd)
	fmt.Println(html)
	// Output: <div id="notifications" hx-swap-oob="beforeend">New message</div>
}
