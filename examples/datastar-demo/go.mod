module github.com/larsartmann/cqrs-htmx/examples/datastar-demo

go 1.26.3

require (
	github.com/larsartmann/go-cqrs-lite/command/v2 v2.0.0
	github.com/larsartmann/go-cqrs-lite/id/v2 v2.0.0
	github.com/larsartmann/go-cqrs-lite/query/v2 v2.0.0
	github.com/starfederation/datastar-go v1.2.1
)

require (
	github.com/CAFxX/httpcompression v0.0.9 // indirect
	github.com/andybalholm/brotli v1.2.1 // indirect
	github.com/klauspost/compress v1.18.6 // indirect
	github.com/larsartmann/go-branded-id v0.3.0 // indirect
	github.com/larsartmann/go-cqrs-lite/codec/v2 v2.0.0 // indirect
	github.com/larsartmann/go-cqrs-lite/dispatcher/v2 v2.0.0 // indirect
	github.com/larsartmann/go-cqrs-lite/event/v2 v2.0.0 // indirect
	github.com/larsartmann/go-error-family v0.3.0 // indirect
	github.com/oklog/ulid/v2 v2.1.1 // indirect
	github.com/samber/lo v1.53.0 // indirect
	github.com/samber/ro v0.3.0 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	golang.org/x/exp v0.0.0-20260529124908-c761662dc8c9 // indirect
	golang.org/x/text v0.37.0 // indirect
)

replace (
	github.com/larsartmann/go-cqrs-lite/codec/v2 => ../../../go-cqrs-lite/codec
	github.com/larsartmann/go-cqrs-lite/command/v2 => ../../../go-cqrs-lite/command
	github.com/larsartmann/go-cqrs-lite/dispatcher/v2 => ../../../go-cqrs-lite/dispatcher
	github.com/larsartmann/go-cqrs-lite/event/v2 => ../../../go-cqrs-lite/event
	github.com/larsartmann/go-cqrs-lite/id/v2 => ../../../go-cqrs-lite/id
	github.com/larsartmann/go-cqrs-lite/memory/v2 => ../../../go-cqrs-lite/memory
	github.com/larsartmann/go-cqrs-lite/query/v2 => ../../../go-cqrs-lite/query
	github.com/larsartmann/go-cqrs-lite/schema/v2 => ../../../go-cqrs-lite/schema
	github.com/larsartmann/go-cqrs-lite/snapshot/v2 => ../../../go-cqrs-lite/snapshot
)
