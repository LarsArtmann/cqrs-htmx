package huma

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
)

// TestCommandOptionApplier_BasicCommandStillSatisfies is the drift guard for the
// shape-locked structural interfaces in enrich.go.
//
// Go cannot share unexported interfaces across modules, so this package carries
// its own copy of the one-method contract the root package declares in
// handler.go. If go-cqrs-lite ever renames ApplyOptions, enrichment would
// silently stop on BOTH transports at once (no compile error, no test failure
// elsewhere) — the request identity would vanish from every dispatched command
// and query. This test is the earliest signal that the twin drifted.
func TestCommandOptionApplier_BasicCommandStillSatisfies(t *testing.T) {
	t.Parallel()

	cmd, err := command.New("Greet", id.NewStreamID())
	if err != nil {
		t.Fatalf("command.New: %v", err)
	}

	applier, ok := any(cmd).(commandOptionApplier)
	if !ok {
		t.Fatalf("*command.BasicCommand no longer satisfies commandOptionApplier — "+
			"enrichment silently stopped (got %T)", cmd)
	}

	applier.ApplyOptions(command.WithRequestID(id.NewRequestID()))

	if cmd.Metadata().RequestID.IsZero() {
		t.Error("ApplyOptions did not reach the command metadata")
	}

	qry, err := query.New("Sum")
	if err != nil {
		t.Fatalf("query.New: %v", err)
	}

	queryApplier, ok := any(qry).(queryOptionApplier)
	if !ok {
		t.Fatalf("*query.BasicQuery no longer satisfies queryOptionApplier — "+
			"query enrichment silently stopped (got %T)", qry)
	}

	queryApplier.ApplyOptions(query.WithRequestID(id.NewRequestID()))

	if qry.Metadata().RequestID.IsZero() {
		t.Error("ApplyOptions did not reach the query metadata")
	}
}
