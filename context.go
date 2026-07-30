package cqrshtmx

import (
	"context"
	"fmt"

	brandid "github.com/larsartmann/go-branded-id"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// UserID is a strongly-typed identifier for users, preventing accidental mixing
// with other ID types at compile time.
type UserID = id.UserID

// NewUserID generates a new random UserID backed by a ULID.
//
// Naming note: this module's NewUserID() GENERATES a fresh ULID, whereas
// usermgmt.NewUserID(string) PARSES or DERIVES a UserID from a string — same
// name, opposite semantics, both visible when a consumer imports both packages.
// Prefer [GenerateUserID] here (unambiguous: it generates) and
// [usermgmt.ParseUserID] / [usermgmt.SyntheticUserID] on the usermgmt side.
func NewUserID() UserID {
	return id.NewUserID()
}

// GenerateUserID generates a new random UserID backed by a ULID. It is the
// unambiguous name for [NewUserID] (which has a same-named counterpart in the
// usermgmt module with parse/derive semantics). Prefer GenerateUserID in new code.
func GenerateUserID() UserID {
	return id.NewUserID()
}

// parseID is a generic helper that wraps an ID parse function with contextual error messages.
func parseID[T any](s string, parse func(string) (T, error), label string) (T, error) {
	v, err := parse(s)
	if err != nil {
		return v, errorfamily.Wrapf(err, event.Rejection, "cqrshtmx.context.id_parse_failed", "parse %s", label)
	}

	return v, nil
}

// ParseUserID converts a string to a UserID.
// Returns an error if the input is not a valid ULID.
func ParseUserID(s string) (UserID, error) {
	return parseID(s, id.ParseUserID, "user id")
}

// MustParseUserID converts a string to a UserID, panicking on error.
func MustParseUserID(s string) UserID {
	v, err := ParseUserID(s)
	if err != nil {
		panic( //cqrs-lint:ignore(C009) Must* function: panics on invalid input by Go convention
			fmt.Sprintf("MustParseUserID: %v", err),
		)
	}

	return v
}

// CorrelationID is a strongly-typed correlation identifier, preventing accidental
// mixing with other ID types at compile time.
type CorrelationID = id.CorrelationID

// NewCorrelationID generates a new random CorrelationID backed by a ULID.
func NewCorrelationID() CorrelationID {
	return id.NewCorrelationID()
}

// ParseCorrelationID converts a string to a CorrelationID.
// Returns an error if the input is not a valid ULID.
func ParseCorrelationID(s string) (CorrelationID, error) {
	return parseID(s, id.ParseCorrelationID, "correlation id")
}

// MustParseCorrelationID converts a string to a CorrelationID, panicking on error.
func MustParseCorrelationID(s string) CorrelationID {
	v, err := ParseCorrelationID(s)
	if err != nil {
		panic( //cqrs-lint:ignore(C009) Must* function: panics on invalid input by Go convention
			fmt.Sprintf("MustParseCorrelationID: %v", err),
		)
	}

	return v
}

type contextKey[T any] struct{ name string }

func (k contextKey[T]) WithValue(ctx context.Context, v T) context.Context {
	return context.WithValue(ctx, k, v)
}

func (k contextKey[T]) FromContext(ctx context.Context) (T, bool) {
	v, ok := ctx.Value(k).(T)

	return v, ok
}

//nolint:gochecknoglobals // context key singletons; one instance per key is standard.
var (
	userIDKeyInstance         = contextKey[UserID]{name: "user_id"}
	correlationIDKeyInstance  = contextKey[CorrelationID]{name: "correlation_id"}
	requestIDKeyInstance      = contextKey[RequestID]{name: "request_id"}
	actorIDKeyInstance        = contextKey[ActorID]{name: "actor_id"}
	impersonatorIDKeyInstance = contextKey[ImpersonatorID]{name: "impersonator_id"}
)

// MetadataKeyActorID is the event metadata custom-data key for the effective
// identity — who the request acts AS. Written by EventOptionsFromContext.
const MetadataKeyActorID = "actor_id"

// MetadataKeyImpersonatorID is the event metadata custom-data key for the real
// authenticated identity (the admin). Written by EventOptionsFromContext when
// impersonation is active.
const MetadataKeyImpersonatorID = "impersonator_id"

// RequestID is a strongly-typed per-request identifier, preventing accidental
// mixing with other ID types at compile time.
type RequestID = id.RequestID

// NewRequestID generates a new random RequestID backed by a ULID.
func NewRequestID() RequestID {
	return id.NewRequestID()
}

// ParseRequestID converts a string to a RequestID.
// Returns an error if the input is not a valid ULID.
func ParseRequestID(s string) (RequestID, error) {
	return parseID(s, id.ParseRequestID, "request id")
}

// MustParseRequestID converts a string to a RequestID, panicking on error.
func MustParseRequestID(s string) RequestID {
	v, err := ParseRequestID(s)
	if err != nil {
		panic( //cqrs-lint:ignore(C009) Must* function: panics on invalid input by Go convention
			fmt.Sprintf("MustParseRequestID: %v", err),
		)
	}

	return v
}

// WithRequestID stores a strongly-typed request ID in the context.
func WithRequestID(ctx context.Context, requestID RequestID) context.Context {
	return requestIDKeyInstance.WithValue(ctx, requestID)
}

// RequestIDFromContext retrieves the request ID stored by WithRequestID.
// Returns the zero value of RequestID if no request ID is present.
func RequestIDFromContext(ctx context.Context) RequestID {
	v, _ := requestIDKeyInstance.FromContext(ctx)

	return v
}

// WithCorrelationID stores a strongly-typed correlation ID in the context.
func WithCorrelationID(ctx context.Context, correlationID CorrelationID) context.Context {
	return correlationIDKeyInstance.WithValue(ctx, correlationID)
}

// CorrelationIDFromContext retrieves the correlation ID stored by WithCorrelationID.
// Returns the zero value of CorrelationID if no correlation ID is present.
func CorrelationIDFromContext(ctx context.Context) CorrelationID {
	v, _ := correlationIDKeyInstance.FromContext(ctx)

	return v
}

// WithUserID stores a strongly-typed user ID in the context for downstream CQRS handlers.
func WithUserID(ctx context.Context, userID UserID) context.Context {
	return userIDKeyInstance.WithValue(ctx, userID)
}

// UserIDFromContext retrieves the user ID stored by WithUserID.
// Returns the zero value of UserID if no user ID is present.
func UserIDFromContext(ctx context.Context) UserID {
	v, _ := userIDKeyInstance.FromContext(ctx)

	return v
}

// actorBrand is the phantom brand type for ActorID, giving compile-time
// protection against mixing ActorID with other string-based IDs.
type actorBrand struct{}

func (actorBrand) Name() string { return "Actor" }

// ActorID is a branded identifier for the effective actor (user or bot) in
// the request context. The string form is a prefixed ID like "user:01JX...".
// Use NewActorID to construct from a string; use .Get() to extract the raw
// string for storage or serialization.
type ActorID = brandid.ID[actorBrand, string]

// NewActorID constructs an ActorID from a prefixed string (e.g. "user:01JX...").
func NewActorID(s string) ActorID { return brandid.NewID[actorBrand](s) }

// ImpersonatorID is the same type as ActorID — an impersonator IS an actor.
// The distinction is contextual (stored under impersonatorIDKey{}, not
// actorIDKey{}), not a different identity type.
type ImpersonatorID = ActorID

// WithActorID stores the effective actor ID in the context.
// This is who the request ACTS AS — the target user in impersonation,
// or the authenticated user in direct login.
func WithActorID(ctx context.Context, actorID ActorID) context.Context {
	return actorIDKeyInstance.WithValue(ctx, actorID)
}

// ActorIDFromContext retrieves the effective actor ID stored by WithActorID.
// Returns the zero value (empty string) if no actor ID is present.
func ActorIDFromContext(ctx context.Context) ActorID {
	v, _ := actorIDKeyInstance.FromContext(ctx)

	return v
}

// WithImpersonatorID stores the real actor (impersonator) in the context.
// When set, the request is an impersonation: ActorID is the target,
// ImpersonatorID is the admin acting on their behalf.
func WithImpersonatorID(ctx context.Context, impersonatorID ImpersonatorID) context.Context {
	return impersonatorIDKeyInstance.WithValue(ctx, impersonatorID)
}

// ImpersonatorIDFromContext retrieves the impersonator ID stored by WithImpersonatorID.
// Returns the zero value (empty string) if not an impersonation request.
func ImpersonatorIDFromContext(ctx context.Context) ImpersonatorID {
	v, _ := impersonatorIDKeyInstance.FromContext(ctx)

	return v
}

// EventOptionsFromContext builds event.Options from request context,
// propagating user identity, correlation ID, request ID, and context deadline
// into event metadata for auditing, tracing, and distributed request correlation.
//
// Returns nil options if none of user ID, correlation ID, request ID, or deadline is found.
// Invalid IDs (non-ULID strings) are silently dropped,
// matching the behavior of ContextEnrichmentMiddleware for user IDs.
func EventOptionsFromContext(ctx context.Context) []event.Option {
	var opts []event.Option

	if userID := UserIDFromContext(ctx); !userID.IsZero() {
		opts = append(opts, event.WithUserID(userID))
	}

	// Propagate actor chain for audit trail.
	// ActorID = who the request acts AS (effective identity).
	// ImpersonatorID = who is REALLY authenticated (the admin).
	// When both are set, every event carries the full chain for compliance queries.
	if actorID := ActorIDFromContext(ctx); !actorID.IsZero() {
		opts = append(opts, event.WithCustom(MetadataKeyActorID, actorID.Get()))
	}

	if impersonatorID := ImpersonatorIDFromContext(ctx); !impersonatorID.IsZero() {
		opts = append(opts, event.WithCustom(MetadataKeyImpersonatorID, impersonatorID.Get()))
	}

	if cid := CorrelationIDFromContext(ctx); !cid.IsZero() {
		opts = append(opts, event.WithCorrelationID(cid))
	}

	if rid := RequestIDFromContext(ctx); !rid.IsZero() {
		opts = append(opts, event.WithRequestID(rid))
	}

	if _, ok := ctx.Deadline(); ok {
		opts = append(opts, event.FromContext(ctx))
	}

	if len(opts) == 0 {
		return nil
	}

	return opts
}

// EventOptionsFromContextWithSource is like EventOptionsFromContext but
// additionally sets the event source when serviceName is a valid event.Source.
// An empty or invalid serviceName is silently dropped (no source option added).
func EventOptionsFromContextWithSource(ctx context.Context, serviceName string) []event.Option {
	opts := EventOptionsFromContext(ctx)
	if serviceName == "" {
		return opts
	}

	if src, err := event.ParseSource(serviceName); err == nil {
		opts = append(opts, event.WithSource(src))
	}

	return opts
}
