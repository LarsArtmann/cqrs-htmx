// Package usermgmt event types are defined as event-sourced payloads in
// es_events.go, es_membership_events.go, es_tenant_events.go, and
// es_bot_events.go. Subscribe to the event bus (event.Bus) to receive
// domain events; the legacy EventHandler callback has been removed.
package usermgmt
