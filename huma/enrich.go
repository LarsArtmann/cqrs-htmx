package huma

import (
	"context"

	"github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
)

// commandOptionApplier is the structural capability interface for commands that
// accept metadata options after construction.
//
// SHAPE-LOCKED TWIN: the root package declares the same unexported interface
// (handler.go) and usermgmt declares a third copy in audit_context.go — Go
// cannot share unexported interfaces across modules. *command.BasicCommand
// satisfies it directly and every wrapper embedding it satisfies it through
// method promotion, which is why the assertion below is wide enough to cover
// the identity-model command pattern without naming it.
//
// TestCommandOptionApplier_BasicCommandStillSatisfies pins the shape: if
// go-cqrs-lite ever renames ApplyOptions, enrichment silently stops on both
// transports at once and this test is the earliest signal.
type commandOptionApplier interface {
	ApplyOptions(opts ...command.Option)
}

// queryOptionApplier is the query-side mirror of commandOptionApplier.
type queryOptionApplier interface {
	ApplyOptions(opts ...query.Option)
}

// enrichCommand injects request-scoped metadata (actor, user, correlation,
// request ID) from ctx into an already-built command before dispatch.
//
// Commands that do not expose ApplyOptions pass through unchanged: the decoder
// (here, the consumer's build function) decides what metadata the command
// carries.
func enrichCommand(ctx context.Context, cmd command.Command) {
	if applier, ok := cmd.(commandOptionApplier); ok {
		applier.ApplyOptions(cqrshtmx.CommandOptionsFromContext(ctx)...)
	}
}

// enrichQuery is the query-side mirror of enrichCommand.
func enrichQuery(ctx context.Context, qry query.Query) {
	if applier, ok := qry.(queryOptionApplier); ok {
		applier.ApplyOptions(cqrshtmx.QueryOptionsFromContext(ctx)...)
	}
}
