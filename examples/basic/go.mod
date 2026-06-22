module github.com/larsartmann/cqrs-htmx/examples/basic

go 1.26.3

require (
	github.com/larsartmann/cqrs-htmx/v3 v3.0.0
	github.com/larsartmann/go-cqrs-lite/command/v3 v3.0.0
	github.com/larsartmann/go-cqrs-lite/id/v3 v3.0.0
	github.com/larsartmann/go-cqrs-lite/query/v3 v3.0.0
)

require (
	github.com/fxamacker/cbor/v2 v2.9.2 // indirect
	github.com/justinas/nosurf v1.2.0 // indirect
	github.com/larsartmann/go-branded-id v0.3.1 // indirect
	github.com/larsartmann/go-cqrs-lite/codec/v3 v3.0.0 // indirect
	github.com/larsartmann/go-cqrs-lite/dispatcher/v3 v3.0.0 // indirect
	github.com/larsartmann/go-cqrs-lite/event/v3 v3.0.0 // indirect
	github.com/larsartmann/go-error-family v0.4.0 // indirect
	github.com/larsartmann/httputil v0.3.0 // indirect
	github.com/oklog/ulid/v2 v2.1.1 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	golang.org/x/time v0.15.0 // indirect
)

replace github.com/larsartmann/cqrs-htmx/v3 => ../..
