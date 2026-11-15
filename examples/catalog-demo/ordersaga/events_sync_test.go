package ordersaga_test

import (
	"os"
	"slices"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/systemscenario/v4"

	"github.com/larsartmann/cqrs-htmx/examples/catalog-demo/ordersaga"
	"gfl-lang/pkg/parser"
)

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

	universe, err := parser.ParseEventUniverse(data)
	if err != nil {
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
