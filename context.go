package cqrshtmx

import (
	"context"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

// UserID is a strongly-typed identifier for users, preventing accidental mixing
// with other ID types at compile time.
type UserID = id.UserID

// NewUserID generates a new random UserID backed by a ULID.
func NewUserID() UserID {
	return id.NewUserID()
}

// ParseUserID converts a string to a UserID.
// Returns an error if the input is not a valid ULID.
func ParseUserID(s string) (UserID, error) {
	userID, err := id.ParseUserID(s)
	if err != nil {
		return UserID{}, fmt.Errorf("parse user id: %w", err)
	}
	return userID, nil
}

// MustParseUserID converts a string to a UserID, panicking on error.
func MustParseUserID(s string) UserID {
	return id.MustParseUserID(s)
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
	correlationID, err := id.ParseCorrelationID(s)
	if err != nil {
		return CorrelationID{}, fmt.Errorf("parse correlation id: %w", err)
	}
	return correlationID, nil
}

// MustParseCorrelationID converts a string to a CorrelationID, panicking on error.
func MustParseCorrelationID(s string) CorrelationID {
	return id.MustParseCorrelationID(s)
}

type userIDKey struct{}

type correlationIDKey struct{}

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
// propagating user identity and correlation ID into event metadata for
// auditing, tracing, and distributed request correlation.
//
// Returns nil options if neither user ID nor correlation ID is found.
// Invalid correlation IDs (non-ULID strings) are silently dropped,
// matching the behavior of ContextEnrichmentMiddleware for user IDs.
func EventOptionsFromContext(ctx context.Context) []event.Option {
	var opts []event.Option

	if userID := UserIDFromContext(ctx); !userID.IsZero() {
		opts = append(opts, event.WithUserID(userID))
	}

	if cid := CorrelationIDFromContext(ctx); !cid.IsZero() {
		opts = append(opts, event.WithCorrelationID(cid))
	}

	if len(opts) == 0 {
		return nil
	}

	return opts
}
