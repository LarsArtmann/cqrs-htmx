package datastar_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	ds "github.com/larsartmann/cqrs-htmx/datastar/v4"
	"github.com/larsartmann/go-datastar/datastartest"
	"github.com/stretchr/testify/require"
)

// newTestResponse builds a recorder-backed Response for GET /events, the
// fixture every response test writes through.
func newTestResponse(t *testing.T) (*httptest.ResponseRecorder, *ds.Response) {
	t.Helper()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/events", nil)

	return w, ds.NewResponse(w, req)
}

// parseEvents decodes the recorder's SSE body into wire-format events via
// datastartest, so assertions run against the parsed protocol structure
// (event types, datalines) instead of raw string fragments.
func parseEvents(t *testing.T, w *httptest.ResponseRecorder) []datastartest.Event {
	t.Helper()

	return datastartest.MustReadEvents(t, strings.NewReader(w.Body.String()))
}

func TestNewResponse(t *testing.T) {
	t.Parallel()

	_, resp := newTestResponse(t)
	require.NotNil(t, resp)
}

func TestResponsePatchElements(t *testing.T) {
	t.Parallel()

	w, resp := newTestResponse(t)

	err := resp.PatchElements("<div>hello</div>", ds.WithSelectorID("feed"))
	require.NoError(t, err)

	events := parseEvents(t, w)
	datastartest.RequireEventCount(t, events, 1)
	datastartest.RequireElements(t, events[0], "#feed", "outer", "<div>hello</div>")
}

func TestResponsePatchSignals(t *testing.T) {
	t.Parallel()

	w, resp := newTestResponse(t)

	err := resp.PatchSignals([]byte(`{"count":1}`))
	require.NoError(t, err)

	events := parseEvents(t, w)
	datastartest.RequireEventCount(t, events, 1)
	datastartest.RequireSignals(t, events[0], `{"count":1}`)
}

func TestResponseMarshalAndPatchSignals(t *testing.T) {
	t.Parallel()

	w, resp := newTestResponse(t)

	err := resp.MarshalAndPatchSignals(map[string]any{"total": 5, "label": "items"})
	require.NoError(t, err)

	events := parseEvents(t, w)
	datastartest.RequireEventCount(t, events, 1)
	datastartest.RequireSignals(t, events[0], `{"total":5,"label":"items"}`)
}

func TestResponseExecuteScript(t *testing.T) {
	t.Parallel()

	w, resp := newTestResponse(t)

	err := resp.ExecuteScript("alert('done')")
	require.NoError(t, err)

	events := parseEvents(t, w)
	datastartest.RequireEventCount(t, events, 1)
	datastartest.RequireScript(t, events[0], "alert('done')")
}

func TestResponseRemoveElement(t *testing.T) {
	t.Parallel()

	w, resp := newTestResponse(t)

	err := resp.RemoveElement("#stale")
	require.NoError(t, err)

	events := parseEvents(t, w)
	datastartest.RequireEventCount(t, events, 1)
	datastartest.RequireElements(t, events[0], "#stale", "remove", "")
}

func TestResponseRemoveElementByID(t *testing.T) {
	t.Parallel()

	w, resp := newTestResponse(t)

	err := resp.RemoveElementByID("item-3")
	require.NoError(t, err)

	events := parseEvents(t, w)
	datastartest.RequireEventCount(t, events, 1)
	datastartest.RequireElements(t, events[0], "#item-3", "remove", "")
}

func TestResponseRedirect(t *testing.T) {
	t.Parallel()

	w, resp := newTestResponse(t)

	err := resp.Redirect("/home")
	require.NoError(t, err)

	events := parseEvents(t, w)
	datastartest.RequireEventCount(t, events, 1)
	require.True(t, events[0].IsScript(), "Redirect should emit a script-bearing patch-elements event, got %q", events[0].Type)
	script := events[0].ScriptContent()
	require.Contains(t, script, "window.location.href")
	require.Contains(t, script, "/home")
}

func TestResponseMultiplePatches(t *testing.T) {
	t.Parallel()

	w, resp := newTestResponse(t)

	require.NoError(t, resp.PatchElements("<div>first</div>"))
	require.NoError(t, resp.PatchSignals([]byte(`{"step":1}`)))
	require.NoError(t, resp.ExecuteScript("console.log('done')"))

	events := parseEvents(t, w)

	// PatchElements + ExecuteScript both produce patch-elements events;
	// ExecuteScript wraps the script in a <script> element and sends it via
	// patch-elements with selector=body, mode=append (matching the DataStar SDK).
	var elementEvents, signalEvents []datastartest.Event
	for _, evt := range events {
		switch {
		case evt.IsElements():
			elementEvents = append(elementEvents, evt)
		case evt.IsSignals():
			signalEvents = append(signalEvents, evt)
		}
	}

	require.Len(t, elementEvents, 2, "PatchElements + ExecuteScript should produce 2 patch-elements events")
	require.Len(t, signalEvents, 1)
	datastartest.RequireScript(t, elementEvents[1], "console.log('done')")
}
