package datastar_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	ds "github.com/larsartmann/cqrs-htmx/datastar/v4"
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

	body := w.Body.String()
	require.Contains(t, body, "event: datastar-patch-elements")
	require.Contains(t, body, "elements <div>hello</div>")
}

func TestResponsePatchSignals(t *testing.T) {
	t.Parallel()

	w, resp := newTestResponse(t)

	err := resp.PatchSignals([]byte(`{"count":1}`))
	require.NoError(t, err)

	body := w.Body.String()
	require.Contains(t, body, "event: datastar-patch-signals")
	require.Contains(t, body, "count")
}

func TestResponseMarshalAndPatchSignals(t *testing.T) {
	t.Parallel()

	w, resp := newTestResponse(t)

	err := resp.MarshalAndPatchSignals(map[string]any{"total": 5, "label": "items"})
	require.NoError(t, err)

	body := w.Body.String()
	require.Contains(t, body, "event: datastar-patch-signals")
	require.Contains(t, body, "total")
	require.Contains(t, body, "5")
}

func TestResponseExecuteScript(t *testing.T) {
	t.Parallel()

	w, resp := newTestResponse(t)

	err := resp.ExecuteScript("alert('done')")
	require.NoError(t, err)

	body := w.Body.String()
	require.Contains(t, body, "event: datastar-patch-elements")
	require.Contains(t, body, "alert('done')")
}

func TestResponseRemoveElement(t *testing.T) {
	t.Parallel()

	w, resp := newTestResponse(t)

	err := resp.RemoveElement("#stale")
	require.NoError(t, err)

	body := w.Body.String()
	require.Contains(t, body, "selector #stale")
	require.Contains(t, body, "mode remove")
}

func TestResponseRemoveElementByID(t *testing.T) {
	t.Parallel()

	w, resp := newTestResponse(t)

	err := resp.RemoveElementByID("item-3")
	require.NoError(t, err)

	body := w.Body.String()
	require.Contains(t, body, "selector #item-3")
	require.Contains(t, body, "mode remove")
}

func TestResponseRedirect(t *testing.T) {
	t.Parallel()

	w, resp := newTestResponse(t)

	err := resp.Redirect("/home")
	require.NoError(t, err)

	body := w.Body.String()
	require.Contains(t, body, "window.location.href")
	require.Contains(t, body, "/home")
}

func TestResponseMultiplePatches(t *testing.T) {
	t.Parallel()

	w, resp := newTestResponse(t)

	require.NoError(t, resp.PatchElements("<div>first</div>"))
	require.NoError(t, resp.PatchSignals([]byte(`{"step":1}`)))
	require.NoError(t, resp.ExecuteScript("console.log('done')"))

	body := w.Body.String()

	// PatchElements + ExecuteScript both produce patch-elements events;
	// ExecuteScript wraps the script in a <script> element and sends it via
	// patch-elements with selector=body, mode=append (matching the DataStar SDK).
	patchElementsCount := strings.Count(body, "event: datastar-patch-elements")
	patchSignalsCount := strings.Count(body, "event: datastar-patch-signals")

	require.Equal(t, 2, patchElementsCount, "PatchElements + ExecuteScript should produce 2 patch-elements events")
	require.Equal(t, 1, patchSignalsCount)
	require.Contains(t, body, "console.log('done')")
}
