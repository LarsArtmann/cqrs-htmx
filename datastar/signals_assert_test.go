package datastar_test

import (
	"encoding/json/v2"
	"testing"

	"github.com/larsartmann/go-datastar/datastartest"
	"github.com/stretchr/testify/require"
)

// requireSignalsSemantically asserts the signals payload by VALUE, not by byte
// order. Upstream NewSignalsPatch marshals map-backed signals through Go's
// randomized map iteration (no key sort), so any multi-key JSON object can
// serialize in either order — a byte-exact RequireSignals pin on such a
// payload is flaky by construction (both orders observed live 2026-11-18,
// the CI datastar-submodule red). Single-key payloads stay byte-pinned with
// datastartest.RequireSignals.
func requireSignalsSemantically(t *testing.T, evt datastartest.Event, wantJSON string) {
	t.Helper()

	datastartest.RequireEventType(t, evt, "datastar-patch-signals")

	var got, want any
	require.NoError(t, json.Unmarshal(evt.SignalsJSON(), &got), "got signals: %s", evt.SignalsJSON())
	require.NoError(t, json.Unmarshal([]byte(wantJSON), &want))
	require.Equal(t, want, got, "signals payload mismatch: got %s, want %s", evt.SignalsJSON(), wantJSON)
}
