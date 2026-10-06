package dashboardui

import (
	"context"
	"net/http"

	"github.com/larsartmann/httputil"
)

// Actor identifies who is performing dashboard write operations, for audit
// attribution. The zero value means "authorized but identity unknown".
type Actor struct {
	// ID is a stable identifier for the actor (e.g. a user ID). May be
	// empty when only a display name is known.
	ID string

	// Name is a human-readable display name (e.g. an email address). May
	// be empty when only an identifier is known.
	Name string
}

// known reports whether the actor carries any identity at all.
func (a Actor) known() bool {
	return a.ID != "" || a.Name != ""
}

type actorContextKey struct{}

// WithActor returns a context that carries the actor for audit attribution.
// Consumer middleware can call this directly — an actor injected that way is
// recorded by dashboard audit log entries even without any dashboard
// Authorizer configured.
func WithActor(ctx context.Context, actor Actor) context.Context {
	return context.WithValue(ctx, actorContextKey{}, actor)
}

// ActorFromContext reports the actor attached to the request context, if any.
func ActorFromContext(ctx context.Context) (Actor, bool) {
	actor, ok := ctx.Value(actorContextKey{}).(Actor)

	return actor, ok
}

// actorAuthorizer resolves the configured authorization into one callable
// returning the actor on success: the actor-aware ActorAuthorizer when set
// (it takes precedence), the legacy Authorizer wrapped to report an unknown
// actor, or nil when neither is configured (allow all).
func (d *Dashboard) actorAuthorizer() func(*http.Request) (*Actor, error) {
	switch {
	case d.config.ActorAuthorizer != nil:
		return func(r *http.Request) (*Actor, error) {
			actor, err := d.config.ActorAuthorizer(r)
			if err != nil {
				return nil, err
			}

			return &actor, nil
		}

	case d.config.Authorizer != nil:
		return func(r *http.Request) (*Actor, error) {
			if err := d.config.Authorizer(r); err != nil {
				return nil, err
			}

			return nil, nil
		}

	default:
		return nil
	}
}

// auditAttrs returns the attribution attributes every dashboardui.audit log
// entry carries: the acting identity (from the request context — set by
// ActorAuthorizer or consumer middleware) and the request ID (when the
// consumer runs httputil's request-ID middleware).
func (d *Dashboard) auditAttrs(r *http.Request) []any {
	attrs := []any{"actor", "anonymous"}

	if actor, ok := ActorFromContext(r.Context()); ok && actor.known() {
		attrs = []any{"actor_id", actor.ID, "actor_name", actor.Name}
	}

	if requestID := httputil.RequestIDFromContext(r.Context()); requestID != "" {
		attrs = append(attrs, "request_id", requestID)
	}

	return attrs
}
