package ordersaga_test

import (
	"encoding/json/v2"
	"os"
	"slices"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/systemscenario/v4"

	"github.com/larsartmann/cqrs-htmx/examples/catalog-demo/ordersaga"
)

// eventUniverse mirrors the `--events-file` JSON shape the gfl CLI's drift
// gate consumes (system + declared event names). Parsed here with the
// stdlib so this public repo never imports the private GFL module — the
// bridge is one-way by design.
type eventUniverse struct {
	System string   `json:"system"`
	Events []string `json:"events"`
}

// TestEventsFixtureInSync pins order-saga-events.json to the real
// DomainConfig.Events of the ordersaga domain: the fixture must never
// drift from the code it claims to snapshot, or the demo's
// `gfl validate --events-file` gate would check against a stale universe.
func TestEventsFixtureInSync(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile("order-saga-events.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	var universe eventUniverse
	if err := json.Unmarshal(data, &universe); err != nil {
		t.Fatalf("fixture is not a valid event universe: %v", err)
	}

	if universe.System != "ordersaga" {
		t.Errorf("fixture system: want %q, got %q", "ordersaga", universe.System)
	}

	if want := ordersaga.EventNames(); !slices.Equal(universe.Events, want) {
		t.Errorf("fixture events drifted from ordersaga.Domain().Events:\nfixture: %v\ncode:    %v",
			universe.Events, want)
	}
}

// TestDomainChainRunsTheSaga executes the domain directly (no GFL, no
// generator): the fixture itself must be a working deriver chain before
// the generated specs can mean anything.
func TestDomainChainRunsTheSaga(t *testing.T) {
	t.Parallel()

	sc := systemscenario.System(t, t.Context(), ordersaga.Domain(), systemscenario.Memory())

	sc.When(ordersaga.Place()).
		Await().
		Then("order.created", "payment.settled").
		ThenCommands("order.create", "payment.settle")
}
