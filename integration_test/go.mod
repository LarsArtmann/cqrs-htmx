module github.com/larsartmann/cqrs-htmx/integration_test

go 1.26.3

require (
	github.com/larsartmann/cqrs-htmx v1.0.0
	github.com/larsartmann/cqrs-htmx/usermgmt v0.0.0
	github.com/larsartmann/go-cqrs-lite/core v1.6.0
)

require (
	github.com/bmatcuk/doublestar/v4 v4.10.0 // indirect
	github.com/casbin/casbin/v3 v3.10.0 // indirect
	github.com/casbin/govaluate v1.10.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/justinas/nosurf v1.2.0 // indirect
	github.com/larsartmann/go-branded-id v0.3.0 // indirect
	github.com/larsartmann/go-error-family v0.2.0 // indirect
	github.com/larsartmann/httputil v0.0.0-20260526092845-4c4df6dce62d // indirect
	github.com/oklog/ulid/v2 v2.1.1 // indirect
	golang.org/x/crypto v0.51.0 // indirect
	golang.org/x/time v0.15.0 // indirect
)

replace (
	github.com/larsartmann/cqrs-htmx => ../
	github.com/larsartmann/cqrs-htmx/usermgmt => ../usermgmt
)
