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

type contextKey string

const userIDKey contextKey = "cqrshtmx_user_id"

const correlationIDKey contextKey = "cqrshtmx_correlation_id"

// WithCorrelationID stores a correlation/request ID in the context.
func WithCorrelationID(ctx context.Context, correlationID string) context.Context {
	return context.WithValue(ctx, correlationIDKey, correlationID)
}

// CorrelationIDFromContext retrieves the correlation ID stored by WithCorrelationID.
// Returns an empty string if no correlation ID is present.
func CorrelationIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(correlationIDKey).(string)
	return v
}

// WithUserID stores a strongly-typed user ID in the context for downstream CQRS handlers.
func WithUserID(ctx context.Context, userID UserID) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// UserIDFromContext retrieves the user ID stored by WithUserID.
// Returns the zero value of UserID if no user ID is present.
func UserIDFromContext(ctx context.Context) UserID {
	v, _ := ctx.Value(userIDKey).(UserID)
	return v
}

// EventOptionsFromContext builds event.Options from request context,
// propagating user identity into event metadata for auditing and tracing.
//
// Returns nil options if no user ID is found in context.
func EventOptionsFromContext(ctx context.Context) []event.Option {
	userID := UserIDFromContext(ctx)
	if userID.IsZero() {
		return nil
	}

	return []event.Option{event.WithUserID(userID)}
}
