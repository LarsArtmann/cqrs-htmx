package integration_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/larsartmann/cqrs-htmx/datastar/v4"
	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-sse"
	"github.com/stretchr/testify/require"
)

// TestDatastarRouteSplitCoexistence proves the fullstack-wiring.md
// "Route-split coexistence" recipe (ADR-0050): one shared go-sse hub behind
// a root cqrs-htmx Broadcaster and a datastar Broadcaster, both mounted as
// separate endpoints, with a single broadcast reaching both transports.
func TestDatastarRouteSplitCoexistence(t *testing.T) {
	// 1. One shared fan-out hub (what setup's Bundle.Broadcaster embeds).
	hub := sse.NewBroadcaster[sse.Event]()
	t.Cleanup(hub.Close)

	// 2. Wrap it for each transport — the recipe's exact constructors.
	htmxProvider := cqrshtmx.NewBroadcasterFromHub(hub)
	dsProvider := datastar.NewBroadcasterFromHub(hub)

	// 3. Mount both endpoints the recipe mounts (compile-proof of the
	// documented wiring; ServeSSE/ServeHTTP are the handler values).
	mux := http.NewServeMux()
	mux.Handle("GET /datastar.js", datastar.ScriptHandler())
	mux.HandleFunc("GET /sse", htmxProvider.ServeSSE)
	mux.Handle("GET /ds/events", gate401(dsProvider))

	// 4. Optional bridge mapping from the recipe compiles.
	bridge := datastar.NewEventBridge(dsProvider)
	bridge.Map("usermgmt.user.registered", func(e event.Event) (datastar.Patch, error) {
		return datastar.ElementsPatch("<tr><td>new user</td></tr>",
			datastar.WithSelectorID("user-rows"), datastar.WithModeAppend()), nil
	})

	// 5. Subscribe through both adapters, broadcast once on the HTMX side.
	htmxCh := htmxProvider.Subscribe()
	t.Cleanup(func() { htmxProvider.Unsubscribe(htmxCh) })
	dsCh := dsProvider.Subscribe()
	t.Cleanup(func() { dsProvider.Unsubscribe(dsCh) })

	htmxProvider.Broadcast(sse.Event{Event: "update", Data: "<div>Hi</div>"})

	select {
	case evt := <-htmxCh:
		require.Equal(t, "update", evt.Event)
	case <-timeAfter():
		t.Fatal("HTMX subscriber did not receive the broadcast")
	}

	select {
	case evt := <-dsCh:
		require.Equal(t, "update", evt.Event)
	case <-timeAfter():
		t.Fatal("DataStar subscriber did not receive the shared-hub broadcast")
	}

	// 6. The script endpoint serves the SDK JS (recipe mount sanity).
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/datastar.js", nil))
	require.Equal(t, http.StatusOK, w.Code)
}

// gate401 stands in for the consumer's session gate in the recipe: the
// DataStar feed must not be reachable unauthenticated (ADR-0050 posture).
func gate401(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Session") == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// timeAfter bounds subscriber assertions so a broken hub fails fast.
func timeAfter() <-chan time.Time {
	return time.After(2 * time.Second)
}
