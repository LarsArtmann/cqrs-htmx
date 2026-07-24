package cqrshtmx

import (
	"encoding/json/v2"

	errorfamily "github.com/larsartmann/go-error-family"
)

// PayloadField describes a single field in an event payload.
type PayloadField struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Required bool   `json:"required,omitempty"`
}

// EventMetadata describes a single event type in the catalog. Consumers
// building custom projections use this to discover what events exist, which
// aggregate owns them, and what the payload looks like.
type EventMetadata struct {
	Type          string         `json:"type"`
	Aggregate     string         `json:"aggregate"`
	SchemaVersion int            `json:"schema_version"`
	Description   string         `json:"description,omitempty"`
	PayloadFields []PayloadField `json:"payload_fields,omitempty"`
}

// EventCatalog is a mutable registry of event types and their metadata.
// Register events during application startup (before serving HTTP), then
// pass the catalog to [EventCatalogHandler] for immutable serving.
//
// The catalog is designed for the Published Language pattern (DDD): events
// are part of the public API surface for any consumer building projections.
// This catalog makes that contract explicit and discoverable.
type EventCatalog struct {
	events []EventMetadata
}

// NewEventCatalog returns an empty EventCatalog ready for registration.
func NewEventCatalog() *EventCatalog {
	return &EventCatalog{}
}

// Register adds an event type to the catalog. Call during startup, before
// creating an [EventCatalogHandler]. Duplicate type+aggregate pairs are
// ignored (last registration wins).
func (c *EventCatalog) Register(meta EventMetadata) {
	for i, existing := range c.events {
		if existing.Type == meta.Type && existing.Aggregate == meta.Aggregate {
			c.events[i] = meta
			return
		}
	}
	c.events = append(c.events, meta)
}

// Events returns a copy of all registered event metadata, ordered by
// registration. Safe to call after registration is complete.
func (c *EventCatalog) Events() []EventMetadata {
	result := make([]EventMetadata, len(c.events))
	copy(result, c.events)
	return result
}

// JSON serializes the catalog to indented JSON suitable for serving at a
// catalog endpoint (e.g. GET /events/catalog).
func (c *EventCatalog) JSON() ([]byte, error) {
	data, err := json.MarshalIndent(c.events, "", "  ")
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err,
			"cqrshtmx.event_catalog.serialize",
			"serialize event catalog to JSON")
	}
	return data, nil
}
