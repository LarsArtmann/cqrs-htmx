module github.com/larsartmann/cqrs-htmx/integration_test

go 1.26.2

require (
	github.com/larsartmann/cqrs-htmx v0.0.0
	github.com/larsartmann/cqrs-htmx/usermgmt v0.0.0
	github.com/larsartmann/go-cqrs-lite/core v1.5.0
)

require (
	github.com/bmatcuk/doublestar/v4 v4.10.0 // indirect
	github.com/casbin/casbin/v3 v3.10.0 // indirect
	github.com/casbin/govaluate v1.10.0 // indirect
	github.com/cockroachdb/errors v1.13.0 // indirect
	github.com/cockroachdb/logtags v0.0.0-20241215232642-bb51bb14a506 // indirect
	github.com/cockroachdb/redact v1.1.8 // indirect
	github.com/getsentry/sentry-go v0.46.2 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/gorilla/csrf v1.7.3 // indirect
	github.com/gorilla/securecookie v1.1.2 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/larsartmann/go-branded-id v0.3.0 // indirect
	github.com/larsartmann/go-error-family v0.1.1 // indirect
	github.com/oklog/ulid/v2 v2.1.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/rogpeppe/go-internal v1.14.1 // indirect
	golang.org/x/crypto v0.51.0 // indirect
	golang.org/x/sys v0.44.0 // indirect
	golang.org/x/text v0.37.0 // indirect
	golang.org/x/time v0.15.0 // indirect
)

replace (
	github.com/larsartmann/cqrs-htmx => ../
	github.com/larsartmann/cqrs-htmx/usermgmt => ../usermgmt
)
