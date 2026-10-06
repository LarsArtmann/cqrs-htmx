//cqrs-lint:ignore(E014) dashboard consumes projections, does not own them
module github.com/larsartmann/cqrs-htmx/dashboardui/v4

go 1.27

require (
	github.com/a-h/templ v0.3.1070
	github.com/dustin/go-humanize v1.1.0
	github.com/larsartmann/cqrs-htmx/v4 v4.13.1
	github.com/larsartmann/go-codec v0.3.1
	github.com/larsartmann/go-cqrs-lite/command/v4 v4.13.1
	github.com/larsartmann/go-cqrs-lite/event/v4 v4.13.1
	github.com/larsartmann/go-cqrs-lite/event/v4/eventtest v0.4.0
	github.com/larsartmann/go-cqrs-lite/id/v4 v4.7.1
	github.com/larsartmann/go-cqrs-lite/listing/v4 v4.4.3
	github.com/larsartmann/go-cqrs-lite/projectionhost/v4 v4.5.3
	github.com/larsartmann/go-cqrs-lite/query/v4 v4.10.1
	github.com/larsartmann/go-cqrs-lite/snapshot/v4 v4.6.1
	github.com/larsartmann/go-cqrs-lite/storage/memory/v4 v4.6.1
	github.com/larsartmann/go-error-family v0.11.0
	github.com/larsartmann/go-sse v0.6.2
	github.com/larsartmann/httputil v1.4.1
	github.com/larsartmann/templ-components v1.20.1
	github.com/larsartmann/templ-components/errorpage v1.20.1
	github.com/larsartmann/templ-components/htmx v1.20.1
	github.com/larsartmann/templ-components/icons v1.20.1
	github.com/larsartmann/templ-components/utils v1.20.1
	github.com/onsi/ginkgo/v2 v2.33.0
	github.com/onsi/gomega v1.44.0
)

require (
	github.com/Masterminds/semver/v3 v3.5.0 // indirect
	github.com/Oudwins/tailwind-merge-go v0.2.3 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/fxamacker/cbor/v2 v2.9.4 // indirect
	github.com/gkampitakis/ciinfo v0.3.4 // indirect
	github.com/gkampitakis/go-snaps v0.5.23 // indirect
	github.com/go-logr/logr v1.4.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-playground/form/v4 v4.5.0 // indirect
	github.com/go-task/slim-sprig/v3 v3.0.0 // indirect
	github.com/goccy/go-yaml v1.19.2 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/pprof v0.0.0-20260926063103-aaccee046517 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/justinas/nosurf v1.2.0 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/larsartmann/go-branded-id v0.7.0 // indirect
	//cqrs-lint:ignore(V006) go-cqrs-lite releases per-module version trains, not lockstep tags (indirect pin resolved by MVS; root go.mod carries the same suppression)
	github.com/larsartmann/go-cqrs-lite/dedup/v4 v4.2.4 // indirect
	github.com/larsartmann/go-cqrs-lite/dispatcher/v4 v4.5.2 // indirect
	github.com/larsartmann/go-cqrs-lite/kv/v4 v4.3.3 // indirect
	github.com/larsartmann/go-cqrs-lite/metadata/v4 v4.7.3 // indirect
	github.com/larsartmann/go-cqrs-lite/otel/v4 v4.5.2 // indirect
	github.com/larsartmann/go-cqrs-lite/projection/v4 v4.4.2 // indirect
	github.com/larsartmann/go-cqrs-lite/record/v4 v4.6.2 // indirect
	github.com/larsartmann/go-cqrs-lite/scheduling/v4 v4.6.1 // indirect
	github.com/larsartmann/go-cqrs-lite/storage/v4 v4.10.4 // indirect
	github.com/larsartmann/go-etag/entitytag v0.6.1 // indirect
	github.com/larsartmann/go-etag/server v0.6.1 // indirect
	github.com/larsartmann/go-flightrecorder v0.2.0 // indirect
	github.com/larsartmann/go-idempotency v0.3.1 // indirect
	github.com/larsartmann/httputil/server_timing v1.0.1 // indirect
	github.com/maruel/natural v1.3.0 // indirect
	github.com/oklog/ulid/v2 v2.1.2 // indirect
	github.com/rogpeppe/go-internal v1.16.0 // indirect
	github.com/sergi/go-diff v1.4.0 // indirect
	github.com/tidwall/gjson v1.20.0 // indirect
	github.com/tidwall/match v1.2.0 // indirect
	github.com/tidwall/pretty v1.2.2 // indirect
	github.com/tidwall/sjson v1.2.5 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/otel v1.47.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.47.0 // indirect
	go.opentelemetry.io/otel/log v1.47.0 // indirect
	go.opentelemetry.io/otel/metric v1.47.0 // indirect
	go.opentelemetry.io/otel/sdk v1.47.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.47.0 // indirect
	go.opentelemetry.io/otel/trace v1.47.0 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/mod v0.41.0 // indirect
	golang.org/x/net v0.59.0 // indirect
	golang.org/x/sync v0.23.0 // indirect
	golang.org/x/sys v0.48.0 // indirect
	golang.org/x/text v0.42.0 // indirect
	golang.org/x/time v0.16.0 // indirect
	golang.org/x/tools v0.51.0 // indirect
)

//cqrs-lint:ignore(F015) dashboard displays data, does not own query planning
//cqrs-lint:ignore(E009) dashboard IS the transport/view layer
//cqrs-lint:ignore(E014) dashboard consumes projections, does not own them
