module github.com/larsartmann/cqrs-htmx/examples/admin-demo

go 1.26.4

require (
	github.com/larsartmann/cqrs-htmx/adminui/v4 v4.0.0-00010101000000-000000000000
	github.com/larsartmann/cqrs-htmx/usermgmt/totp/v4 v4.0.0-00010101000000-000000000000
	github.com/larsartmann/cqrs-htmx/usermgmt/v4 v4.0.0-00010101000000-000000000000
	github.com/larsartmann/cqrs-htmx/v4 v4.0.0-00010101000000-000000000000
	github.com/larsartmann/go-error-family v0.6.1
)

require (
	github.com/ThreeDotsLabs/watermill v1.5.2 // indirect
	github.com/a-h/templ v0.3.1020 // indirect
	github.com/bmatcuk/doublestar/v4 v4.10.0 // indirect
	github.com/boombuler/barcode v1.1.0 // indirect
	github.com/casbin/casbin/v3 v3.10.0 // indirect
	github.com/casbin/govaluate v1.10.0 // indirect
	github.com/cenkalti/backoff/v5 v5.0.3 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/fxamacker/cbor/v2 v2.9.2 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-playground/form/v4 v4.3.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx/v5 v5.10.0 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/justinas/nosurf v1.2.0 // indirect
	github.com/larsartmann/go-branded-id v0.3.1 // indirect
	github.com/larsartmann/go-cqrs-lite/codec/v3 v3.7.4 // indirect
	github.com/larsartmann/go-cqrs-lite/command/v3 v3.7.4 // indirect
	github.com/larsartmann/go-cqrs-lite/decider/v3 v3.7.4 // indirect
	github.com/larsartmann/go-cqrs-lite/dedup/v3 v3.7.4 // indirect
	github.com/larsartmann/go-cqrs-lite/dispatcher/v3 v3.7.4 // indirect
	github.com/larsartmann/go-cqrs-lite/event/v3 v3.7.4 // indirect
	github.com/larsartmann/go-cqrs-lite/id/v3 v3.7.4 // indirect
	github.com/larsartmann/go-cqrs-lite/idempotency/v3 v3.7.4 // indirect
	github.com/larsartmann/go-cqrs-lite/kv/v3 v3.7.4 // indirect
	github.com/larsartmann/go-cqrs-lite/listing/v3 v3.7.4 // indirect
	github.com/larsartmann/go-cqrs-lite/otel/v3 v3.7.4 // indirect
	github.com/larsartmann/go-cqrs-lite/projection/v3 v3.7.4 // indirect
	github.com/larsartmann/go-cqrs-lite/query/v3 v3.7.4 // indirect
	github.com/larsartmann/go-cqrs-lite/scheduling/v3 v3.7.4 // indirect
	github.com/larsartmann/go-cqrs-lite/snapshot/v3 v3.7.4 // indirect
	github.com/larsartmann/go-cqrs-lite/stack/postgres/v3 v3.7.1 // indirect
	github.com/larsartmann/go-cqrs-lite/stack/sqlite/v3 v3.7.1 // indirect
	github.com/larsartmann/go-cqrs-lite/stack/v3 v3.7.4 // indirect
	github.com/larsartmann/go-cqrs-lite/storage/memory/v3 v3.7.4 // indirect
	github.com/larsartmann/go-cqrs-lite/storage/v3 v3.7.4 // indirect
	github.com/larsartmann/go-cqrs-lite/transport/http/v3 v3.7.4 // indirect
	github.com/larsartmann/go-cqrs-lite/watermill/v3 v3.7.4 // indirect
	github.com/larsartmann/httputil v0.5.0 // indirect
	github.com/larsartmann/templ-components v0.13.0 // indirect
	github.com/lithammer/shortuuid/v3 v3.0.7 // indirect
	github.com/mattn/go-isatty v0.0.22 // indirect
	github.com/maypok86/otter/v2 v2.3.0 // indirect
	github.com/ncruces/go-strftime v1.0.0 // indirect
	github.com/oklog/ulid v1.3.1 // indirect
	github.com/oklog/ulid/v2 v2.1.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/pquerna/otp v1.5.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/sony/gobreaker v1.0.0 // indirect
	github.com/stretchr/testify v1.11.1 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/otel v1.44.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.44.0 // indirect
	go.opentelemetry.io/otel/metric v1.44.0 // indirect
	go.opentelemetry.io/otel/sdk v1.44.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.44.0 // indirect
	go.opentelemetry.io/otel/trace v1.44.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.40.0 // indirect
	golang.org/x/time v0.15.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	modernc.org/libc v1.74.1 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.11.0 // indirect
	modernc.org/sqlite v1.53.0 // indirect
)

replace (
	github.com/larsartmann/cqrs-htmx/adminui/v4 => ../../adminui
	github.com/larsartmann/cqrs-htmx/usermgmt/totp/v4 => ../../usermgmt/totp
	github.com/larsartmann/cqrs-htmx/usermgmt/v4 => ../../usermgmt
	github.com/larsartmann/cqrs-htmx/v4 => ../..
)

replace github.com/larsartmann/go-cqrs-lite/event/v3/eventtest => ../../.vendor-local/eventtest
