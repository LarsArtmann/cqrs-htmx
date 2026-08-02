package datastar_test

import (
	"context"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	ds "github.com/larsartmann/cqrs-htmx/datastar/v4"
	"github.com/stretchr/testify/require"
)

// mockTemplComponent implements ds.TemplComponent for testing.
type mockTemplComponent struct {
	html string
}

func (m mockTemplComponent) Render(_ context.Context, w io.Writer) error {
	_, err := w.Write([]byte(m.html))
	return err
}

// applyPatch is a test helper that applies a patch through the public Response
// API and returns the resulting SSE body.
func applyPatch(t *testing.T, patch ds.Patch) string {
	t.Helper()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/events", nil)

	ds.NewResponse(w, req).ApplyPatches(patch)

	return w.Body.String()
}

func TestElementsPatchProducesSSE(t *testing.T) {
	t.Parallel()

	body := applyPatch(t, ds.ElementsPatch(`<div id="todo-1">Hello</div>`))

	require.Contains(t, body, "datastar-patch-elements")
	require.Contains(t, body, "todo-1")
}

func TestElementsPatchWithOptions(t *testing.T) {
	t.Parallel()

	body := applyPatch(t, ds.ElementsPatch("<li>New item</li>", ds.WithSelectorID("todo-list"), ds.WithModeAppend()))

	require.Contains(t, body, "todo-list")
	require.Contains(t, body, "append")
}

func TestElementsTemplPatchProducesSSE(t *testing.T) {
	t.Parallel()

	component := mockTemplComponent{html: "<div>Rendered</div>"}
	body := applyPatch(t, ds.ElementsTemplPatch(component, ds.WithSelectorID("target")))

	require.Contains(t, body, "Rendered")
}

func TestSignalsPatchProducesSSE(t *testing.T) {
	t.Parallel()

	body := applyPatch(t, ds.SignalsPatch(map[string]any{
		"notification": map[string]string{"level": "success", "message": "Created!"},
		"count":        42,
	}))

	require.Contains(t, body, "datastar-patch-signals")
	require.Contains(t, body, "Created!")
	require.Contains(t, body, "42")
}

func TestSignalsIfMissingPatchProducesSSE(t *testing.T) {
	t.Parallel()

	body := applyPatch(t, ds.SignalsIfMissingPatch(map[string]any{"initialized": true}))

	require.Contains(t, body, "onlyIfMissing")
	require.Contains(t, body, "true")
}

func TestRemovePatchProducesSSE(t *testing.T) {
	t.Parallel()

	body := applyPatch(t, ds.RemovePatch("#todo-123"))

	require.Contains(t, body, "remove")
	require.Contains(t, body, "todo-123")
}

func TestScriptPatchProducesSSE(t *testing.T) {
	t.Parallel()

	body := applyPatch(t, ds.ScriptPatch("console.log('hello')"))

	require.Contains(t, body, "console.log")
}

func TestRedirectPatchProducesSSE(t *testing.T) {
	t.Parallel()

	body := applyPatch(t, ds.RedirectPatch("/dashboard"))

	require.Contains(t, body, "/dashboard")
}

func TestMultiplePatchesInOrder(t *testing.T) {
	t.Parallel()

	patches := []ds.Patch{
		ds.SignalsPatch(map[string]any{"step": 1}),
		ds.ElementsPatch("<div>first</div>"),
		ds.SignalsPatch(map[string]any{"step": 2}),
		ds.ElementsPatch("<div>second</div>"),
	}

	var sb strings.Builder
	sb.WriteString(applyPatch(t, patches[0]))
	for _, p := range patches[1:] {
		sb.WriteString(applyPatch(t, p))
	}

	body := sb.String()

	require.Contains(t, body, "first")
	require.Contains(t, body, "second")
}
