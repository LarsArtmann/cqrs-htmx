module github.com/larsartmann/cqrs-htmx/integration_test

go 1.26.3

require (
	github.com/larsartmann/cqrs-htmx v1.0.0
	github.com/larsartmann/cqrs-htmx/usermgmt v0.0.0
	github.com/larsartmann/go-cqrs-lite/command/v2 v2.0.0
)

require (
	github.com/bmatcuk/doublestar/v4 v4.10.0 // indirect
	github.com/casbin/casbin/v3 v3.10.0 // indirect
	github.com/casbin/govaluate v1.10.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/justinas/nosurf v1.2.0 // indirect
	github.com/larsartmann/go-branded-id v0.3.0 // indirect
	github.com/larsartmann/go-cqrs-lite/codec/v2 v2.0.0 // indirect
	github.com/larsartmann/go-cqrs-lite/dispatcher/v2 v2.0.0 // indirect
	github.com/larsartmann/go-cqrs-lite/event/v2 v2.0.0 // indirect
	github.com/larsartmann/go-cqrs-lite/id/v2 v2.0.0 // indirect
	github.com/larsartmann/go-cqrs-lite/query/v2 v2.0.0 // indirect
	github.com/larsartmann/go-error-family v0.3.0 // indirect
	github.com/larsartmann/httputil v0.0.0-20260528145236-22c2616f7c39 // indirect
	github.com/oklog/ulid/v2 v2.1.1 // indirect
	github.com/samber/lo v1.53.0 // indirect
	github.com/samber/ro v0.3.0 // indirect
	golang.org/x/crypto v0.52.0 // indirect
	golang.org/x/exp v0.0.0-20260529124908-c761662dc8c9 // indirect
	golang.org/x/text v0.37.0 // indirect
	golang.org/x/time v0.15.0 // indirect
)

replace (
	github.com/larsartmann/cqrs-htmx => ../
	github.com/larsartmann/cqrs-htmx/usermgmt => ../usermgmt
	github.com/larsartmann/go-cqrs-lite/codec/v2 => ../../go-cqrs-lite/codec
	github.com/larsartmann/go-cqrs-lite/command/v2 => ../../go-cqrs-lite/command
	github.com/larsartmann/go-cqrs-lite/dispatcher/v2 => ../../go-cqrs-lite/dispatcher
	github.com/larsartmann/go-cqrs-lite/event/v2 => ../../go-cqrs-lite/event
	github.com/larsartmann/go-cqrs-lite/id/v2 => ../../go-cqrs-lite/id
	github.com/larsartmann/go-cqrs-lite/memory/v2 => ../../go-cqrs-lite/memory
	github.com/larsartmann/go-cqrs-lite/query/v2 => ../../go-cqrs-lite/query
	github.com/larsartmann/go-cqrs-lite/schema/v2 => ../../go-cqrs-lite/schema
	github.com/larsartmann/go-cqrs-lite/snapshot/v2 => ../../go-cqrs-lite/snapshot
)
