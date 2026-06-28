package usermgmt

import (
	"context"
	"fmt"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/stack/v3"
	cqrswatermill "github.com/larsartmann/go-cqrs-lite/watermill/v3"
)

// MaterializeProjection wraps a [stack.Materialize] as an [event.Projection],
// bridging go-cqrs-lite's declarative read-model builder with our manual
// projection dispatch ([StartProjections]).
//
// In go-cqrs-lite v3.1.0, [stack.Materialize] only exposes HandlerFunc for
// Watermill router integration — it does not implement [event.Projection]
// directly. This adapter fills that gap by:
//
//   - Declaring event types explicitly (Materialize returns nil, which our
//     slices.Contains-based dispatch treats as "accept none").
//   - Routing Handle calls through [watermill.EventToMessage] →
//     [Materialize.HandlerFunc] → upstream's handleEvent. This delegates to
//     upstream dispatch logic with zero replication — if upstream changes
//     the dispatch (adds retry, changes tombstone routing, etc.), we
//     automatically benefit.
//
// When go-cqrs-lite ships Materialize with native event.Projection conformance
// (the development branch already has Name/Handle/EventTypes), this adapter
// becomes a thin pass-through that can be removed.
//
// Usage:
//
//	store := kv.NewTypedStore[Tenant, TenantID](kv.NewMemStore())
//	mat := &stack.Materialize[Tenant, TenantID]{
//	    Store:        store,
//	    KeyFromEvent: func(evt event.Event) (TenantID, error) { ... },
//	    OnCreate:     func(ctx, evt) (*Tenant, error) { ... },
//	    OnUpdate:     func(ctx, evt, existing *Tenant) (*Tenant, error) { ... },
//	    OnTombstone:  func(ctx, evt, existing *Tenant) (*Tenant, error) { ... },
//	}
//	proj := NewMaterializeProjection(mat, "tenant-materialize", allTenantEventTypes)
//	// proj satisfies event.Projection — pass to StartProjections.
type MaterializeProjection[V any, K fmt.Stringer] struct {
	mat        *stack.Materialize[V, K]
	handler    message.NoPublishHandlerFunc
	name       string
	eventTypes []event.Type
}

// NewMaterializeProjection creates an [event.Projection] wrapper around a
// [stack.Materialize]. The caller provides:
//
//   - mat: a configured Materialize with Store, KeyFromEvent, and On* callbacks.
//   - name: projection name for diagnostics and runner registration.
//   - eventTypes: the event types this projection handles (e.g., allTenantEventTypes).
//
// The returned value satisfies [event.Projection] and can be passed to
// [StartProjections]. It also exposes [MaterializeProjection.Materialize] for
// direct query access (View, List).
func NewMaterializeProjection[V any, K fmt.Stringer](
	mat *stack.Materialize[V, K],
	name string,
	eventTypes []event.Type,
) *MaterializeProjection[V, K] {
	return &MaterializeProjection[V, K]{
		mat:        mat,
		handler:    mat.HandlerFunc(),
		name:       name,
		eventTypes: eventTypes,
	}
}

// Materialize returns the underlying [*stack.Materialize], exposing View and
// List for direct queries.
func (m *MaterializeProjection[V, K]) Materialize() *stack.Materialize[V, K] {
	return m.mat
}

// Name implements [event.Projection].
func (m *MaterializeProjection[V, K]) Name() string { return m.name }

// EventTypes implements [event.Projection]. Returns the caller-provided event
// types — NOT nil — so [slices.Contains]-based dispatch routes events here.
func (m *MaterializeProjection[V, K]) EventTypes() []event.Type { return m.eventTypes }

// Handle implements [event.Projection]. Converts the event to a Watermill
// message, sets the context, and delegates to [Materialize.HandlerFunc] —
// which calls upstream's handleEvent dispatch (OnCreate/OnUpdate/OnTombstone).
//
// This round-trip ([watermill.EventToMessage] → MessageToEvent) preserves
// tombstone metadata (verified by upstream TestMessageToEvent_TombstoneRoundtrip).
func (m *MaterializeProjection[V, K]) Handle(ctx context.Context, evt event.Event) error {
	msg := cqrswatermill.EventToMessage(evt)
	msg.SetContext(ctx)
	if err := m.handler(msg); err != nil {
		return event.Wrapf(err, event.Classify(err),
			"usermgmt.materialize_projection.handle",
			"handle event %s in materialize projection", evt.Type())
	}
	return nil
}

// Compile-time assertion.
var _ event.Projection = (*MaterializeProjection[any, dummyMaterializeStringer])(nil)

type dummyMaterializeStringer string

func (dummyMaterializeStringer) String() string { return "" }
