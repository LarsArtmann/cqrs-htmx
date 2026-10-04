package dashboardui

import (
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	memorystorage "github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

func TestAutodetect_MemoryStoreWiresReadInterfaces(t *testing.T) {
	store := memorystorage.NewMemoryStore()

	config, err := Autodetect(store)
	if err != nil {
		t.Fatalf("Autodetect: %v", err)
	}

	if config.EventSource == nil {
		t.Error("EventSource should be detected on the memory store")
	}

	if config.Journal == nil {
		t.Error("Journal should be detected on the memory store")
	}

	if config.SeekableJournal != nil {
		t.Error("SeekableJournal should stay nil — the memory store does not implement it")
	}

	if config.ProjectionHost != nil {
		t.Error("ProjectionHost should stay nil")
	}

	if config.EventBus != nil {
		t.Error("EventBus should stay nil")
	}
}

func TestAutodetect_CapabilitiesMatchExplicitConfig(t *testing.T) {
	store := memorystorage.NewMemoryStore()

	auto, err := Autodetect(store)
	if err != nil {
		t.Fatalf("Autodetect: %v", err)
	}

	autoDash, err := New(auto)
	if err != nil {
		t.Fatalf("New(autodetected): %v", err)
	}

	explicitDash, err := New(Config{EventSource: store, Journal: store})
	if err != nil {
		t.Fatalf("New(explicit): %v", err)
	}

	if autoDash.Capabilities() != explicitDash.Capabilities() {
		t.Errorf(
			"capabilities diverge: autodetect %+v, explicit %+v",
			autoDash.Capabilities(),
			explicitDash.Capabilities(),
		)
	}
}

func TestAutodetect_BareValueIsRejection(t *testing.T) {
	config, err := Autodetect(struct{ Name string }{Name: "not a store"})
	if err == nil {
		t.Fatal("Autodetect should reject a value implementing no read interface")
	}

	if config.EventSource != nil || config.Journal != nil || config.SeekableJournal != nil {
		t.Errorf("config should carry no read interfaces on error, got %+v", config)
	}

	if msg := err.Error(); !strings.Contains(msg, "implements none of the interfaces") {
		t.Errorf("error message %q should name the missing read interfaces", msg)
	}
}

func TestAutodetect_NilStoreIsRejection(t *testing.T) {
	if _, err := Autodetect(nil); err == nil {
		t.Fatal("Autodetect(nil) should be a rejection")
	}
}

func TestAutodetect_MinimalConfigFlowsThroughNew(t *testing.T) {
	store := memorystorage.NewMemoryStore()

	config, err := Autodetect(store)
	if err != nil {
		t.Fatalf("Autodetect: %v", err)
	}

	config.Title = "Tiny App"
	config.BasePath = "/obs"
	config.ReadOnly = true

	dash, err := New(config)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if got := dash.Config().Title; got != "Tiny App" {
		t.Errorf("Title = %q, want %q", got, "Tiny App")
	}

	if !dash.Capabilities().HasEventRead() {
		t.Error("event read capability missing after autodetect + New")
	}
}

// interfaceProbe confirms Autodetect wires every probed interface, using
// narrow fakes that each implement exactly one interface.
type (
	probeJournal struct{ event.Journal }
	probeBus     struct{ event.Bus }
)

func TestAutodetect_WiresEachImplementedInterface(t *testing.T) {
	journal := probeJournal{}

	config, err := Autodetect(journal)
	if err != nil {
		t.Fatalf("Autodetect: %v", err)
	}

	if config.Journal == nil {
		t.Error("Journal should be detected on a Journal-only value")
	}

	if config.EventSource != nil {
		t.Error("EventSource should stay nil — the probe does not implement it")
	}
}

func TestAutodetect_BusOnlyValueStillNeedsReadInterface(t *testing.T) {
	if _, err := Autodetect(probeBus{}); err == nil {
		t.Fatal("a Bus-only value has no read interface and must be rejected")
	}
}
