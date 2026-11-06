package cqrshtmx_test

import (
	"testing"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/cqrs-htmx/v4/htmx"
)

// The HTMX script handler and the offline-sync JS handlers share one serving
// routine. Its HTMX side is covered by the htmx subpackage's own suite; these
// specs pin the conditional-GET contract through the root facade for both the
// HTMX and sync assets.

func TestHTMXScriptHandler_IfNoneMatchSpec(t *testing.T) {
	runConditionalGetSpec(t, htmx.ScriptHandler(), "/htmx.js")
}

func TestSyncHandlers_IfNoneMatchSpec(t *testing.T) {
	t.Run("sync-worker", func(t *testing.T) {
		runConditionalGetSpec(t, cqrshtmx.SyncWorkerHandler(), "/sync-worker.js")
	})

	t.Run("sync-client", func(t *testing.T) {
		runConditionalGetSpec(t, cqrshtmx.SyncClientHandler(), "/sync-client.js")
	})
}
