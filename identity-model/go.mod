module github.com/larsartmann/cqrs-htmx/identity-model/v4

go 1.27

require (
	github.com/casbin/casbin/v3 v3.11.0
	github.com/larsartmann/go-branded-id v0.8.0
	github.com/larsartmann/go-codec v0.3.4
	github.com/larsartmann/go-cqrs-lite/command/v4 v4.14.2
	github.com/larsartmann/go-cqrs-lite/event/v4 v4.15.3
	github.com/larsartmann/go-cqrs-lite/id/v4 v4.7.6
	github.com/larsartmann/go-error-family v0.12.0
	github.com/oklog/ulid/v2 v2.1.2
)

require (
	github.com/bmatcuk/doublestar/v4 v4.10.4 // indirect
	github.com/casbin/govaluate v1.10.0 // indirect
	github.com/fxamacker/cbor/v2 v2.10.0 // indirect
	github.com/google/pprof v0.0.0-20261105003125-4bf881d7fb0b // indirect
	github.com/google/uuid v1.6.0 // indirect
	//cqrs-lint:ignore(V006) go-cqrs-lite releases per-module version trains, not lockstep tags (indirect pin resolved by MVS; root go.mod carries the same suppression)
	github.com/larsartmann/go-cqrs-lite/dispatcher/v4 v4.5.2 // indirect
	github.com/larsartmann/go-cqrs-lite/metadata/v4 v4.7.3 // indirect
	github.com/larsartmann/go-cqrs-lite/record/v4 v4.6.2 // indirect
	github.com/larsartmann/go-cqrs-lite/schema/v4 v4.7.2 // indirect
	github.com/onsi/ginkgo/v2 v2.34.0 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	golang.org/x/mod v0.43.0 // indirect
	golang.org/x/sync v0.25.0 // indirect
	golang.org/x/sys v0.50.0 // indirect
	golang.org/x/tools v0.52.0 // indirect
)
