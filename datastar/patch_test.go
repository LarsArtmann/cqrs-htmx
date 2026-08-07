package datastar_test

import (
	"strings"
	"testing"

	ds "github.com/larsartmann/cqrs-htmx/datastar/v4"
	"github.com/stretchr/testify/require"
)

func TestElementsPatch(t *testing.T) {
	t.Parallel()

	patch := ds.ElementsPatch("<div>hello</div>", ds.WithSelectorID("feed"))
	require.NotNil(t, patch)

	evt := patch.Event()
	require.Equal(t, "datastar-patch-elements", evt.Event)
	require.Contains(t, evt.Data, "elements <div>hello</div>")
}

func TestElementsPatchWithOptions(t *testing.T) {
	t.Parallel()

	patch := ds.ElementsPatch("<li>item</li>",
		ds.WithSelectorID("list"),
		ds.WithModeAppend(),
	)
	evt := patch.Event()

	require.Contains(t, evt.Data, "selector #list")
	require.Contains(t, evt.Data, "mode append")
}

func TestSignalsPatch(t *testing.T) {
	t.Parallel()

	patch, err := ds.SignalsPatch(map[string]any{"count": 1})
	require.NoError(t, err)
	require.NotNil(t, patch)

	evt := patch.Event()
	require.Equal(t, "datastar-patch-signals", evt.Event)
	require.Contains(t, evt.Data, "count")
}

func TestSignalsPatchError(t *testing.T) {
	t.Parallel()

	// A value that cannot be marshaled to JSON
	_, err := ds.SignalsPatch(make(chan int))
	require.Error(t, err)
}

func TestSignalsIfMissingPatch(t *testing.T) {
	t.Parallel()

	patch, err := ds.SignalsIfMissingPatch(map[string]any{"initialized": true})
	require.NoError(t, err)
	require.NotNil(t, patch)

	evt := patch.Event()
	require.Equal(t, "datastar-patch-signals", evt.Event)
	require.Contains(t, evt.Data, "onlyIfMissing true")
}

func TestRemovePatch(t *testing.T) {
	t.Parallel()

	patch := ds.RemovePatch("#old-item")
	require.NotNil(t, patch)

	evt := patch.Event()
	require.Equal(t, "datastar-patch-elements", evt.Event)
	require.Contains(t, evt.Data, "selector #old-item")
	require.Contains(t, evt.Data, "mode remove")
}

func TestRemoveByIDPatch(t *testing.T) {
	t.Parallel()

	patch := ds.RemoveByIDPatch("todo-1")
	require.NotNil(t, patch)

	evt := patch.Event()
	require.Contains(t, evt.Data, "selector #todo-1")
	require.Contains(t, evt.Data, "mode remove")
}

func TestScriptPatch(t *testing.T) {
	t.Parallel()

	patch := ds.ScriptPatch("console.log('hi')", ds.WithScriptAutoRemove(true))
	require.NotNil(t, patch)

	evt := patch.Event()
	require.Equal(t, "datastar-execute-script", evt.Event)
	require.Contains(t, evt.Data, "console.log('hi')")
}

func TestRedirectPatch(t *testing.T) {
	t.Parallel()

	patch := ds.RedirectPatch("/dashboard")
	require.NotNil(t, patch)

	evt := patch.Event()
	require.Equal(t, "datastar-execute-script", evt.Event)
	require.Contains(t, evt.Data, "window.location.href")
	require.Contains(t, evt.Data, "/dashboard")
}

func TestPatchEventTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		patch     ds.Patch
		eventType string
	}{
		{"elements", ds.ElementsPatch("<div>"), "datastar-patch-elements"},
		{"remove", ds.RemovePatch("#x"), "datastar-patch-elements"},
		{"script", ds.ScriptPatch("1+1"), "datastar-execute-script"},
		{"redirect", ds.RedirectPatch("/"), "datastar-execute-script"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tt.eventType, tt.patch.Event().Event)
		})
	}
}

func TestElementsPatchMultiLineHTML(t *testing.T) {
	t.Parallel()

	html := "<div>\n<span>line1</span>\n<span>line2</span>\n</div>"
	patch := ds.ElementsPatch(html)
	evt := patch.Event()

	lines := strings.Split(evt.Data, "\n")
	var elementLines int
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "data: elements") {
			elementLines++
		}
	}

	require.Equal(t, 4, elementLines, "multi-line HTML should produce one data: elements line per line")
}
