package integration_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	memorystorage "github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
	"github.com/larsartmann/go-cqrs-lite/watermill/v4"
	"github.com/stretchr/testify/require"
)

// slowJournal delays every ReadFrom call so the projection drain window is
// observably long. Without it, an in-memory journal drains in microseconds and
// the first /health poll may already observe 200 — making the 503→200
// transition untestable. The delay is real-world honest: it emulates replaying
// a large journal from persistent storage.
type slowJournal struct {
	event.Store
	event.SeekableJournal
	delay time.Duration
	reads atomic.Int64
}

func (j *slowJournal) ReadFrom(ctx context.Context, after id.EventID, limit int) ([]event.Event, error) {
	j.reads.Add(1)
	time.Sleep(j.delay)
	return j.SeekableJournal.ReadFrom(ctx, after, limit)
}

// TestAsyncStartupReadinessLifecycle verifies the full async-startup lifecycle
// end to end: with AsyncStartup=true the service (and its HTTP server) starts
// immediately while projections replay the journal in the background, and the
// readiness endpoint transitions 503 (draining) → 200 (ready) as workers catch
// up. It also verifies read-your-writes after readiness: users registered
// before the restart are queryable from the replayed read models.
func TestAsyncStartupReadinessLifecycle(t *testing.T) {
	ctx := context.Background()

	// Phase 1: build a journal backlog synchronously.
	store := memorystorage.NewMemoryStore()
	seedBus := watermill.NewEventBus()
	seed, err := usermgmt.NewService(usermgmt.ServiceConfig{
		EventStore: store,
		EventBus:   seedBus,
		AuditLog:   usermgmt.NewAuditLog(),
	})
	require.NoError(t, err)

	const seededUsers = 25
	var lastUID cqrshtmx.UserID
	for i := range seededUsers {
		lastUID = cqrshtmx.NewUserID()
		registerTestUser(t, seed, lastUID, fmt.Sprintf("async-startup-%03d@test.com", i))
	}
	seed.Stop()

	// Phase 2: restart on the same journal with async startup and slowed reads.
	journal := &slowJournal{SeekableJournal: store, delay: 15 * time.Millisecond}
	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{
		EventStore:   journal,
		EventBus:     watermill.NewEventBus(),
		AuditLog:     usermgmt.NewAuditLog(),
		AsyncStartup: true,
	})
	require.NoError(t, err)
	t.Cleanup(svc.Stop)

	// Real HTTP server with the readiness gate mounted at /health (the same
	// handler setup.Bundle mounts by default).
	mux := http.NewServeMux()
	mux.Handle("/health", cqrshtmx.ReadinessHandler(cqrshtmx.ProjectionReadinessCheck(svc)))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	getHealth := func() int {
		t.Helper()
		resp, err := http.Get(server.URL + "/health")
		require.NoError(t, err)
		defer resp.Body.Close()
		return resp.StatusCode
	}

	// The server is up and answering immediately (liveness decoupled from
	// readiness) — but not ready: the slowed drain guarantees 503 here.
	require.Equal(t, http.StatusServiceUnavailable, getHealth(),
		"first /health must be 503 while projections drain (got %d delayed reads so far)", journal.reads.Load())

	// Poll until the readiness gate flips to 200.
	deadline := time.Now().Add(30 * time.Second)
	readyCode := getHealth()
	for readyCode != http.StatusOK && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
		readyCode = getHealth()
	}
	require.Equal(t, http.StatusOK, readyCode, "/health never became ready")
	require.Positive(t, journal.reads.Load(), "drain must have replayed the journal via ReadFrom")

	// Read-your-writes after readiness: the user registered in phase 1 is
	// queryable from the replayed read model.
	user, err := svc.GetUser(ctx, lastUID)
	require.NoError(t, err, "seeded user must be readable once projections are live")
	require.Equal(t, fmt.Sprintf("async-startup-%03d@test.com", seededUsers-1), user.Email)

	// Steady state: readiness stays 200 once drained.
	require.Equal(t, http.StatusOK, getHealth())
}
