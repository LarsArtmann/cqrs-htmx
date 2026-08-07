module github.com/larsartmann/cqrs-htmx/datastar/v4

go 1.26.5

require (
	github.com/larsartmann/go-cqrs-lite/event/v4 v4.2.0
	github.com/larsartmann/go-cqrs-lite/id/v4 v4.2.0
	github.com/larsartmann/go-datastar v0.0.0
	github.com/larsartmann/go-sse v0.0.0
	github.com/stretchr/testify v1.11.1
)

replace github.com/larsartmann/go-datastar => ../../go-datastar

replace github.com/larsartmann/go-sse => ../../go-sse

