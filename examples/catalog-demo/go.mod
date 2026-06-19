module github.com/larsartmann/cqrs-htmx/examples/catalog-demo

go 1.26.3

require (
	github.com/larsartmann/cqrs-htmx/catalog/v2 v2.0.0
	github.com/larsartmann/go-cqrs-lite/catalog/v2 v2.6.0
)

require (
	github.com/go-faster/errors v0.7.1 // indirect
	github.com/go-faster/jx v1.2.0 // indirect
	github.com/go-faster/yaml v0.4.6 // indirect
	github.com/larsartmann/go-error-family v0.4.0 // indirect
	github.com/segmentio/asm v1.2.1 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/sys v0.46.0 // indirect
)

replace github.com/larsartmann/cqrs-htmx/catalog/v2 => ../../catalog
