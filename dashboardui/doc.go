// Package dashboardui provides a plug-in CQRS/Event-Sourcing observability
// dashboard for applications built on go-cqrs-lite and cqrs-htmx.
//
// The dashboard renders HTML and reads from go-cqrs-lite introspection
// interfaces (EventSource, Journal, SeekableJournal, StreamReader,
// projectionhost.Host, DeadLetterStore, SnapshotStore, CommandJournal,
// QueryJournal).
//
// # Which dashboard?
//
// cqrs-htmx ships two dashboards: dashboardui (this package) introspects the
// event store (journal, aggregates, projections, dead letters, snapshots) and
// is read-only by default;
// [github.com/larsartmann/cqrs-htmx/adminui/v4] manages users, tenants,
// members, and the audit log over a usermgmt service, with write actions.
// They are complementary: most apps mount both, and
// [github.com/larsartmann/cqrs-htmx/setup/v4] wires them in one call.
//
// Each panel is conditionally active based on which interfaces the consumer
// provides. The dashboard auto-detects available capabilities and shows only
// relevant panels.
package dashboardui
