module github.com/larsartmann/cqrs-htmx/setup/v4

go 1.26.7

require (
	github.com/larsartmann/cqrs-htmx/dashboardui/v4 v4.8.1
	github.com/larsartmann/cqrs-htmx/usermgmt/v4 v4.8.1
)

replace github.com/larsartmann/cqrs-htmx/usermgmt/v4 => ../usermgmt
