package usermgmt

import (
	"bytes"
	"flag"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/larsartmann/cqrs-htmx/v4/eventcatalog"
)

// update regenerates event-catalog.json:
// go test ./usermgmt/ -run TestEventCatalogArtifactInSync -update
var catalogUpdate = flag.Bool(
	"update",
	false,
	"rewrite generated golden artifacts",
) //nolint:gochecknoglobals // standard golden-artifact update flag

// TestEventCatalogArtifactInSync pins event-catalog.json (the artifact the
// serving path loads) byte-for-byte to the specs table's rendered catalog.
// The file is generated, never hand-edited; regenerate after any specs,
// payload-struct, or description change with the -update flag.
func TestEventCatalogArtifactInSync(t *testing.T) {
	want, err := DefaultEventCatalog().JSON()
	if err != nil {
		t.Fatalf("render catalog JSON: %v", err)
	}

	if *catalogUpdate {
		if err := os.WriteFile("event-catalog.json", want, 0o600); err != nil {
			t.Fatalf("rewrite event-catalog.json: %v", err)
		}
	}

	raw, err := os.ReadFile("event-catalog.json")
	if err != nil {
		t.Fatalf("read event-catalog.json: %v", err)
	}

	if !bytes.Equal(raw, want) {
		t.Errorf(
			"event-catalog.json drifted from the specs table; regenerate with -update (want %d bytes, artifact has %d)",
			len(want),
			len(raw),
		)
	}
}

// TestEventCatalogServedFromArtifact asserts the integration the artifact
// exists for: the handler over the SERVING catalog serves the artifact bytes
// exactly (byte-identical to the committed JSON, not a re-render), with the
// shared ETag revalidation behavior intact.
func TestEventCatalogServedFromArtifact(t *testing.T) {
	handler, err := eventcatalog.Handler(servedEventCatalog())
	if err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/events/catalog", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	if !bytes.Equal(w.Body.Bytes(), eventCatalogArtifact) {
		t.Errorf("served catalog is not byte-identical to the committed artifact (served %d bytes, artifact %d)",
			w.Body.Len(), len(eventCatalogArtifact))
	}

	etag := w.Header().Get("ETag")
	if etag == "" {
		t.Fatal("expected an ETag on the served catalog")
	}

	revalidating := httptest.NewRequest(http.MethodGet, "/events/catalog", nil)
	revalidating.Header.Set("If-None-Match", etag)

	second := httptest.NewRecorder()
	handler.ServeHTTP(second, revalidating)

	if second.Code != http.StatusNotModified {
		t.Errorf("revalidation status = %d, want %d", second.Code, http.StatusNotModified)
	}
}
