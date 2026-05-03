module github.com/larsartmann/cqrs-htmx

go 1.26.2

require (
	github.com/casbin/casbin/v3 v3.10.0
	github.com/cockroachdb/errors v1.12.0
	github.com/larsartmann/go-cqrs-lite/core v0.0.0
	github.com/onsi/ginkgo/v2 v2.28.3
	github.com/onsi/gomega v1.40.0
)

require (
	github.com/Masterminds/semver/v3 v3.4.0 // indirect
	github.com/bmatcuk/doublestar/v4 v4.6.1 // indirect
	github.com/casbin/govaluate v1.3.0 // indirect
	github.com/cockroachdb/logtags v0.0.0-20241215232642-bb51bb14a506 // indirect
	github.com/cockroachdb/redact v1.1.8 // indirect
	github.com/getsentry/sentry-go v0.45.1 // indirect
	github.com/go-json-experiment/json v0.0.0-20260430182902-b6187a392ed4 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-task/slim-sprig/v3 v3.0.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/pprof v0.0.0-20260402051712-545e8a4df936 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/larsartmann/go-branded-id v0.1.0 // indirect
	github.com/oklog/ulid/v2 v2.1.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/rogpeppe/go-internal v1.14.1 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/mod v0.35.0 // indirect
	golang.org/x/net v0.53.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/sys v0.43.0 // indirect
	golang.org/x/text v0.36.0 // indirect
	golang.org/x/tools v0.44.0 // indirect
)

replace (
	github.com/larsartmann/go-cqrs-lite/core => ../go-cqrs-lite/core
	github.com/larsartmann/go-cqrs-lite/memory => ../go-cqrs-lite/memory
	github.com/larsartmann/go-cqrs-lite/testhelpers => ../go-cqrs-lite/testhelpers
)
