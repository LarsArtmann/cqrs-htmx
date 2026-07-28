package usermgmt

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

// recordingStore embeds *memory.MemoryStore, overriding Save to count writes.
// It satisfies event.Store and event.Journal via promotion while being a
// distinct type (so the journal-detection fallback path is exercised).
type recordingStore struct {
	*memory.MemoryStore
	saves atomic.Int64
}

func (s *recordingStore) Save(
	ctx context.Context,
	ref id.StreamRef,
	events []event.Event,
	expectedVersion event.Version,
) error {
	s.saves.Add(int64(len(events)))
	return s.MemoryStore.Save(ctx, ref, events, expectedVersion)
}

// storeRecorder captures whether the StoreWrapper was invoked and the store it returned.
type storeRecorder struct {
	called atomic.Bool
	store  *recordingStore
}

var errStoreWrapperSentinel = errors.New("store-wrapper-sentinel")

func TestService_StoreWrapper_AppliedAndUsed(t *testing.T) {
	var rec storeRecorder

	svc := newTestServiceWithConfig(t, ServiceConfig{
		SecurityHooks: SecurityHooks{
			StoreWrapper: func(inner event.Store) (event.Store, error) {
				rec.called.Store(true)
				memStore, ok := inner.(*memory.MemoryStore)
				if !ok {
					t.Fatalf("expected *memory.MemoryStore, got %T", inner)
				}
				rs := &recordingStore{MemoryStore: memStore}
				rec.store = rs
				return rs, nil
			},
		},
	})

	registerTestUser(t, svc, "u1", "a@b.com")

	if !rec.called.Load() {
		t.Fatal("StoreWrapper was not invoked")
	}
	if rec.store == nil {
		t.Fatal("wrapper returned nil store")
	}
	if got := rec.store.saves.Load(); got == 0 {
		t.Error("expected Save to be called through the wrapped store, got 0 events saved")
	}
	// Projections must still work through the wrapped (Journal-implementing) store.
	if _, ok := svc.readModel.FindByEmail("a@b.com"); !ok {
		t.Error("expected user in read model after wrapping the store")
	}
}

func TestService_StoreWrapper_NilResultIsNoOp(t *testing.T) {
	svc := newTestServiceWithConfig(t, ServiceConfig{
		SecurityHooks: SecurityHooks{
			StoreWrapper: func(event.Store) (event.Store, error) {
				return nil, nil //nolint:nilnil // intentionally tests nil-result handling
			},
		},
	})

	// Registration must still succeed with the default (unwrapped) store.
	registerTestUser(t, svc, "u1", "a@b.com")
}

func TestService_StoreWrapper_ErrorPropagates(t *testing.T) {
	_, err := NewService(ServiceConfig{
		SecurityHooks: SecurityHooks{
			StoreWrapper: func(event.Store) (event.Store, error) { return nil, errStoreWrapperSentinel },
		},
	})
	assertErrorIs(t, err, errStoreWrapperSentinel, "wrapper error propagation")
}

func TestService_PublishMiddleware_AppliedBeforeProjections(t *testing.T) {
	var publishCalls atomic.Int64

	svc := newTestServiceWithConfig(t, ServiceConfig{
		SecurityHooks: SecurityHooks{
			PublishMiddleware: []event.PublishMiddleware{
				func(next event.Publisher) event.Publisher {
					return event.PublisherFunc(func(ctx context.Context, events ...event.Event) error {
						publishCalls.Add(int64(len(events)))
						return next.Publish(ctx, events...)
					})
				},
			},
		},
	})

	registerTestUser(t, svc, "u1", "a@b.com")

	if got := publishCalls.Load(); got == 0 {
		t.Error("expected PublishMiddleware to run during registration, got 0 calls")
	}
	if _, ok := svc.readModel.FindByEmail("a@b.com"); !ok {
		t.Error("expected user in read model — projections must not be blocked by middleware")
	}
}

func TestService_HandlerMiddleware_AppliedBeforeProjections(t *testing.T) {
	var handleCalls atomic.Int64

	svc := newTestServiceWithConfig(t, ServiceConfig{
		SecurityHooks: SecurityHooks{
			HandlerMiddleware: []event.Middleware{
				countingHandlerMW(&handleCalls),
			},
		},
	})

	registerTestUser(t, svc, "u1", "a@b.com")

	if got := handleCalls.Load(); got == 0 {
		t.Error("expected HandlerMiddleware to run during registration, got 0 calls")
	}
	if _, ok := svc.readModel.FindByEmail("a@b.com"); !ok {
		t.Error("expected user in read model — projections must not be blocked by middleware")
	}
}

// markerPublishMW returns a PublishMiddleware that records its label in order
// before delegating, verifying that registration order == execution order
// (first registered runs first / outermost).
func markerPublishMW(label string, order *[]string, mu *sync.Mutex) event.PublishMiddleware {
	return func(next event.Publisher) event.Publisher {
		return event.PublisherFunc(func(ctx context.Context, events ...event.Event) error {
			mu.Lock()
			*order = append(*order, label)
			mu.Unlock()
			return next.Publish(ctx, events...)
		})
	}
}

// markerHandlerMW returns a Middleware that records its label in order
// before delegating.
func markerHandlerMW(label string, order *[]string, mu *sync.Mutex) event.Middleware {
	return func(next event.Handler) event.Handler {
		return func(ctx context.Context, evt event.Event) error {
			mu.Lock()
			*order = append(*order, label)
			mu.Unlock()
			return next(ctx, evt)
		}
	}
}

// countingHandlerMW returns an event.Middleware that increments counter once
// per handled event before delegating. Used to assert that HandlerMiddleware fires.
func countingHandlerMW(counter *atomic.Int64) event.Middleware {
	return func(next event.Handler) event.Handler {
		return func(ctx context.Context, evt event.Event) error {
			counter.Add(1)
			return next(ctx, evt)
		}
	}
}

func TestService_MiddlewareOrdering(t *testing.T) {
	var mu sync.Mutex
	var order []string

	svc := newTestServiceWithConfig(t, ServiceConfig{
		SecurityHooks: SecurityHooks{
			PublishMiddleware: []event.PublishMiddleware{
				markerPublishMW("first", &order, &mu),
				markerPublishMW("second", &order, &mu),
			},
			HandlerMiddleware: []event.Middleware{
				markerHandlerMW("h-first", &order, &mu),
				markerHandlerMW("h-second", &order, &mu),
			},
		},
	})

	registerTestUser(t, svc, "u1", "a@b.com")

	mu.Lock()
	defer mu.Unlock()
	// Publishing registers a single UserRegistered event; then the handler chain
	// fires for that event. Outermost-first for each axis.
	want := []string{"first", "second", "h-first", "h-second"}
	if len(order) < len(want) {
		t.Fatalf("expected at least %d markers, got %d: %v", len(want), len(order), order)
	}
	for i, w := range want {
		if order[i] != w {
			t.Errorf("marker[%d] = %q, want %q (full: %v)", i, order[i], w, order)
		}
	}
}

// TestNewEventSourcedSetup_SecurityHooks verifies that the lower-level setup
// path (NewEventSourcedSetup) supports the same security hooks as NewService,
// eliminating the split brain between the two infrastructure creation paths.
func TestNewEventSourcedSetup_SecurityHooks(t *testing.T) {
	var publishCalls atomic.Int64
	var handleCalls atomic.Int64

	setup, err := NewEventSourcedSetup(EventSourcedConfig{
		SecurityHooks: SecurityHooks{
			PublishMiddleware: []event.PublishMiddleware{
				func(next event.Publisher) event.Publisher {
					return event.PublisherFunc(func(ctx context.Context, events ...event.Event) error {
						publishCalls.Add(int64(len(events)))
						return next.Publish(ctx, events...)
					})
				},
			},
			HandlerMiddleware: []event.Middleware{
				countingHandlerMW(&handleCalls),
			},
		},
	})
	if err != nil {
		t.Fatalf("NewEventSourcedSetup: %v", err)
	}
	t.Cleanup(func() { closeBus(setup.Bus) })

	// Publish through the bus to verify middleware fires.
	aggID := id.NewStreamID()
	evt, evtErr := event.New(eventUserRegistered, aggID, "User", 1, []byte(`{}`))
	if evtErr != nil {
		t.Fatalf("event.New: %v", evtErr)
	}
	if err := setup.Bus.Publish(context.Background(), evt); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	if got := publishCalls.Load(); got == 0 {
		t.Error("expected PublishMiddleware to fire in NewEventSourcedSetup")
	}
	if got := handleCalls.Load(); got == 0 {
		t.Error("expected HandlerMiddleware to fire in NewEventSourcedSetup")
	}
}

// TestNewEventSourcedSetup_StoreWrapper verifies that StoreWrapper is applied
// in the lower-level setup path, same as NewService.
func TestNewEventSourcedSetup_StoreWrapper(t *testing.T) {
	var wrapped atomic.Bool

	setup, err := NewEventSourcedSetup(EventSourcedConfig{
		SecurityHooks: SecurityHooks{
			StoreWrapper: func(s event.Store) (event.Store, error) {
				wrapped.Store(true)
				return s, nil // pass-through for this test
			},
		},
	})
	if err != nil {
		t.Fatalf("NewEventSourcedSetup: %v", err)
	}
	t.Cleanup(func() { closeBus(setup.Bus) })

	if !wrapped.Load() {
		t.Error("expected StoreWrapper to be invoked in NewEventSourcedSetup")
	}
}

// TestService_StoreWrapper_TransformationRoundTrip verifies that a wrapper which
// transforms event payloads (the defining property of encryption/signing wrappers)
// produces correct events at the projection layer. This is the end-to-end proof
// that the StoreWrapper seam works for stateful transformations — not just
// pass-through recorders.
//
// The wrapper XORs each event's payload with a fixed key on Save (stand-in for
// encryption) and reverses the transform on Load (stand-in for decryption).
// Projections must observe the original plaintext; the persisted store holds ciphertext.
func TestService_StoreWrapper_TransformationRoundTrip(t *testing.T) {
	const xorKey byte = 0xAA // trivially reversible — stand-in for AES

	var innerStore *memory.MemoryStore // captured by the wrapper closure

	svc := newTestServiceWithConfig(t, ServiceConfig{
		SecurityHooks: SecurityHooks{
			StoreWrapper: func(inner event.Store) (event.Store, error) {
				if mem, ok := inner.(*memory.MemoryStore); ok {
					innerStore = mem
				}
				return &xorTransformStore{Store: inner, key: xorKey}, nil
			},
		},
	})

	// Register a user — this triggers Save through the wrapper (XOR transform applied).
	reg := registerTestUser(t, svc, "xor-user", "xor@test.com")

	// Projection must have seen PLAINTEXT events (the wrapper reversed the transform on Load).
	user, ok := svc.readModel.FindByEmail("xor@test.com")
	if !ok {
		t.Fatal("expected user in read model after transformation round-trip")
	}
	if user.Email != "xor@test.com" {
		t.Errorf("email mismatch after round-trip: %q", user.Email)
	}
	if user.ID != reg.User.ID {
		t.Errorf("user ID mismatch after round-trip: %q vs %q", user.ID, reg.User.ID)
	}

	// CRITICAL: verify the inner store actually holds TRANSFORMED (XORed) data,
	// not plaintext. This proves the wrapper is genuinely transforming — not just
	// passing through. We load directly from the inner store, bypassing the wrapper,
	// and confirm the payload bytes differ from plaintext.
	if innerStore == nil {
		t.Fatal("inner store was not captured — wrapper did not receive *memory.MemoryStore")
	}
	rawEvents, err := innerStore.Load(context.Background(), id.StreamRef{
		ID: mustParseAggIDSvc(t, reg.User.ID.Get().String()), Type: aggregateTypeUser,
	})
	if err != nil {
		t.Fatalf("inner store Load: %v", err)
	}
	if len(rawEvents) == 0 {
		t.Fatal("inner store has no events — wrapper did not forward Save")
	}
	for i, rawEvt := range rawEvents {
		rawPayload := rawEvt.Payload()
		if len(rawPayload) == 0 {
			t.Errorf("event[%d] payload is empty", i)
			continue
		}
		// The first byte of a JSON object is '{' (0x7B). After XOR with 0xAA,
		// it becomes 0xD1. If we see '{' here, the wrapper didn't transform.
		if rawPayload[0] == '{' {
			t.Errorf("event[%d] inner-store payload starts with '{' (plaintext) — wrapper did not transform", i)
		}
	}
}

func mustParseAggIDSvc(t *testing.T, s string) id.StreamID {
	t.Helper()
	a, err := id.ParseStreamID(s)
	if err != nil {
		t.Fatalf("ParseAggregateID(%q): %v", s, err)
	}
	return a
}

// xorTransformStore is a stand-in for encryption.NewEncryptedStore.
// It XORs event payloads with a fixed key on Save and reverses on Load.
// Satisfies event.Store and (when the inner store does) event.Journal.
type xorTransformStore struct {
	event.Store
	key byte
}

func (s *xorTransformStore) Save(
	ctx context.Context,
	ref id.StreamRef,
	events []event.Event,
	expectedVersion event.Version,
) error {
	transformed := make([]event.Event, len(events))
	for i, evt := range events {
		payload := evt.Payload()
		xored := make([]byte, len(payload))
		for j, b := range payload {
			xored[j] = b ^ s.key
		}
		newEvt, err := event.New(evt.Type(), evt.StreamID(), evt.AggregateType(), evt.Version(), xored)
		if err != nil {
			return fmt.Errorf("xor transform: rebuild event: %w", err)
		}
		transformed[i] = newEvt
	}
	return s.Store.Save(ctx, ref, transformed, expectedVersion)
}

func (s *xorTransformStore) Load(
	ctx context.Context,
	ref id.StreamRef,
) ([]event.Event, error) {
	events, err := s.Store.Load(ctx, ref)
	if err != nil {
		return nil, err
	}
	for i, evt := range events {
		payload := evt.Payload()
		unxored := make([]byte, len(payload))
		for j, b := range payload {
			unxored[j] = b ^ s.key
		}
		newEvt, err := event.New(evt.Type(), evt.StreamID(), evt.AggregateType(), evt.Version(), unxored)
		if err != nil {
			return nil, fmt.Errorf("xor transform: rebuild event on load: %w", err)
		}
		events[i] = newEvt
	}
	return events, nil
}
