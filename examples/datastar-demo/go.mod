module github.com/larsartmann/cqrs-htmx/examples/datastar-demo

go 1.26.4

require (
	github.com/larsartmann/go-cqrs-lite/command/v3 v3.7.4
	github.com/larsartmann/go-cqrs-lite/event/v3 v3.7.4
	github.com/larsartmann/go-cqrs-lite/id/v3 v3.7.4
	github.com/larsartmann/go-cqrs-lite/query/v3 v3.7.4
	github.com/larsartmann/go-error-family v0.6.1
	github.com/starfederation/datastar-go v1.2.2
)

require (
	github.com/CAFxX/httpcompression v0.0.9 // indirect
	github.com/andybalholm/brotli v1.2.2 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/fxamacker/cbor/v2 v2.9.2 // indirect
	github.com/klauspost/compress v1.19.0 // indirect
	github.com/larsartmann/go-branded-id v0.3.1 // indirect
	github.com/larsartmann/go-cqrs-lite/codec/v3 v3.7.4 // indirect
	github.com/larsartmann/go-cqrs-lite/dispatcher/v3 v3.7.4 // indirect
	github.com/oklog/ulid/v2 v2.1.1 // indirect
	github.com/onsi/ginkgo/v2 v2.32.0 // indirect
	github.com/onsi/gomega v1.42.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	golang.org/x/text v0.39.0 // indirect
	golang.org/x/tools v0.47.0 // indirect
)

replace github.com/larsartmann/go-cqrs-lite/event/v3/eventtest => ../../.vendor-local/eventtest
