package usermgmt

import (
	"context"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/middleware/v4"
)

// commandOptionApplier is the structural capability interface for commands
// that accept metadata options after construction. Every identity-model
// command embeds *command.BasicCommand, whose ApplyOptions method is promoted
// to the wrapper, so all 20 domain commands satisfy this interface without
// knowing about it. Hand-rolled command implementations that do not expose
// ApplyOptions pass through unchanged.
type commandOptionApplier interface {
	ApplyOptions(...command.Option)
}

// commandAuditMiddleware is the middleware chain every usermgmt dispatcher
// runs: identity bridge, command-metadata enrichment, actor context, and
// command causation. Ordering matters and is pinned by
// TestAuditMiddleware_OrderPinned: auditContextEnrichment must run before
// middleware.CommandActorContext so the actor lands on the command metadata
// before it is lifted into the handler context; consumer middleware (added
// after these via Use) runs innermost and sees fully enriched commands.
func commandAuditMiddleware() []command.Middleware {
	return []command.Middleware{
		auditContextEnrichment(),
		middleware.CommandActorContext(),
		commandCausalityContext(),
	}
}

// auditContextEnrichment returns a command middleware that bridges
// usermgmt-authenticated identity into cqrshtmx's shared context chain (when
// not already set) and applies the resulting request-scoped metadata onto the
// dispatched command. Paired with the repository's
// CompositeEnricher(ActorEnricher, CommandCausalityEnricher), this makes
// every event emitted by usermgmt record who issued the command and which
// command caused the event — without any consumer wiring.
func auditContextEnrichment() command.Middleware {
	return func(next command.Handler) command.Handler {
		return func(ctx context.Context, cmd command.Command) error {
			ctx = bridgeSessionIdentity(ctx)
			if applier, ok := cmd.(commandOptionApplier); ok {
				applier.ApplyOptions(cqrshtmx.CommandOptionsFromContext(ctx)...)
			}
			return next(ctx, cmd)
		}
	}
}

// bridgeSessionIdentity lifts the authenticated user from usermgmt's session
// context into cqrshtmx's shared actor context key. The session middleware
// stores the *User under usermgmt's own key; without this bridge, dispatched
// commands carry no actor even on authenticated requests because
// cqrshtmx.CommandOptionsFromContext reads only cqrshtmx's keys. Values
// already present win: consumer-set identity (via
// cqrshtmx.ContextEnrichmentMiddleware or explicit WithActorID) is never
// overwritten. Impersonator bridging remains consumer-side (see
// NewSessionMiddleware docs).
func bridgeSessionIdentity(ctx context.Context) context.Context {
	if !cqrshtmx.ActorIDFromContext(ctx).IsZero() {
		return ctx
	}
	user, ok := UserFromContext(ctx)
	if !ok || user == nil {
		return ctx
	}
	return cqrshtmx.WithActorID(ctx, identitymodel.ActorIDFromUser(user.ID))
}

// commandCausalityContext returns a command middleware that stores the
// command's type and ID in the handler context (event.WithCommandCausality)
// so the repository's CommandCausalityEnricher can stamp every emitted event
// with the command that caused it. This is the causation counterpart to the
// actor context: actor answers "who", causation answers "which command".
func commandCausalityContext() command.Middleware {
	return func(next command.Handler) command.Handler {
		return func(ctx context.Context, cmd command.Command) error {
			if !cmd.ID().IsZero() {
				ctx = event.WithCommandCausality(ctx, string(cmd.Type()), cmd.ID())
			}
			return next(ctx, cmd)
		}
	}
}
