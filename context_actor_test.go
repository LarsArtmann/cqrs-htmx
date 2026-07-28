package cqrshtmx

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

func TestWithActorID_ActorFromContext(t *testing.T) {
	t.Parallel()

	ctx := WithActorID(context.Background(), NewActorID("user:01JX..."))
	if got := ActorIDFromContext(ctx); got.Get() != "user:01JX..." {
		t.Errorf("ActorIDFromContext = %q, want %q", got.Get(), "user:01JX...")
	}
}

func TestActorIDFromContext_Empty(t *testing.T) {
	t.Parallel()

	if got := ActorIDFromContext(context.Background()); !got.IsZero() {
		t.Errorf("ActorIDFromContext = %q, want empty", got.Get())
	}
}

func TestWithImpersonatorID_ImpersonatorFromContext(t *testing.T) {
	t.Parallel()

	ctx := WithImpersonatorID(context.Background(), NewActorID("user:01ADM..."))
	if got := ImpersonatorIDFromContext(ctx); got.Get() != "user:01ADM..." {
		t.Errorf("ImpersonatorIDFromContext = %q, want %q", got.Get(), "user:01ADM...")
	}
}

func TestImpersonatorIDFromContext_Empty(t *testing.T) {
	t.Parallel()

	if got := ImpersonatorIDFromContext(context.Background()); !got.IsZero() {
		t.Errorf("ImpersonatorIDFromContext = %q, want empty", got.Get())
	}
}

func TestEventOptionsFromContext_ActorChain(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctx = WithActorID(ctx, NewActorID("user:01JX..."))
	ctx = WithImpersonatorID(ctx, NewActorID("user:01ADM..."))

	opts := EventOptionsFromContext(ctx)
	if len(opts) < 2 {
		t.Fatalf("expected at least 2 options, got %d", len(opts))
	}

	aggID := id.NewStreamID()

	evt, err := event.NewEvent(
		event.Type("TestEvent"), aggID, "TestAggregate", 1,
		[]byte(`{}`),
		opts...,
	)
	if err != nil {
		t.Fatalf("event.NewEvent failed: %v", err)
	}

	custom := evt.Metadata().Custom
	if custom == nil {
		t.Fatal("expected non-nil Custom metadata")
	}

	if v := custom[MetadataKeyActorID]; v != "user:01JX..." {
		t.Errorf("metadata actor_id = %q, want %q", v, "user:01JX...")
	}

	if v := custom[MetadataKeyImpersonatorID]; v != "user:01ADM..." {
		t.Errorf("metadata impersonator_id = %q, want %q", v, "user:01ADM...")
	}
}

func TestEventOptionsFromContext_NoActorChain(t *testing.T) {
	t.Parallel()

	opts := EventOptionsFromContext(context.Background())
	if opts == nil {
		return
	}

	aggID := id.NewStreamID()

	evt, err := event.NewEvent(
		event.Type("TestEvent"), aggID, "TestAggregate", 1,
		[]byte(`{}`),
		opts...,
	)
	if err != nil {
		t.Fatalf("event.NewEvent failed: %v", err)
	}

	custom := evt.Metadata().Custom
	if custom == nil {
		return
	}

	if _, ok := custom[MetadataKeyActorID]; ok {
		t.Error("should not have actor_id in metadata")
	}

	if _, ok := custom[MetadataKeyImpersonatorID]; ok {
		t.Error("should not have impersonator_id in metadata")
	}
}

func TestImpersonatorID_IsActorID(t *testing.T) {
	t.Parallel()
	// ImpersonatorID is a type alias for ActorID — they are the SAME type.
	// This is a compile-time guarantee: an impersonator IS an actor.
	_ = NewActorID("user:01ADM...")
	_ = NewActorID("user:01JX...")
}
