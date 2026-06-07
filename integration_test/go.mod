module github.com/larsartmann/cqrs-htmx/integration_test

go 1.26.3

require (
	github.com/larsartmann/cqrs-htmx/usermgmt/v2 v2.0.0
	github.com/larsartmann/cqrs-htmx/v2 v2.0.0
	github.com/larsartmann/go-cqrs-lite/command/v2 v2.2.0
)

require (
	github.com/bmatcuk/doublestar/v4 v4.10.0 // indirect
	github.com/casbin/casbin/v3 v3.10.0 // indirect
	github.com/casbin/govaluate v1.10.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/justinas/nosurf v1.2.0 // indirect
	github.com/larsartmann/go-branded-id v0.3.0 // indirect
	github.com/larsartmann/go-cqrs-lite/codec/v2 v2.2.0 // indirect
	github.com/larsartmann/go-cqrs-lite/dispatcher/v2 v2.2.0 // indirect
	github.com/larsartmann/go-cqrs-lite/event/v2 v2.2.0 // indirect
	github.com/larsartmann/go-cqrs-lite/id/v2 v2.2.0 // indirect
	github.com/larsartmann/go-cqrs-lite/query/v2 v2.2.0 // indirect
	github.com/larsartmann/go-error-family v0.3.0 // indirect
	github.com/larsartmann/httputil v0.0.0-20260607223019-1cb4408b77a7 // indirect
	github.com/oklog/ulid/v2 v2.1.1 // indirect
	github.com/samber/lo v1.53.0 // indirect
	github.com/samber/ro v0.3.0 // indirect
	golang.org/x/crypto v0.52.0 // indirect
	golang.org/x/exp v0.0.0-20260603202125-055de637280b // indirect
	golang.org/x/text v0.37.0 // indirect
	golang.org/x/time v0.15.0 // indirect
)

replace (
	github.com/larsartmann/cqrs-htmx/usermgmt/v2 => ../usermgmt
	github.com/larsartmann/cqrs-htmx/v2 => ../
)
