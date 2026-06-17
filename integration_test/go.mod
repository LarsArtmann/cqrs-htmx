module github.com/larsartmann/cqrs-htmx/integration_test

go 1.26.3

require (
	github.com/larsartmann/cqrs-htmx/usermgmt/v2 v2.0.0
	github.com/larsartmann/cqrs-htmx/v2 v2.0.0
	github.com/larsartmann/go-cqrs-lite/command/v2 v2.3.0
	github.com/larsartmann/go-cqrs-lite/query/v2 v2.3.0
)

require (
	github.com/bmatcuk/doublestar/v4 v4.10.0 // indirect
	github.com/boombuler/barcode v1.0.1-0.20190219062509-6c824513bacc // indirect
	github.com/casbin/casbin/v3 v3.10.0 // indirect
	github.com/casbin/govaluate v1.10.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/fxamacker/cbor/v2 v2.9.2 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-viper/mapstructure/v2 v2.5.0 // indirect
	github.com/go-webauthn/webauthn v0.17.4 // indirect
	github.com/go-webauthn/x v0.2.6 // indirect
	github.com/golang-jwt/jwt/v5 v5.3.1 // indirect
	github.com/google/go-tpm v0.9.8 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/justinas/nosurf v1.2.0 // indirect
	github.com/larsartmann/go-branded-id v0.3.0 // indirect
	github.com/larsartmann/go-cqrs-lite/codec/v2 v2.3.0 // indirect
	github.com/larsartmann/go-cqrs-lite/decider/v2 v2.3.0 // indirect
	github.com/larsartmann/go-cqrs-lite/dispatcher/v2 v2.3.0 // indirect
	github.com/larsartmann/go-cqrs-lite/event/v2 v2.3.1 // indirect
	github.com/larsartmann/go-cqrs-lite/id/v2 v2.3.0 // indirect
	github.com/larsartmann/go-cqrs-lite/memory/v2 v2.3.0 // indirect
	github.com/larsartmann/go-cqrs-lite/otel/v2 v2.3.0 // indirect
	github.com/larsartmann/go-cqrs-lite/projection/v2 v2.3.0 // indirect
	github.com/larsartmann/go-cqrs-lite/snapshot/v2 v2.3.0 // indirect
	github.com/larsartmann/go-error-family v0.3.0 // indirect
	github.com/larsartmann/httputil v0.2.0 // indirect
	github.com/oklog/ulid/v2 v2.1.1 // indirect
	github.com/philhofer/fwd v1.2.0 // indirect
	github.com/pquerna/otp v1.5.0 // indirect
	github.com/samber/lo v1.53.0 // indirect
	github.com/samber/ro v0.3.0 // indirect
	github.com/tinylib/msgp v1.6.4 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/otel v1.44.0 // indirect
	go.opentelemetry.io/otel/metric v1.44.0 // indirect
	go.opentelemetry.io/otel/trace v1.44.0 // indirect
	golang.org/x/crypto v0.53.0 // indirect
	golang.org/x/exp v0.0.0-20260611194520-c48552f49976 // indirect
	golang.org/x/sync v0.21.0 // indirect
	golang.org/x/sys v0.46.0 // indirect
	golang.org/x/text v0.38.0 // indirect
	golang.org/x/time v0.15.0 // indirect
)

replace (
	github.com/larsartmann/cqrs-htmx/usermgmt/v2 => ../usermgmt
	github.com/larsartmann/cqrs-htmx/v2 => ../
)
