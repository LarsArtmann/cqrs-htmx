package cqrshtmx

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

type contextKey string

const userIDKey contextKey = "cqrshtmx_user_id"

// WithUserID stores a user ID in the context for downstream CQRS handlers.
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// UserIDFromContext retrieves the user ID stored by WithUserID.
// Returns an empty string if no user ID is present.
func UserIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(userIDKey).(string)
	return v
}

// EventOptionsFromContext builds event.Options from request context,
// propagating user identity into event metadata for auditing and tracing.
//
// Returns nil options if no user ID is found in context.
func EventOptionsFromContext(ctx context.Context) []event.Option {
	userID := UserIDFromContext(ctx)
	if userID == "" {
		return nil
	}

	parsed, err := id.ParseUserID(userID)
	if err != nil {
		return []event.Option{event.WithUserID(id.UserID{})}
	}

	return []event.Option{event.WithUserID(parsed)}
}
