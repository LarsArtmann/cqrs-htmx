module github.com/larsartmann/cqrs-htmx/usermgmt/v4

go 1.27.3

require (
	github.com/ThreeDotsLabs/watermill v1.5.3
	github.com/go-sql-driver/mysql v1.10.1
	github.com/larsartmann/cqrs-htmx/identity-model/v4 v4.17.0
	github.com/larsartmann/cqrs-htmx/v4 v4.18.1
	github.com/larsartmann/go-codec v0.3.3
	github.com/larsartmann/go-cqrs-lite/catalog/v4 v4.14.0
	github.com/larsartmann/go-cqrs-lite/command/v4 v4.14.1
	github.com/larsartmann/go-cqrs-lite/commandlifecycle/v4 v4.3.1
	github.com/larsartmann/go-cqrs-lite/decider/v4 v4.8.0
	//cqrs-lint:ignore(V006) go-cqrs-lite releases per-module version trains, not lockstep tags (indirect pin resolved by MVS; root go.mod carries the same suppression)
	github.com/larsartmann/go-cqrs-lite/dedup/v4 v4.3.0
	github.com/larsartmann/go-cqrs-lite/event/v4 v4.15.2
	github.com/larsartmann/go-cqrs-lite/id/v4 v4.7.5
	github.com/larsartmann/go-cqrs-lite/kv/v4 v4.4.0
	github.com/larsartmann/go-cqrs-lite/middleware/v4 v4.7.2
	github.com/larsartmann/go-cqrs-lite/projection/v4 v4.4.2
	github.com/larsartmann/go-cqrs-lite/projectionhost/v4 v4.7.1
	github.com/larsartmann/go-cqrs-lite/query/v4 v4.10.1
	github.com/larsartmann/go-cqrs-lite/record/v4 v4.6.2
	github.com/larsartmann/go-cqrs-lite/scenario/v4 v4.4.3
	github.com/larsartmann/go-cqrs-lite/snapshot/v4 v4.7.0
	github.com/larsartmann/go-cqrs-lite/stack/v4 v4.5.1
	github.com/larsartmann/go-cqrs-lite/storage/memory/v4 v4.6.2
	github.com/larsartmann/go-cqrs-lite/storage/v4 v4.11.1
	github.com/larsartmann/go-cqrs-lite/watermill/v4 v4.7.0
	github.com/larsartmann/go-error-family v0.12.0
	github.com/larsartmann/go-idempotency v0.3.2
	github.com/larsartmann/httputil v1.8.0
	github.com/testcontainers/testcontainers-go/modules/mysql v0.44.0
	modernc.org/sqlite v1.61.1
	pgregory.net/rapid v1.3.0
)

require (
	dario.cat/mergo v1.0.2 // indirect
	filippo.io/edwards25519 v1.2.0 // indirect
	github.com/Azure/go-ansiterm v0.0.0-20261013165250-51075c6ea6bc // indirect
	github.com/Microsoft/go-winio v0.6.4 // indirect
	github.com/bits-and-blooms/bitset v1.29.0 // indirect
	github.com/bmatcuk/doublestar/v4 v4.10.2 // indirect
	github.com/casbin/casbin/v3 v3.11.0 // indirect
	github.com/casbin/govaluate v1.10.0 // indirect
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/cenkalti/backoff/v5 v5.0.3 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/containerd/errdefs v1.0.0 // indirect
	github.com/containerd/errdefs/pkg v0.3.0 // indirect
	github.com/containerd/log v0.2.0 // indirect
	github.com/containerd/platforms v0.2.1 // indirect
	github.com/cpuguy83/dockercfg v0.3.2 // indirect
	github.com/distribution/reference v0.6.0 // indirect
	github.com/docker/go-connections v0.8.2 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/dustin/go-humanize v1.1.0 // indirect
	github.com/ebitengine/purego v0.11.3 // indirect
	github.com/failsafe-go/failsafe-go v0.9.8 // indirect
	github.com/felixge/httpsnoop v1.1.0 // indirect
	github.com/fxamacker/cbor/v2 v2.10.0 // indirect
	github.com/go-faster/errors v0.8.0 // indirect
	github.com/go-faster/jx v1.2.0 // indirect
	github.com/go-faster/yaml v0.4.6 // indirect
	github.com/go-logr/logr v1.4.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-ole/go-ole v1.3.0 // indirect
	github.com/go-playground/form/v4 v4.5.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/jackc/pgx/v5 v5.12.0 // indirect
	github.com/justinas/nosurf v1.2.0 // indirect
	github.com/klauspost/compress v1.20.3 // indirect
	github.com/larsartmann/go-branded-id v0.7.1 // indirect
	github.com/larsartmann/go-cqrs-lite/dispatcher/v4 v4.5.2 // indirect
	github.com/larsartmann/go-cqrs-lite/listing/v4 v4.4.4 // indirect
	github.com/larsartmann/go-cqrs-lite/metadata/v4 v4.7.3 // indirect
	github.com/larsartmann/go-cqrs-lite/metaengine/projectionadapter/v4 v4.6.2 // indirect
	github.com/larsartmann/go-cqrs-lite/metaengine/v4 v4.22.0 // indirect
	github.com/larsartmann/go-cqrs-lite/otel/v4 v4.5.3 // indirect
	github.com/larsartmann/go-cqrs-lite/scheduling/v4 v4.7.0 // indirect
	github.com/larsartmann/go-cqrs-lite/schema/v4 v4.7.1 // indirect
	github.com/larsartmann/go-etag/entitytag v0.6.1 // indirect
	github.com/larsartmann/go-etag/server v0.6.1 // indirect
	github.com/larsartmann/go-flightrecorder v0.2.1 // indirect
	github.com/larsartmann/go-retry v0.9.0 // indirect
	github.com/larsartmann/go-sqlitestore v0.1.1 // indirect
	github.com/larsartmann/go-sse v0.7.0 // indirect
	github.com/larsartmann/httputil/server_timing v1.0.3 // indirect
	github.com/lithammer/shortuuid/v3 v3.0.7 // indirect
	github.com/lufia/plan9stats v0.0.0-20261101121723-be84dd9509a0 // indirect
	github.com/magiconair/properties v1.18.12 // indirect
	github.com/mattn/go-isatty v0.0.24 // indirect
	github.com/maypok86/otter/v2 v2.4.0 // indirect
	github.com/moby/docker-image-spec v1.3.1 // indirect
	github.com/moby/go-archive v0.3.3 // indirect
	github.com/moby/moby/api v1.56.1 // indirect
	github.com/moby/moby/client v0.6.2 // indirect
	github.com/moby/patternmatcher v0.6.1 // indirect
	github.com/moby/sys/sequential v0.7.0 // indirect
	github.com/moby/sys/user v0.5.0 // indirect
	github.com/moby/sys/userns v0.2.1 // indirect
	github.com/moby/term v0.5.2 // indirect
	github.com/ncruces/go-strftime v1.1.0 // indirect
	github.com/oklog/ulid v1.3.1 // indirect
	github.com/oklog/ulid/v2 v2.1.2 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/power-devops/perfstat v0.0.0-20260916203055-22a1a467d9f0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/segmentio/asm v1.2.1 // indirect
	github.com/shirou/gopsutil/v4 v4.26.10 // indirect
	github.com/sirupsen/logrus v1.10.2 // indirect
	github.com/sony/gobreaker v1.0.0 // indirect
	github.com/stretchr/testify v1.12.1 // indirect
	github.com/testcontainers/testcontainers-go v0.44.0 // indirect
	github.com/tklauser/go-sysconf v0.4.0 // indirect
	github.com/tklauser/numcpus v0.12.0 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	github.com/yusufpapurcu/wmi v1.2.4 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.72.0 // indirect
	go.opentelemetry.io/otel v1.47.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.47.0 // indirect
	go.opentelemetry.io/otel/log v1.47.0 // indirect
	go.opentelemetry.io/otel/metric v1.47.0 // indirect
	go.opentelemetry.io/otel/sdk v1.47.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.47.0 // indirect
	go.opentelemetry.io/otel/trace v1.47.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.yaml.in/yaml/v3 v3.0.6 // indirect
	golang.org/x/crypto v0.58.0 // indirect
	golang.org/x/sync v0.24.0 // indirect
	golang.org/x/sys v0.49.0 // indirect
	golang.org/x/time v0.16.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20261026165753-2ffab32bce7e // indirect
	google.golang.org/grpc v1.85.0 // indirect
	modernc.org/libc v1.80.0 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.12.1 // indirect
)
