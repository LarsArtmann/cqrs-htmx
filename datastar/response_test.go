package datastar_test

import (
	"context"
	"io"
	"net/http/httptest"
	"testing"

	ds "github.com/larsartmann/cqrs-htmx/datastar/v4"
	"github.com/stretchr/testify/require"
)

type testTemplComponent struct {
	html string
}

func (c testTemplComponent) Render(_ context.Context, w io.Writer) error {
	_, err := w.Write([]byte(c.html))
	return err
}

func TestNewResponsePatchSignals(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/todos", nil)

	resp := ds.NewResponse(w, req).
		PatchSignals(map[string]any{"title": "", "notification": "Created!"})

	body := w.Body.String()
	require.Contains(t, body, "datastar-patch-signals")
	require.Contains(t, body, "Created!")
	require.Same(t, resp, resp.Apply()) // Apply returns same pointer (no-op)
}

func TestResponsePatchElements(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/todos", nil)

	ds.NewResponse(w, req).PatchElements(
		"<div id='todo-1'>Buy milk</div>",
		ds.WithSelectorID("todo-list"),
		ds.WithModeAppend(),
	)

	body := w.Body.String()
	require.Contains(t, body, "datastar-patch-elements")
	require.Contains(t, body, "Buy milk")
	require.Contains(t, body, "todo-list")
}

func TestResponsePatchElementsTempl(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/todos", nil)

	component := testTemplComponent{html: "<div>Templ rendered</div>"}
	ds.NewResponse(w, req).PatchElementsTempl(component, ds.WithSelectorID("target"))

	body := w.Body.String()
	require.Contains(t, body, "Templ rendered")
}

func TestResponseRemoveElement(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/todos/todo-1", nil)

	ds.NewResponse(w, req).RemoveElement("#todo-1")

	body := w.Body.String()
	require.Contains(t, body, "remove")
	require.Contains(t, body, "todo-1")
}

func TestResponseRedirect(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/login", nil)

	ds.NewResponse(w, req).Redirect("/dashboard")

	body := w.Body.String()
	require.Contains(t, body, "/dashboard")
}

func TestResponseExecuteScript(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/action", nil)

	ds.NewResponse(w, req).ExecuteScript("console.log('done')")

	body := w.Body.String()
	require.Contains(t, body, "console.log")
}

func TestResponseConsoleLog(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/action", nil)

	ds.NewResponse(w, req).ConsoleLog("debug-here")

	require.Contains(t, w.Body.String(), "debug-here")
}

func TestResponseConsoleError(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/action", nil)

	ds.NewResponse(w, req).ConsoleError(testError("boom-failed"))

	require.Contains(t, w.Body.String(), "boom-failed")
}

func TestResponseDispatchCustomEvent(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/action", nil)

	ds.NewResponse(w, req).DispatchCustomEvent("cartUpdated", map[string]any{"count": 3})

	require.Contains(t, w.Body.String(), "cartUpdated")
}

func TestResponseReplaceURL(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/action", nil)

	ds.NewResponse(w, req).ReplaceURL("/items/42")

	require.Contains(t, w.Body.String(), "/items/42")
}

func TestResponseReplaceURLInvalidIgnored(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/action", nil)

	ds.NewResponse(w, req).ReplaceURL("%zz") // invalid percent-escape -> silently ignored

	require.NotContains(t, w.Body.String(), "replace-url")
}

func TestResponseRemoveElementByID(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/todos/todo-1", nil)

	ds.NewResponse(w, req).RemoveElementByID("todo-1")

	body := w.Body.String()
	require.Contains(t, body, "remove")
	require.Contains(t, body, "todo-1")
}

func TestResponsePrefetch(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/page", nil)

	ds.NewResponse(w, req).Prefetch("/assets/big.css")

	require.Contains(t, w.Body.String(), "/assets/big.css")
}

func TestResponseChainedMethods(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/todos", nil)

	ds.NewResponse(w, req).
		PatchSignals(map[string]any{"title": ""}).
		PatchElements("<div>New todo</div>", ds.WithSelectorID("list"), ds.WithModeInner()).
		Redirect("/todos")

	body := w.Body.String()
	require.Contains(t, body, "datastar-patch-signals")
	require.Contains(t, body, "datastar-patch-elements")
	require.Contains(t, body, "/todos")
}

func TestResponsePatchSignalsIfMissing(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/events", nil)

	ds.NewResponse(w, req).PatchSignalsIfMissing(map[string]any{"initialized": true})

	body := w.Body.String()
	require.Contains(t, body, "onlyIfMissing")
}

func TestResponseApplyPatches(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/todos", nil)

	patches := []ds.Patch{
		ds.SignalsPatch(map[string]any{"count": 1}),
		ds.ElementsPatch("<div>item</div>"),
		ds.RemovePatch("#old-item"),
	}

	ds.NewResponse(w, req).ApplyPatches(patches...)

	body := w.Body.String()
	require.Contains(t, body, "datastar-patch-signals")
	require.Contains(t, body, "datastar-patch-elements")
	require.Contains(t, body, "remove")
}

func TestResponseSSEAccessor(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/events", nil)

	resp := ds.NewResponse(w, req)
	sse := resp.SSE()

	require.NotNil(t, sse)
	require.NotNil(t, sse.Context())
}

func TestErrorResponse(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/todos", nil)

	ds.ErrorResponse(w, req, testError("title is required"))

	body := w.Body.String()
	require.Contains(t, body, "datastar-patch-signals")
	require.Contains(t, body, "title is required")
	require.Contains(t, body, "error")
}

func TestNotificationResponse(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/todos", nil)

	ds.NotificationResponse(w, req, "success", "Todo created!")

	body := w.Body.String()
	require.Contains(t, body, "success")
	require.Contains(t, body, "Todo created!")
}

type testError string

func (e testError) Error() string { return string(e) }
