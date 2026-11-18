package integration_test

// SSE/event-envelope parity between the legacy usermgmt.Service path and the
// systemadapter system path — W1.3 of the usermgmt→system migration plan
// (docs/planning/2026-11-07_10-45_SUPERB-usermgmt-system-metaengine-migration-plan.md,
// F20). Identical commands are dispatched through BOTH paths, every resulting
// event is run through the one shared SSE encoder (transport.DomainEventToSSE),
// and the envelopes must agree: same event-type sequence, same
// streamType/version metadata per position, identical envelope structure.
// Volatile fields (stream/event IDs, occurrence timestamps) differ per
// dispatch by construction; they are asserted well-formed, never equal.
//
// The golden envelope SHAPE is pinned byte-exactly in the root module
// (transport/event_sse_test.go); this test pins the parity CONTRACT across
// dispatch paths — the migration's read-tier cutover (W3) may replace how
// events are produced, never what the browser envelope looks like.

import (
	"context"
	"encoding/json/v2"
	"testing"
	"time"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	systemadapter "github.com/larsartmann/cqrs-htmx/systemadapter/v4"
	"github.com/larsartmann/cqrs-htmx/v4/transport"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	memorystorage "github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
	"github.com/larsartmann/go-sse"
	"github.com/stretchr/testify/require"
)

// sseEnvelopes converts a stream's events to their decoded SSE envelope
// payloads exactly as a browser would receive them.
func sseEnvelopes(t *testing.T, events []event.Event) []transport.EventPayload {
	t.Helper()

	envelopes := make([]transport.EventPayload, 0, len(events))

	for _, evt := range events {
		sseEvt := transport.DomainEventToSSE(evt)
		require.Equal(t, "event", sseEvt.Event, "envelope SSE event name")
		require.Equal(t, sse.NewEventID(evt.ID().String()), sseEvt.ID, "envelope SSE id must carry the event id")

		var payload transport.EventPayload
		require.NoError(t, json.Unmarshal([]byte(sseEvt.Data), &payload), "envelope data must decode")
		envelopes = append(envelopes, payload)
	}

	return envelopes
}

// requireWellFormedEnvelope asserts the volatile per-dispatch fields are
// honestly populated (opaque IDs, RFC3339 timestamps) without pinning values.
func requireWellFormedEnvelope(t *testing.T, name string, p transport.EventPayload) {
	t.Helper()

	require.NotEmpty(t, p.StreamID, "%s: streamId must be populated", name)
	require.NotEmpty(t, p.EventID, "%s: eventId must be populated", name)

	_, err := time.Parse(time.RFC3339, p.OccurredAt)
	require.NoError(t, err, "%s: occurredAt must be RFC3339, got %q", name, p.OccurredAt)
}

func TestSSEEnvelopeParity_LegacyVsSystem(t *testing.T) {
	ctx := context.Background()

	// Path A — the legacy usermgmt.Service over a memory event store.
	legacyStore := memorystorage.NewMemoryStore()

	legacySvc, err := usermgmt.NewService(usermgmt.ServiceConfig{
		EventStore: legacyStore,
		AuditLog:   usermgmt.NewAuditLog(),
	})
	require.NoError(t, err)

	legacyUser, err := legacySvc.Register(ctx, usermgmt.RegisterRequest{
		ID:          id.NewUserID(),
		Email:       "parity-legacy@example.com",
		DisplayName: "Parity Legacy",
	})
	require.NoError(t, err)

	legacyStream, err := id.ParseStreamID(legacyUser.User.ID.String())
	require.NoError(t, err)

	legacyEvents, err := legacyStore.Load(
		ctx,
		id.NewStreamRef(identitymodel.AggregateTypeUser, legacyStream),
	)
	require.NoError(t, err)
	require.NotEmpty(t, legacyEvents, "legacy path must record the registration events")

	// Path B — the system path: systemadapter.DomainConfig over the memory
	// preset, dispatched through the system's command dispatcher.
	sys, err := system.New(
		ctx,
		systemadapter.DomainConfig(),
		systemadapter.RecommendedMemoryDeployment(),
	)
	require.NoError(t, err)

	require.NoError(t, sys.Start(ctx))
	t.Cleanup(func() { _ = sys.Close() })

	sysStream := id.NewStreamID()
	require.NoError(t, sys.CommandDispatcher().Dispatch(ctx, identitymodel.NewRegisterUserCmd(
		sysStream,
		"parity-system@example.com",
		"Parity System",
		[]identitymodel.Role{identitymodel.RoleViewer, identitymodel.RoleUser},
	)))

	sysEvents, err := sys.EventStore().Load(
		ctx,
		id.NewStreamRef(identitymodel.AggregateTypeUser, sysStream),
	)
	require.NoError(t, err)
	require.NotEmpty(t, sysEvents, "system path must record the registration events")

	legacyEnvelopes := sseEnvelopes(t, legacyEvents)
	sysEnvelopes := sseEnvelopes(t, sysEvents)

	// The dispatch paths must produce the same EVENT SEQUENCE and the same
	// per-position envelope metadata: this is the parity contract.
	require.Len(t, sysEnvelopes, len(legacyEnvelopes),
		"both paths must emit the same number of events for an identical command")

	for i := range legacyEnvelopes {
		legacyEnv, sysEnv := legacyEnvelopes[i], sysEnvelopes[i]

		require.Equal(t, legacyEnv.Type, sysEnv.Type, "envelope %d: event type must match", i)
		require.Equal(t, legacyEnv.StreamType, sysEnv.StreamType, "envelope %d: stream type must match", i)
		require.Equal(t, legacyEnv.Version, sysEnv.Version, "envelope %d: version must match", i)

		requireWellFormedEnvelope(t, "legacy", legacyEnv)
		requireWellFormedEnvelope(t, "system", sysEnv)
	}

	// The registration command's first event is UserRegistered on both paths
	// (the identity-model decider is shared — this pins the shared-decider
	// premise the parity assertion rests on).
	require.Equal(t, identitymodel.EventUserRegistered.String(), legacyEnvelopes[0].Type)
	require.Equal(t, identitymodel.EventUserRegistered.String(), sysEnvelopes[0].Type)
}
