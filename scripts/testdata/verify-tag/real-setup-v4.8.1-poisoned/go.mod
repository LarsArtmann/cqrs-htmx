//cqrs-lint:ignore(E009) setup is the composition root — it wires transport, domain, and UI by design
//cqrs-lint:ignore(E014) setup consumes projections from usermgmt; it does not own them
module github.com/larsartmann/cqrs-htmx/setup/v4

go 1.26

replace github.com/larsartmann/go-appkit => ../../go-appkit

// DEV-ONLY (remove before the next family tag): setup requires usermgmt
// v4.8.1, which is NOT published (remote max v4.8.0), and adopts
// *usermgmt.Service via Service.Journal()/EventBus() accessors that do not
// exist in any published tag (verified against usermgmt/v4.8.0 on
// 2026-08-29). Strip this replace when the train tags+pushes
// usermgmt/v4.8.x carrying those accessors (see
// docs/runbooks/release-v4.8.0-push-plan.md §4 pattern).
replace github.com/larsartmann/cqrs-htmx/usermgmt/v4 => ../usermgmt

// DEV-ONLY (remove before the next family tag): setup requires dashboardui
// v4.8.1, which is cut locally but NOT pushed (remote max v4.8.0), and uses
// the SSEMaxReplay field that does not exist in dashboardui/v4.8.0
// (verified 2026-08-29). Strip this replace when the train pushes
// dashboardui/v4.8.x carrying SSEMaxReplay.
replace github.com/larsartmann/cqrs-htmx/dashboardui/v4 => ../dashboardui
