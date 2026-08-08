module github.com/larsartmann/cqrs-htmx/usermgmt/v4

go 1.26.5

require (
	github.com/go-sql-driver/mysql v1.10.0
	github.com/larsartmann/cqrs-htmx/identity-model/v4 v4.2.0
	github.com/larsartmann/cqrs-htmx/v4 v4.7.0
	github.com/larsartmann/go-cqrs-lite/command/v4 v4.4.0
	github.com/larsartmann/go-cqrs-lite/decider/v4 v4.2.0
	github.com/larsartmann/go-cqrs-lite/dedup/v4 v4.2.0
	github.com/larsartmann/go-cqrs-lite/event/v4 v4.4.0
	github.com/larsartmann/go-cqrs-lite/id/v4 v4.2.0
	github.com/larsartmann/go-cqrs-lite/projection/v4 v4.2.0
	github.com/larsartmann/go-cqrs-lite/projectionhost/v4 v4.2.0
	github.com/larsartmann/go-cqrs-lite/query/v4 v4.2.0
	github.com/larsartmann/go-cqrs-lite/scenario/v4 v4.0.1
	github.com/larsartmann/go-cqrs-lite/stack/v4 v4.3.0
	github.com/larsartmann/go-cqrs-lite/storage/memory/v4 v4.3.0
	github.com/larsartmann/go-cqrs-lite/storage/v4 v4.6.0
	github.com/larsartmann/go-cqrs-lite/watermill/v4 v4.2.0
	github.com/larsartmann/go-error-family v0.10.0
	github.com/larsartmann/httputil v0.10.0
	github.com/testcontainers/testcontainers-go/modules/mysql v0.43.0
	modernc.org/sqlite v1.56.0
	pgregory.net/rapid v1.3.0
)

require (
	github.com/ThreeDotsLabs/watermill v1.5.2
	github.com/cenkalti/backoff/v5 v5.0.3 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/larsartmann/go-cqrs-lite/codec/v4 v4.2.0
	github.com/larsartmann/go-cqrs-lite/dispatcher/v4 v4.2.0 // indirect
	github.com/larsartmann/go-cqrs-lite/kv/v4 v4.2.0
	github.com/larsartmann/go-cqrs-lite/listing/v4 v4.2.0 // indirect
	github.com/larsartmann/go-cqrs-lite/otel/v4 v4.2.0 // indirect
	github.com/larsartmann/go-cqrs-lite/snapshot/v4 v4.2.0
	github.com/lithammer/shortuuid/v3 v3.0.7 // indirect
	github.com/maypok86/otter/v2 v2.3.0 // indirect
	github.com/oklog/ulid v1.3.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/sony/gobreaker v1.0.0 // indirect
	github.com/stretchr/testify v1.11.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

require (
	dario.cat/mergo v1.0.2 // indirect
	filippo.io/edwards25519 v1.2.0 // indirect
	github.com/Azure/go-ansiterm v0.0.0-20250102033503-faa5f7b0171c // indirect
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/bmatcuk/doublestar/v4 v4.10.0 // indirect
	github.com/casbin/casbin/v3 v3.11.0 // indirect
	github.com/casbin/govaluate v1.10.0 // indirect
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/containerd/errdefs v1.0.0 // indirect
	github.com/containerd/errdefs/pkg v0.3.0 // indirect
	github.com/containerd/log v0.1.0 // indirect
	github.com/containerd/platforms v0.2.1 // indirect
	github.com/cpuguy83/dockercfg v0.3.2 // indirect
	github.com/distribution/reference v0.6.0 // indirect
	github.com/docker/go-connections v0.6.0 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/ebitengine/purego v0.10.2 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/fxamacker/cbor/v2 v2.9.2 // indirect
	github.com/go-logr/logr v1.4.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/go-playground/form/v4 v4.3.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/justinas/nosurf v1.2.0 // indirect
	github.com/klauspost/compress v1.19.2 // indirect
	github.com/larsartmann/go-branded-id v0.5.1 // indirect
	github.com/larsartmann/go-cqrs-lite/flightrecorder/v4 v4.0.0-20260808205557-5709c34eac8d // indirect
	github.com/larsartmann/go-cqrs-lite/idempotency/v4 v4.3.0 // indirect
	github.com/larsartmann/go-cqrs-lite/metadata/v4 v4.2.0 // indirect
	github.com/larsartmann/go-cqrs-lite/metaengine/v4 v4.6.0 // indirect
	github.com/larsartmann/go-cqrs-lite/record/v4 v4.0.0 // indirect
	github.com/larsartmann/go-cqrs-lite/scheduling/v4 v4.2.0 // indirect
	github.com/larsartmann/go-etag v0.1.0 // indirect
	github.com/larsartmann/go-idempotency v0.1.2 // indirect
	github.com/larsartmann/go-sse v0.4.0 // indirect
	github.com/larsartmann/httputil/server_timing v0.10.0 // indirect
	github.com/lufia/plan9stats v0.0.0-20211012122336-39d0f177ccd0 // indirect
	github.com/magiconair/properties v1.8.10 // indirect
	github.com/mattn/go-isatty v0.0.24 // indirect
	github.com/moby/docker-image-spec v1.3.1 // indirect
	github.com/moby/go-archive v0.2.0 // indirect
	github.com/moby/moby/api v1.54.2 // indirect
	github.com/moby/moby/client v0.4.0 // indirect
	github.com/moby/patternmatcher v0.6.1 // indirect
	github.com/moby/sys/sequential v0.6.0 // indirect
	github.com/moby/sys/user v0.4.0 // indirect
	github.com/moby/sys/userns v0.1.0 // indirect
	github.com/moby/term v0.5.2 // indirect
	github.com/ncruces/go-strftime v1.0.0 // indirect
	github.com/oklog/ulid/v2 v2.1.2 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.1 // indirect
	github.com/power-devops/perfstat v0.0.0-20240221224432-82ca36839d55 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/shirou/gopsutil/v4 v4.26.5 // indirect
	github.com/sirupsen/logrus v1.9.4 // indirect
	github.com/testcontainers/testcontainers-go v0.43.0 // indirect
	github.com/tklauser/go-sysconf v0.3.16 // indirect
	github.com/tklauser/numcpus v0.11.0 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	github.com/yusufpapurcu/wmi v1.2.4 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.61.0 // indirect
	go.opentelemetry.io/otel v1.45.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.45.0 // indirect
	go.opentelemetry.io/otel/metric v1.45.0 // indirect
	go.opentelemetry.io/otel/sdk v1.45.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.45.0 // indirect
	go.opentelemetry.io/otel/trace v1.45.0 // indirect
	golang.org/x/crypto v0.54.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/time v0.15.0 // indirect
	modernc.org/libc v1.75.0 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.12.0 // indirect
)
