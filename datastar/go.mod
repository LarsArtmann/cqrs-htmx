module github.com/larsartmann/cqrs-htmx/datastar/v4

go 1.27.1

require (
	github.com/larsartmann/go-cqrs-lite/event/v4 v4.11.1
	github.com/larsartmann/go-cqrs-lite/id/v4 v4.6.1
	github.com/larsartmann/go-datastar v0.6.0
	github.com/larsartmann/go-datastar/broadcast v0.6.0
	github.com/larsartmann/go-sse v0.6.1
	github.com/stretchr/testify v1.12.1
)

require (
	github.com/fxamacker/cbor/v2 v2.9.4 // indirect
	github.com/larsartmann/go-branded-id v0.6.0 // indirect
	github.com/larsartmann/go-codec v0.3.0 // indirect
	github.com/larsartmann/go-cqrs-lite/metadata/v4 v4.7.1 // indirect
	//cqrs-lint:ignore(V006) go-cqrs-lite releases per-module version trains, not lockstep tags (indirect pin resolved by MVS; root go.mod carries the same suppression)
	github.com/larsartmann/go-cqrs-lite/record/v4 v4.6.0 // indirect
	github.com/larsartmann/go-cqrs-lite/snapshot/v4 v4.5.1 // indirect
	github.com/larsartmann/go-cqrs-lite/storage/memory/v4 v4.5.2 // indirect
	github.com/larsartmann/go-datastar/static v0.6.0 // indirect
	github.com/larsartmann/go-error-family v0.10.1 // indirect
	github.com/oklog/ulid/v2 v2.1.2 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
)
