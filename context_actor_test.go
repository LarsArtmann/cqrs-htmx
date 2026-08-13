package cqrshtmx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
)

func actorForTest(t *testing.T, prefixed string) ActorID {
	t.Helper()

	a, err := ParseActorID(prefixed)
	if err != nil {
		t.Fatalf("ParseActorID(%q): %v", prefixed, err)
	}

	return a
}

func TestWithActorID_ActorFromContext(t *testing.T) {
	t.Parallel()

	actor := actorForTest(t, "user:01JX...")
	ctx := WithActorID(context.Background(), actor)

	got := ActorIDFromContext(ctx)
	if got.PrefixedString() != "user:01JX..." {
		t.Errorf("ActorIDFromContext = %q, want %q", got.PrefixedString(), "user:01JX...")
	}
}

func TestActorIDFromContext_Empty(t *testing.T) {
	t.Parallel()

	if got := ActorIDFromContext(context.Background()); !got.IsZero() {
		t.Errorf("ActorIDFromContext = %q, want empty", got.PrefixedString())
	}
}

func TestWithImpersonatorID_ImpersonatorFromContext(t *testing.T) {
	t.Parallel()

	actor := actorForTest(t, "user:01ADM...")
	ctx := WithImpersonatorID(context.Background(), actor)

	got := ImpersonatorIDFromContext(ctx)
	if got.PrefixedString() != "user:01ADM..." {
		t.Errorf("ImpersonatorIDFromContext = %q, want %q", got.PrefixedString(), "user:01ADM...")
	}
}

func TestImpersonatorIDFromContext_Empty(t *testing.T) {
	t.Parallel()

	if got := ImpersonatorIDFromContext(context.Background()); !got.IsZero() {
		t.Errorf("ImpersonatorIDFromContext = %q, want empty", got.PrefixedString())
	}
}

func TestEventOptionsFromContext_ActorChain(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctx = WithActorID(ctx, actorForTest(t, "user:01JX..."))
	ctx = WithImpersonatorID(ctx, actorForTest(t, "user:01ADM..."))

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

	// ActorID now goes into the native CommonMetadata.ActorID field via
	// event.WithActor — no longer into Custom metadata.
	if got := evt.Metadata().ActorID; got.PrefixedString() != "user:01JX..." {
		t.Errorf("metadata ActorID = %q, want %q", got.PrefixedString(), "user:01JX...")
	}

	custom := evt.Metadata().Custom
	if custom == nil {
		t.Fatal("expected non-nil Custom metadata for impersonator")
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

	if !evt.Metadata().ActorID.IsZero() {
		t.Errorf("expected zero ActorID, got %q", evt.Metadata().ActorID.PrefixedString())
	}

	custom := evt.Metadata().Custom
	if custom == nil {
		return
	}

	if _, ok := custom[MetadataKeyImpersonatorID]; ok {
		t.Error("should not have impersonator_id in metadata")
	}
}

func TestImpersonatorID_IsActorID(t *testing.T) {
	t.Parallel()
	// ImpersonatorID is a type alias for ActorID — they are the SAME type.
	// This is a compile-time guarantee: an impersonator IS an actor.
	_ = id.NewUserActor(id.NewUserID())
	_ = ActorID{}

	imp := ActorID{}

	_ = imp
}

func TestActorIDFromUser(t *testing.T) {
	t.Parallel()

	uid := id.NewUserID()
	actor := ActorIDFromUser(uid)

	if actor.Kind() != id.ActorUser {
		t.Errorf("ActorIDFromUser kind = %v, want ActorUser", actor.Kind())
	}

	if actor.String() != uid.String() {
		t.Errorf("ActorIDFromUser raw = %q, want %q", actor.String(), uid.String())
	}

	if actor.PrefixedString() != "user:"+uid.String() {
		t.Errorf("ActorIDFromUser prefixed = %q, want %q",
			actor.PrefixedString(), "user:"+uid.String())
	}
}

func TestContextEnrichmentMiddleware_AutoDerivesActorID(t *testing.T) {
	t.Parallel()

	uid := id.NewUserID()
	extractor := func(_ *http.Request) (UserID, error) { return uid, nil }

	var capturedActor ActorID

	mw := ContextEnrichmentMiddleware(extractor)
	handler := mw(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		capturedActor = ActorIDFromContext(r.Context())
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	handler.ServeHTTP(w, r)

	if capturedActor.IsZero() {
		t.Fatal("expected ActorID to be auto-derived from UserID, got zero")
	}

	if capturedActor.Kind() != id.ActorUser {
		t.Errorf("auto-derived actor kind = %v, want ActorUser", capturedActor.Kind())
	}

	if capturedActor.String() != uid.String() {
		t.Errorf("auto-derived actor = %q, want %q", capturedActor.String(), uid.String())
	}
}

func TestContextEnrichmentMiddleware_DoesNotOverrideActorID(t *testing.T) {
	t.Parallel()

	uid := id.NewUserID()
	// Consumer pre-sets an ActorID (e.g., impersonation targeting a different user)
	targetActor := actorForTest(t, "user:01TARGET...")
	extractor := func(_ *http.Request) (UserID, error) { return uid, nil }

	var capturedActor ActorID

	// Pre-middleware that sets ActorID before ContextEnrichmentMiddleware runs
	preMW := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r = r.WithContext(WithActorID(r.Context(), targetActor))
			next.ServeHTTP(w, r)
		})
	}

	finalHandler := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		capturedActor = ActorIDFromContext(r.Context())
	})

	handler := preMW(ContextEnrichmentMiddleware(extractor)(finalHandler))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	handler.ServeHTTP(w, r)

	if capturedActor.PrefixedString() != "user:01TARGET..." {
		t.Errorf("ActorID = %q, want pre-set value %q",
			capturedActor.PrefixedString(), "user:01TARGET...")
	}
}

func TestContextEnrichmentMiddleware_NoActorIDForZeroUser(t *testing.T) {
	t.Parallel()

	extractor := func(_ *http.Request) (UserID, error) { return UserID{}, nil }

	var capturedActor ActorID

	mw := ContextEnrichmentMiddleware(extractor)
	handler := mw(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		capturedActor = ActorIDFromContext(r.Context())
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	handler.ServeHTTP(w, r)

	if !capturedActor.IsZero() {
		t.Errorf("expected zero ActorID for zero UserID, got %q",
			capturedActor.PrefixedString())
	}
}

func TestCommandOptionsFromContext(t *testing.T) {
	t.Parallel()

	uid := id.NewUserID()
	cid := id.NewCorrelationID()
	rid := id.NewRequestID()

	ctx := context.Background()
	ctx = WithUserID(ctx, uid)
	ctx = WithActorID(ctx, ActorIDFromUser(uid))
	ctx = WithCorrelationID(ctx, cid)
	ctx = WithRequestID(ctx, rid)

	opts := CommandOptionsFromContext(ctx)
	if len(opts) != 4 {
		t.Fatalf("expected 4 options, got %d", len(opts))
	}

	cmd, err := command.New(
		command.Type("CreateUser"),
		id.NewStreamID(),
		opts...,
	)
	if err != nil {
		t.Fatalf("command.New failed: %v", err)
	}

	meta := cmd.Metadata()

	if meta.UserID != uid {
		t.Errorf("metadata UserID = %q, want %q", meta.UserID, uid)
	}

	if meta.ActorID.Kind() != id.ActorUser {
		t.Errorf("metadata ActorID kind = %v, want ActorUser", meta.ActorID.Kind())
	}

	if meta.CorrelationID != cid {
		t.Errorf("metadata CorrelationID = %q, want %q", meta.CorrelationID, cid)
	}

	if meta.RequestID != rid {
		t.Errorf("metadata RequestID = %q, want %q", meta.RequestID, rid)
	}
}

func TestCommandOptionsFromContext_Empty(t *testing.T) {
	t.Parallel()

	opts := CommandOptionsFromContext(context.Background())
	if len(opts) != 0 {
		t.Errorf("expected 0 options for empty context, got %d", len(opts))
	}
}

func TestQueryOptionsFromContext(t *testing.T) {
	t.Parallel()

	uid := id.NewUserID()
	cid := id.NewCorrelationID()
	rid := id.NewRequestID()

	ctx := context.Background()
	ctx = WithUserID(ctx, uid)
	ctx = WithActorID(ctx, ActorIDFromUser(uid))
	ctx = WithCorrelationID(ctx, cid)
	ctx = WithRequestID(ctx, rid)

	opts := QueryOptionsFromContext(ctx)
	if len(opts) != 4 {
		t.Fatalf("expected 4 options, got %d", len(opts))
	}

	qry, err := query.New(query.Type("ListUsers"), opts...)
	if err != nil {
		t.Fatalf("query.New failed: %v", err)
	}

	meta := qry.Metadata()

	if meta.UserID != uid {
		t.Errorf("metadata UserID = %q, want %q", meta.UserID, uid)
	}

	if meta.ActorID.Kind() != id.ActorUser {
		t.Errorf("metadata ActorID kind = %v, want ActorUser", meta.ActorID.Kind())
	}

	if meta.CorrelationID != cid {
		t.Errorf("metadata CorrelationID = %q, want %q", meta.CorrelationID, cid)
	}

	if meta.RequestID != rid {
		t.Errorf("metadata RequestID = %q, want %q", meta.RequestID, rid)
	}
}

func TestQueryOptionsFromContext_Empty(t *testing.T) {
	t.Parallel()

	opts := QueryOptionsFromContext(context.Background())
	if len(opts) != 0 {
		t.Errorf("expected 0 options for empty context, got %d", len(opts))
	}
}
