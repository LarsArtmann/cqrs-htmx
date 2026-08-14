module github.com/larsartmann/cqrs-htmx/examples/datastar-demo

go 1.26.5

require (
	github.com/larsartmann/cqrs-htmx/datastar/v4 v4.7.0
	github.com/larsartmann/go-cqrs-lite/command/v4 v4.6.0
	github.com/larsartmann/go-cqrs-lite/event/v4 v4.6.0
	github.com/larsartmann/go-cqrs-lite/id/v4 v4.4.0
	github.com/larsartmann/go-cqrs-lite/query/v4 v4.5.0
	github.com/larsartmann/go-error-family v0.10.0
	github.com/larsartmann/httputil v0.11.0
)

require (
	github.com/fxamacker/cbor/v2 v2.9.2 // indirect
	github.com/justinas/nosurf v1.2.0 // indirect
	github.com/larsartmann/go-branded-id v0.5.1 // indirect
	github.com/larsartmann/go-codec v0.1.0 // indirect
	github.com/larsartmann/go-cqrs-lite/codec/v4 v4.4.0 // indirect
	github.com/larsartmann/go-cqrs-lite/dispatcher/v4 v4.3.0 // indirect
	github.com/larsartmann/go-cqrs-lite/metadata/v4 v4.4.0 // indirect
	github.com/larsartmann/go-cqrs-lite/record/v4 v4.2.0 // indirect
	github.com/larsartmann/go-cqrs-lite/snapshot/v4 v4.2.0 // indirect
	github.com/larsartmann/go-datastar v0.2.0 // indirect
	github.com/larsartmann/go-datastar/static v0.2.0 // indirect
	github.com/larsartmann/go-etag v0.1.1 // indirect
	github.com/larsartmann/go-sse v0.5.0 // indirect
	github.com/oklog/ulid/v2 v2.1.2 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	golang.org/x/time v0.15.0 // indirect
)

replace github.com/larsartmann/cqrs-htmx/datastar/v4 => ../../datastar

replace github.com/larsartmann/go-datastar => ../../../go-datastar

replace github.com/larsartmann/go-sse => ../../../go-sse

replace github.com/larsartmann/go-datastar/static => ../../../go-datastar/static
