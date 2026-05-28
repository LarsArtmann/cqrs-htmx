package usermgmt

import (
	"time"
)

// EventHandler is called after successful domain operations.
// The event argument is one of the *Event types defined in this file.
// Handlers run synchronously within the service method — keep them fast
// or dispatch to a background worker. Errors from the handler are logged
// but do not fail the operation.
type EventHandler func(userID UserID, event any)

// UserRegisteredEvent is emitted after a successful user registration.
type UserRegisteredEvent struct {
	Email       string    `json:"email"`
	DisplayName string    `json:"display_name,omitempty"`
	Roles       []Role    `json:"roles"`
	OccurredAt  time.Time `json:"occurred_at"`
}

// UserLoggedInEvent is emitted after a successful login.
type UserLoggedInEvent struct {
	Email      string    `json:"email"`
	OccurredAt time.Time `json:"occurred_at"`
}

// PasswordChangedEvent is emitted after a successful password change.
type PasswordChangedEvent struct {
	OccurredAt time.Time `json:"occurred_at"`
}

// RolesUpdatedEvent is emitted after a successful role update.
type RolesUpdatedEvent struct {
	Roles      []Role    `json:"roles"`
	Domain     string    `json:"domain"`
	OccurredAt time.Time `json:"occurred_at"`
}
