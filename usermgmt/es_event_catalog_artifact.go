package usermgmt

import (
	_ "embed"

	"encoding/json/v2"

	"github.com/larsartmann/cqrs-htmx/v4/eventcatalog"
)

// eventCatalogArtifact is the committed, generated serialization of the
// served event catalog — exactly the bytes [DefaultEventCatalog].JSON()
// produces. The serving path loads this artifact instead of running
// registration code, so GET /events/catalog serves the reviewed, diffable
// artifact; TestEventCatalogArtifactInSync fails when the artifact and the
// specs table drift apart. Regenerate with:
//
//	go test ./usermgmt/ -run TestEventCatalogArtifactInSync -update
//
//go:embed event-catalog.json
var eventCatalogArtifact []byte

// servedEventCatalog builds the served catalog from the committed artifact.
// A malformed artifact panics at startup: an impossible state while the sync
// test stays green, and the honest failure if it ever is not.
//
// The parse loop is local (not eventcatalog.NewFromJSON) because this
// module's hermetic build resolves the PUBLISHED root pin, and the loader
// shipped there only on the next root train — switch to it at the pin bump.
func servedEventCatalog() *eventcatalog.Catalog {
	var events []eventcatalog.EventMetadata

	if err := json.Unmarshal(eventCatalogArtifact, &events); err != nil {
		panic(err)
	}

	catalog := eventcatalog.New()

	for _, meta := range events {
		catalog.Register(meta)
	}

	return catalog
}
