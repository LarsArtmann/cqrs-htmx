// Package dashboardui provides a plug-in CQRS/Event-Sourcing observability
// dashboard for applications built on go-cqrs-lite and cqrs-htmx.
//
// The dashboard composes templ-components for the UI and reads from
// go-cqrs-lite introspection interfaces (EventSource, Journal,
// SeekableJournal, StreamReader, projectionhost.Host, DeadLetterStore,
// SnapshotStore, CommandJournal, QueryJournal).
//
// Each panel is conditionally active based on which interfaces the consumer
// provides. The dashboard auto-detects available capabilities and shows only
// relevant panels.
package dashboardui
