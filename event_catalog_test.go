package cqrshtmx_test

import (
	"encoding/json/v2"
	"strings"
	"testing"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
)

func TestEventCatalog_RegisterAndEvents(t *testing.T) {
	catalog := cqrshtmx.NewEventCatalog()

	catalog.Register(cqrshtmx.EventMetadata{
		Type:          "UserRegistered",
		Aggregate:     "User",
		SchemaVersion: 2,
		Description:   "A user registered",
		PayloadFields: []cqrshtmx.PayloadField{
			{Name: "email", Type: "string", Required: true},
			{Name: "display_name", Type: "string"},
		},
	})

	catalog.Register(cqrshtmx.EventMetadata{
		Type:          "MemberAdded",
		Aggregate:     "Membership",
		SchemaVersion: 2,
	})

	events := catalog.Events()
	if len(events) != 2 {
		t.Fatalf("Events() returned %d events, want 2", len(events))
	}

	if events[0].Type != "UserRegistered" {
		t.Errorf("first event type = %q, want %q", events[0].Type, "UserRegistered")
	}

	if events[1].Aggregate != "Membership" {
		t.Errorf("second event aggregate = %q, want %q", events[1].Aggregate, "Membership")
	}
}

func TestEventCatalog_RegisterDuplicateReplaces(t *testing.T) {
	catalog := cqrshtmx.NewEventCatalog()

	catalog.Register(cqrshtmx.EventMetadata{
		Type:          "UserRegistered",
		Aggregate:     "User",
		SchemaVersion: 1,
		Description:   "original",
	})

	catalog.Register(cqrshtmx.EventMetadata{
		Type:          "UserRegistered",
		Aggregate:     "User",
		SchemaVersion: 2,
		Description:   "updated",
	})

	events := catalog.Events()
	if len(events) != 1 {
		t.Fatalf("duplicate Register produced %d events, want 1", len(events))
	}

	if events[0].Description != "updated" {
		t.Errorf("description = %q, want %q (last registration should win)", events[0].Description, "updated")
	}

	if events[0].SchemaVersion != 2 {
		t.Errorf("schema_version = %d, want 2", events[0].SchemaVersion)
	}
}

func TestEventCatalog_EventsReturnsCopy(t *testing.T) {
	catalog := cqrshtmx.NewEventCatalog()
	catalog.Register(cqrshtmx.EventMetadata{Type: "TestEvent", Aggregate: "Test"})

	first := catalog.Events()
	first[0].Type = "MUTATED"

	second := catalog.Events()
	if second[0].Type != "TestEvent" {
		t.Error("Events() did not return a copy — mutating the slice affected internal state")
	}
}

func TestEventCatalog_JSON(t *testing.T) {
	catalog := cqrshtmx.NewEventCatalog()

	catalog.Register(cqrshtmx.EventMetadata{
		Type:          "UserRegistered",
		Aggregate:     "User",
		SchemaVersion: 2,
		Description:   "A user registered",
		PayloadFields: []cqrshtmx.PayloadField{
			{Name: "email", Type: "string", Required: true},
		},
	})

	catalog.Register(cqrshtmx.EventMetadata{
		Type:          "TenantCreated",
		Aggregate:     "Tenant",
		SchemaVersion: 2,
	})

	data, err := catalog.JSON()
	if err != nil {
		t.Fatalf("JSON() returned error: %v", err)
	}

	body := string(data)

	for _, want := range []string{
		`"type": "UserRegistered"`,
		`"aggregate": "User"`,
		`"schema_version": 2`,
		`"type": "TenantCreated"`,
		`"aggregate": "Tenant"`,
		`"email"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("JSON output missing %q\nbody: %s", want, body)
		}
	}

	var events []cqrshtmx.EventMetadata
	if err := json.Unmarshal(data, &events); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	if len(events) != 2 {
		t.Errorf("unmarshaled %d events, want 2", len(events))
	}
}

func TestEventCatalog_JSONEmpty(t *testing.T) {
	catalog := cqrshtmx.NewEventCatalog()

	data, err := catalog.JSON()
	if err != nil {
		t.Fatalf("JSON() on empty catalog returned error: %v", err)
	}

	body := strings.TrimSpace(string(data))
	if body != "[]" {
		t.Errorf("empty catalog JSON = %q, want %q", body, "[]")
	}
}
