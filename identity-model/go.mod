module github.com/larsartmann/cqrs-htmx/identity-model/v4

go 1.27

require (
	github.com/casbin/casbin/v3 v3.11.0
	github.com/larsartmann/go-branded-id v0.7.0
	github.com/larsartmann/go-codec v0.3.1
	github.com/larsartmann/go-cqrs-lite/command/v4 v4.13.1
	github.com/larsartmann/go-cqrs-lite/event/v4 v4.13.1
	github.com/larsartmann/go-cqrs-lite/id/v4 v4.7.1
	github.com/larsartmann/go-error-family v0.11.0
	github.com/oklog/ulid/v2 v2.1.2
)

require (
	github.com/bmatcuk/doublestar/v4 v4.10.2 // indirect
	github.com/casbin/govaluate v1.10.0 // indirect
	github.com/fxamacker/cbor/v2 v2.9.4 // indirect
	github.com/google/uuid v1.6.0 // indirect
	//cqrs-lint:ignore(V006) go-cqrs-lite releases per-module version trains, not lockstep tags (indirect pin resolved by MVS; root go.mod carries the same suppression)
	github.com/larsartmann/go-cqrs-lite/dispatcher/v4 v4.5.2 // indirect
	github.com/larsartmann/go-cqrs-lite/metadata/v4 v4.7.3 // indirect
	github.com/larsartmann/go-cqrs-lite/query/v4 v4.10.1 // indirect
	github.com/larsartmann/go-cqrs-lite/record/v4 v4.6.2 // indirect
	github.com/larsartmann/go-cqrs-lite/snapshot/v4 v4.6.1 // indirect
	github.com/larsartmann/go-cqrs-lite/storage/memory/v4 v4.6.1 // indirect
	github.com/tidwall/gjson v1.20.0 // indirect
	github.com/x448/float16 v0.8.4 // indirect
)
