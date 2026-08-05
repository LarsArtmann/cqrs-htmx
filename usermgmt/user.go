package usermgmt

import (
	"time"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
)

type (
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	User = identitymodel.User
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	Session = identitymodel.Session
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	SessionOrigin = identitymodel.SessionOrigin
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	DirectLogin = identitymodel.DirectLogin
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	Impersonation = identitymodel.Impersonation
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	Membership = identitymodel.Membership
)

// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
func NewUser(id UserID, email, displayName string) *User {
	return identitymodel.NewUser(id, email, displayName)
}

// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
func NewSession(userID UserID, ttl time.Duration) (*Session, error) {
	return identitymodel.NewSession(userID, ttl)
}

// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
func NewImpersonationSession(target, impersonator ActorID, reason string, ttl time.Duration) (*Session, error) {
	return identitymodel.NewImpersonationSession(target, impersonator, reason, ttl)
}
