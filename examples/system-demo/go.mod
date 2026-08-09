module github.com/larsartmann/cqrs-htmx/examples/system-demo

go 1.26.5

require (
	github.com/larsartmann/cqrs-htmx/identity-model/v4 v4.0.0
	systemadapter "github.com/larsartmann/cqrs-htmx/systemadapter/v4"
	github.com/larsartmann/go-cqrs-lite/id/v4 v4.2.0
	github.com/larsartmann/go-cqrs-lite/system/v4 v4.1.0
)

replace github.com/larsartmann/cqrs-htmx => ../..

replace github.com/larsartmann/cqrs-htmx/identity-model/v4 => ../../identity-model

replace github.com/larsartmann/cqrs-htmx/systemadapter/v4 => ../../systemadapter

replace github.com/larsartmann/cqrs-htmx/usermgmt/v4 => ../../usermgmt
