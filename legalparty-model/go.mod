module github.com/larsartmann/cqrs-htmx/legalparty-model/v4

go 1.27.3

require (
	github.com/larsartmann/cqrs-htmx/identity-model/v4 v4.17.0
	github.com/larsartmann/go-branded-id v0.7.1
	github.com/larsartmann/go-codec v0.3.3
	github.com/larsartmann/go-cqrs-lite/command/v4 v4.14.1
	github.com/larsartmann/go-cqrs-lite/decider/v4 v4.8.0
	github.com/larsartmann/go-cqrs-lite/event/v4 v4.15.2
	github.com/larsartmann/go-cqrs-lite/id/v4 v4.7.5
	github.com/larsartmann/go-error-family v0.12.0
	github.com/oklog/ulid/v2 v2.1.2
)

require (
	github.com/bmatcuk/doublestar/v4 v4.10.2 // indirect
	github.com/casbin/casbin/v3 v3.11.0 // indirect
	github.com/casbin/govaluate v1.10.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/fxamacker/cbor/v2 v2.10.0 // indirect
	github.com/go-logr/logr v1.4.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	//cqrs-lint:ignore(V006) go-cqrs-lite releases per-module version trains, not lockstep tags (indirect pin resolved by MVS; root go.mod carries the same suppression)
	github.com/larsartmann/go-cqrs-lite/dispatcher/v4 v4.5.2 // indirect
	github.com/larsartmann/go-cqrs-lite/metadata/v4 v4.7.3 // indirect
	github.com/larsartmann/go-cqrs-lite/otel/v4 v4.5.3 // indirect
	github.com/larsartmann/go-cqrs-lite/record/v4 v4.6.2 // indirect
	github.com/larsartmann/go-cqrs-lite/snapshot/v4 v4.7.0 // indirect
	github.com/larsartmann/go-flightrecorder v0.2.1 // indirect
	github.com/larsartmann/go-legalparty v0.1.0 // indirect
	github.com/maypok86/otter/v2 v2.3.0 // indirect
	github.com/stretchr/testify v1.12.1 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/otel v1.47.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.47.0 // indirect
	go.opentelemetry.io/otel/log v1.47.0 // indirect
	go.opentelemetry.io/otel/metric v1.47.0 // indirect
	go.opentelemetry.io/otel/sdk v1.47.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.47.0 // indirect
	go.opentelemetry.io/otel/trace v1.47.0 // indirect
	go.yaml.in/yaml/v3 v3.0.6 // indirect
	golang.org/x/sync v0.24.0 // indirect
	golang.org/x/sys v0.49.0 // indirect
)
