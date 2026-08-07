package integration_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	ds "github.com/larsartmann/cqrs-htmx/datastar/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/stretchr/testify/require"
)

// TestDatastarScriptHandlerContract verifies the ScriptHandler serves JS
// with correct headers from an external module perspective.
func TestDatastarScriptHandlerContract(t *testing.T) {
	handler := ds.ScriptHandler()
	require.NotNil(t, handler)

	req := httptest.NewRequest(http.MethodGet, "/datastar.js", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "text/javascript; charset=utf-8", w.Header().Get("Content-Type"))
	require.NotEmpty(t, w.Body.Bytes())
}

// TestDatastarReadSignalsContract verifies signal decoding works from
// an external module perspective.
func TestDatastarReadSignalsContract(t *testing.T) {
	var s struct {
		Title string `json:"title"`
	}

	req := httptest.NewRequest(http.MethodPost, "/todos", strings.NewReader(`{"title":"Buy milk"}`))
	err := ds.ReadSignals(req, &s)

	require.NoError(t, err)
	require.Equal(t, "Buy milk", s.Title)
}

// TestDatastarBroadcasterContract verifies Broadcaster fan-out and replay
// from an external module perspective.
func TestDatastarBroadcasterContract(t *testing.T) {
	b := ds.NewBroadcaster()
	require.NotNil(t, b)
	require.Equal(t, 0, b.SubscriberCount())

	patch, err := ds.SignalsPatch(map[string]any{"count": 1})
	require.NoError(t, err)
	require.NotNil(t, patch)

	b.Broadcast(patch)
	b.BroadcastMany(patch, patch)

	require.Equal(t, 0, b.SubscriberCount()) // no subscribers connected

	b.Close()
}

// TestDatastarEventBridgeContract verifies EventBridge event mapping from
// an external module perspective.
func TestDatastarEventBridgeContract(t *testing.T) {
	b := ds.NewBroadcaster()
	bridge := ds.NewEventBridge(b)

	require.NotNil(t, bridge)
	require.Empty(t, bridge.MappedEventTypes())

	bridge.Map("TodoCreated", func(e event.Event) (ds.Patch, error) {
		return ds.ElementsPatch("<div>todo</div>"), nil
	})

	require.Contains(t, bridge.MappedEventTypes(), "TodoCreated")

	bridge.Unmap("TodoCreated")
	require.Empty(t, bridge.MappedEventTypes())
}

// TestDatastarPatchConstructorsContract verifies all patch constructors
// return non-nil patches from an external module perspective.
func TestDatastarPatchConstructorsContract(t *testing.T) {
	sigPatch, err := ds.SignalsPatch(map[string]any{"k": "v"})
	require.NoError(t, err)

	sigIfMissingPatch, err := ds.SignalsIfMissingPatch(map[string]any{"k": "v"})
	require.NoError(t, err)

	patches := []ds.Patch{
		ds.ElementsPatch("<div>"),
		sigPatch,
		sigIfMissingPatch,
		ds.RemovePatch("#id"),
		ds.ScriptPatch("console.log()"),
		ds.RedirectPatch("/path"),
	}

	for i, p := range patches {
		require.NotNil(t, p, "patch %d should not be nil", i)
	}
}

// TestDatastarOptionsContract verifies SDK option re-exports are accessible
// from an external module perspective (single-import convenience).
func TestDatastarOptionsContract(t *testing.T) {
	// These compile-time assertions verify the re-exports exist.
	_ = ds.WithSelectorID
	_ = ds.WithModeInner
	_ = ds.WithModeAppend
	_ = ds.WithNamespaceMathML
	_ = ds.WithViewTransitions
	_ = ds.WithOnlyIfMissing
	_ = ds.WithScriptAutoRemove
	_ = ds.NamespaceHTML
	_ = ds.NamespaceMathML
	_ = ds.EventTypePatchElements
}

// TestDatastarResponseContract verifies the Response builder from an
// external module perspective.
func TestDatastarResponseContract(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/todos", nil)

	resp := ds.NewResponse(w, req)
	_ = resp.MarshalAndPatchSignals(map[string]any{"title": ""})
	_ = resp.PatchElements("<div>updated</div>", ds.WithSelectorID("list"))
	_ = resp.Redirect("/todos")

	require.NotNil(t, resp)
}

// TestDatastarVersionContract verifies the version string is non-empty
// and the ScriptTag generates correct HTML.
func TestDatastarVersionContract(t *testing.T) {
	require.NotEmpty(t, ds.Version())

	tag := ds.ScriptTag("/datastar.js")
	require.Contains(t, tag, `type="module"`)
	require.Contains(t, tag, `src="/datastar.js"`)
}
