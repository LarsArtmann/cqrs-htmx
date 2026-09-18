package cqrshtmx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
)

// wrappedCmd embeds *command.BasicCommand the same way every identity-model
// command does: payload fields live on the wrapper, the CQRS plumbing is
// promoted from the embedded basic command.
type wrappedCmd struct {
	*command.BasicCommand

	payload string
}

// handRolledCmd implements command.Command without any BasicCommand — the
// enrichment skip path.
type handRolledCmd struct{}

func (handRolledCmd) Type() command.Type    { return "HandRolled" }
func (handRolledCmd) StreamID() id.StreamID { return id.NewStreamID() }
func (handRolledCmd) ID() id.CommandID      { return id.NewCommandID() }

// wrappedQuery embeds *query.BasicQuery like wrappedCmd does for commands.
type wrappedQuery struct {
	*query.BasicQuery

	filter string
}

func newWrappedCmd(t *testing.T) *wrappedCmd {
	t.Helper()

	basic, err := command.New("Wrapped", id.NewStreamID())
	if err != nil {
		t.Fatalf("command.New: %v", err)
	}

	return &wrappedCmd{BasicCommand: basic, payload: "x"}
}

// TestEnrichCommandFromContext_EmbeddedWrapperIsEnriched is the regression
// test for the structural fix: a wrapper embedding *command.BasicCommand
// (the identity-model pattern) MUST receive context-derived metadata. The
// pre-fix concrete-type assertion (*command.BasicCommand) silently skipped
// every such wrapper.
func TestEnrichCommandFromContext_EmbeddedWrapperIsEnriched(t *testing.T) {
	t.Parallel()

	ctx := WithActorID(t.Context(), actorForTest(t, "user:01JXWRAPPERTEST000000"))
	ctx = WithCorrelationID(ctx, NewCorrelationID())

	cmd := newWrappedCmd(t)
	enrichCommandFromContext(ctx, cmd)

	meta := cmd.Metadata()
	if want := actorForTest(t, "user:01JXWRAPPERTEST000000"); meta.ActorID != want {
		t.Errorf("wrapper command actor = %q, want %q", meta.ActorID, want)
	}

	if meta.CorrelationID.IsZero() {
		t.Error("wrapper command correlation ID = zero, want the context correlation ID")
	}
}

// TestEnrichCommandFromContext_PlainBasicCommandStillEnriched proves no
// regression for the previously-working case.
func TestEnrichCommandFromContext_PlainBasicCommandStillEnriched(t *testing.T) {
	t.Parallel()

	basic, err := command.New("Plain", id.NewStreamID())
	if err != nil {
		t.Fatalf("command.New: %v", err)
	}

	ctx := WithActorID(t.Context(), actorForTest(t, "user:01JXPLAINTEST0000000000"))
	enrichCommandFromContext(ctx, basic)

	if want := actorForTest(t, "user:01JXPLAINTEST0000000000"); basic.Metadata().ActorID != want {
		t.Errorf("plain command actor = %q, want %q", basic.Metadata().ActorID, want)
	}
}

// TestEnrichCommandFromContext_HandRolledSkippedWithoutPanic proves commands
// without an ApplyOptions method still dispatch untouched (and the
// enrichment path never panics on them).
func TestEnrichCommandFromContext_HandRolledSkippedWithoutPanic(t *testing.T) {
	t.Parallel()

	ctx := WithActorID(t.Context(), actorForTest(t, "user:01JXHANDROLLEDTEST00000"))
	enrichCommandFromContext(ctx, handRolledCmd{}) // must not panic
}

// TestEnrichQueryFromContext_EmbeddedWrapperIsEnriched is the query-side
// mirror of the command regression test.
func TestEnrichQueryFromContext_EmbeddedWrapperIsEnriched(t *testing.T) {
	t.Parallel()

	basic, err := query.New("WrappedQuery")
	if err != nil {
		t.Fatalf("query.New: %v", err)
	}

	qry := &wrappedQuery{BasicQuery: basic, filter: "all"}

	ctx := WithActorID(t.Context(), actorForTest(t, "user:01JXQWRAPPERTEST00000000"))
	enrichQueryFromContext(ctx, qry)

	if want := actorForTest(t, "user:01JXQWRAPPERTEST00000000"); qry.Metadata().ActorID != want {
		t.Errorf("wrapper query actor = %q, want %q", qry.Metadata().ActorID, want)
	}
}

// TestCommandPipeline_EnrichesEmbeddedWrapper proves the fix end-to-end
// through the public pipeline: a command endpoint dispatches a wrapper
// command, and context set by an outer HTTP middleware reaches the handler
// as command metadata.
func TestCommandPipeline_EnrichesEmbeddedWrapper(t *testing.T) {
	t.Parallel()

	disp := command.NewDispatcher()

	var gotActor id.ActorID

	if err := disp.Register("Wrapped", func(_ context.Context, cmd command.Command) error {
		if wrapped, ok := cmd.(*wrappedCmd); ok {
			gotActor = wrapped.Metadata().ActorID
		}

		return nil
	}); err != nil {
		t.Fatalf("Register: %v", err)
	}

	app := MustNew(Config{Commands: disp})

	mux := http.NewServeMux()
	mux.Handle("POST /wrapped", app.Command("Wrapped",
		DecodeJSON(func(_ struct{}) (command.Command, error) {
			return newWrappedCmd(t), nil
		}),
	))

	actor := actorForTest(t, "user:01JXE2EWRAPPERTEST000000")
	outer := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r = r.WithContext(WithActorID(r.Context(), actor))
			next.ServeHTTP(w, r)
		})
	}

	req := httptest.NewRequest(http.MethodPost, "/wrapped", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	outer(mux).ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("POST /wrapped status = %d, want 204", w.Code)
	}

	if gotActor != actor {
		t.Errorf("handler saw actor %q, want %q (wrapper command must be enriched)", gotActor, actor)
	}
}
