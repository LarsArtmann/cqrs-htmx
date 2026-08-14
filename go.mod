//cqrs-lint:ignore(E003) identity-model is intentionally a cohesive pure-domain package (IDs, events, commands, fold); splitting would break usermgmt's alias/re-export architecture
module github.com/larsartmann/cqrs-htmx/v4

go 1.26.5

require (
	github.com/casbin/casbin/v3 v3.10.0
	github.com/go-playground/form/v4 v4.3.0
	github.com/larsartmann/go-cqrs-lite/command/v4 v4.6.0
	github.com/larsartmann/go-cqrs-lite/event/v4 v4.6.0
	github.com/larsartmann/go-cqrs-lite/id/v4 v4.4.0
	github.com/larsartmann/go-cqrs-lite/idempotency/v4 v4.4.0
	github.com/larsartmann/go-cqrs-lite/query/v4 v4.5.0
	github.com/larsartmann/go-cqrs-lite/storage/memory/v4 v4.2.0
	github.com/larsartmann/go-error-family v0.10.0
	github.com/larsartmann/go-sse v0.5.0
	github.com/larsartmann/httputil v0.11.0
	github.com/larsartmann/httputil/server_timing v0.10.0
	github.com/oklog/ulid/v2 v2.1.2
	github.com/onsi/ginkgo/v2 v2.32.0
	github.com/onsi/gomega v1.42.1
)

require (
	github.com/Masterminds/semver/v3 v3.5.0 // indirect
	github.com/bmatcuk/doublestar/v4 v4.10.0 // indirect
	github.com/casbin/govaluate v1.10.0 // indirect
	github.com/fxamacker/cbor/v2 v2.9.2 // indirect
	github.com/go-logr/logr v1.4.4 // indirect
	github.com/go-task/slim-sprig/v3 v3.0.0 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/pprof v0.0.0-20260802141513-ef3492d7dac3 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/justinas/nosurf v1.2.0 // indirect
	github.com/larsartmann/go-branded-id v0.5.1 // indirect
	github.com/larsartmann/go-codec v0.1.0 // indirect
	github.com/larsartmann/go-cqrs-lite/codec/v4 v4.4.0 // indirect
	github.com/larsartmann/go-cqrs-lite/dispatcher/v4 v4.3.0 // indirect
	github.com/larsartmann/go-cqrs-lite/metadata/v4 v4.4.0 // indirect
	github.com/larsartmann/go-cqrs-lite/record/v4 v4.2.0 // indirect
	github.com/larsartmann/go-cqrs-lite/snapshot/v4 v4.2.0 // indirect
	github.com/larsartmann/go-etag v0.1.1 // indirect
	github.com/larsartmann/go-idempotency v0.1.2 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/mod v0.39.0 // indirect
	golang.org/x/net v0.57.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.41.0 // indirect
	golang.org/x/time v0.15.0 // indirect
	golang.org/x/tools v0.48.0 // indirect
)
