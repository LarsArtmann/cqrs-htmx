package cqrshtmx

import (
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

func TestSyncPullDebug(t *testing.T) {
	store := memory.NewMemoryStore()
	events := seedSyncEvents(t, 2)
	appendSyncEvents(t, store, events)
	handler := SyncPullHandler(store)
	rec := doPull(handler, "/sync/pull")
	fmt.Printf("code=%d headers=%v body=%q\n", rec.Code, rec.Header(), rec.Body.String())
}
