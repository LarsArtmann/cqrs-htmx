package datastar_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	ds "github.com/larsartmann/cqrs-htmx/datastar/v4"
	"github.com/stretchr/testify/require"
)

func TestNewResponse(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/events", nil)

	resp := ds.NewResponse(w, req)
	require.NotNil(t, resp)
}

func TestResponsePatchElements(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/events", nil)
	resp := ds.NewResponse(w, req)

	err := resp.PatchElements("<div>hello</div>", ds.WithSelectorID("feed"))
	require.NoError(t, err)

	body := w.Body.String()
	require.Contains(t, body, "event: datastar-patch-elements")
	require.Contains(t, body, "elements <div>hello</div>")
}

func TestResponsePatchSignals(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/events", nil)
	resp := ds.NewResponse(w, req)

	err := resp.PatchSignals([]byte(`{"count":1}`))
	require.NoError(t, err)

	body := w.Body.String()
	require.Contains(t, body, "event: datastar-patch-signals")
	require.Contains(t, body, "count")
}

func TestResponseMarshalAndPatchSignals(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/events", nil)
	resp := ds.NewResponse(w, req)

	err := resp.MarshalAndPatchSignals(map[string]any{"total": 5, "label": "items"})
	require.NoError(t, err)

	body := w.Body.String()
	require.Contains(t, body, "event: datastar-patch-signals")
	require.Contains(t, body, "total")
	require.Contains(t, body, "5")
}

func TestResponseExecuteScript(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/events", nil)
	resp := ds.NewResponse(w, req)

	err := resp.ExecuteScript("alert('done')")
	require.NoError(t, err)

	body := w.Body.String()
	require.Contains(t, body, "event: datastar-execute-script")
	require.Contains(t, body, "alert('done')")
}

func TestResponseRemoveElement(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/events", nil)
	resp := ds.NewResponse(w, req)

	err := resp.RemoveElement("#stale")
	require.NoError(t, err)

	body := w.Body.String()
	require.Contains(t, body, "selector #stale")
	require.Contains(t, body, "mode remove")
}

func TestResponseRemoveElementByID(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/events", nil)
	resp := ds.NewResponse(w, req)

	err := resp.RemoveElementByID("item-3")
	require.NoError(t, err)

	body := w.Body.String()
	require.Contains(t, body, "selector #item-3")
	require.Contains(t, body, "mode remove")
}

func TestResponseRedirect(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/events", nil)
	resp := ds.NewResponse(w, req)

	err := resp.Redirect("/home")
	require.NoError(t, err)

	body := w.Body.String()
	require.Contains(t, body, "window.location.href")
	require.Contains(t, body, "/home")
}

func TestResponseMultiplePatches(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/events", nil)
	resp := ds.NewResponse(w, req)

	require.NoError(t, resp.PatchElements("<div>first</div>"))
	require.NoError(t, resp.PatchSignals([]byte(`{"step":1}`)))
	require.NoError(t, resp.ExecuteScript("console.log('done')"))

	body := w.Body.String()

	patchElementsCount := strings.Count(body, "event: datastar-patch-elements")
	patchSignalsCount := strings.Count(body, "event: datastar-patch-signals")
	scriptCount := strings.Count(body, "event: datastar-execute-script")

	require.Equal(t, 1, patchElementsCount)
	require.Equal(t, 1, patchSignalsCount)
	require.Equal(t, 1, scriptCount)
}
