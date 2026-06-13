package cqrshtmx

import (
	"context"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

// UserID is a strongly-typed identifier for users, preventing accidental mixing
// with other ID types at compile time.
type UserID = id.UserID

// NewUserID generates a new random UserID backed by a ULID.
func NewUserID() UserID {
	return id.NewUserID()
}

// parseID is a generic helper that wraps an ID parse function with contextual error messages.
func parseID[T any](s string, parse func(string) (T, error), label string) (T, error) {
	v, err := parse(s)
	if err != nil {
		return v, fmt.Errorf("parse %s: %w", label, err)
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
		panic(fmt.Sprintf("MustParseUserID: %v", err))
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
		panic(fmt.Sprintf("MustParseCorrelationID: %v", err))
	}
	return v
}

type userIDKey struct{}

type correlationIDKey struct{}

type requestIDKey struct{}

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
		panic(fmt.Sprintf("MustParseRequestID: %v", err))
	}
	return v
}

// WithRequestID stores a strongly-typed request ID in the context.
func WithRequestID(ctx context.Context, requestID RequestID) context.Context {
	return context.WithValue(ctx, requestIDKey{}, requestID)
}

// RequestIDFromContext retrieves the request ID stored by WithRequestID.
// Returns the zero value of RequestID if no request ID is present.
func RequestIDFromContext(ctx context.Context) RequestID {
	v, _ := ctx.Value(requestIDKey{}).(RequestID)
	return v
}

// WithCorrelationID stores a strongly-typed correlation ID in the context.
func WithCorrelationID(ctx context.Context, correlationID CorrelationID) context.Context {
	return context.WithValue(ctx, correlationIDKey{}, correlationID)
}

// CorrelationIDFromContext retrieves the correlation ID stored by WithCorrelationID.
// Returns the zero value of CorrelationID if no correlation ID is present.
func CorrelationIDFromContext(ctx context.Context) CorrelationID {
	v, _ := ctx.Value(correlationIDKey{}).(CorrelationID)
	return v
}

// WithUserID stores a strongly-typed user ID in the context for downstream CQRS handlers.
func WithUserID(ctx context.Context, userID UserID) context.Context {
	return context.WithValue(ctx, userIDKey{}, userID)
}

// UserIDFromContext retrieves the user ID stored by WithUserID.
// Returns the zero value of UserID if no user ID is present.
func UserIDFromContext(ctx context.Context) UserID {
	v, _ := ctx.Value(userIDKey{}).(UserID)
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
