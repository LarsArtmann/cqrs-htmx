//cqrs-lint:ignore(E009) setup is the composition root — it wires transport, domain, and UI by design
//cqrs-lint:ignore(E014) setup consumes projections from usermgmt; it does not own them
module github.com/larsartmann/cqrs-htmx/setup/v4

go 1.26.5

require (
	github.com/larsartmann/cqrs-htmx/adminui/v4 v4.7.0
	github.com/larsartmann/cqrs-htmx/dashboardui/v4 v4.2.0
	github.com/larsartmann/cqrs-htmx/loginpage/v4 v4.7.0
	github.com/larsartmann/cqrs-htmx/usermgmt/v4 v4.7.1
	github.com/larsartmann/cqrs-htmx/v4 v4.7.0
	github.com/larsartmann/go-cqrs-lite/event/v4 v4.4.0
	github.com/larsartmann/go-cqrs-lite/storage/memory/v4 v4.3.0
	github.com/larsartmann/httputil v0.11.0
)

replace github.com/larsartmann/cqrs-htmx/v4 => ../

replace github.com/larsartmann/cqrs-htmx/usermgmt/v4 => ../usermgmt

replace github.com/larsartmann/cqrs-htmx/adminui/v4 => ../adminui

replace github.com/larsartmann/cqrs-htmx/dashboardui/v4 => ../dashboardui

replace github.com/larsartmann/cqrs-htmx/loginpage/v4 => ../loginpage
