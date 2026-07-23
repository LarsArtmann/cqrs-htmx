package usermgmt

import (
	"time"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
)

type (
	User          = identitymodel.User
	Session       = identitymodel.Session
	SessionOrigin = identitymodel.SessionOrigin
	DirectLogin   = identitymodel.DirectLogin
	Impersonation = identitymodel.Impersonation
	Membership    = identitymodel.Membership
)

func NewUser(id UserID, email, displayName string) *User {
	return identitymodel.NewUser(id, email, displayName)
}

func NewSession(userID UserID, ttl time.Duration) (*Session, error) {
	return identitymodel.NewSession(userID, ttl)
}

func NewImpersonationSession(target, impersonator ActorID, reason string, ttl time.Duration) (*Session, error) {
	return identitymodel.NewImpersonationSession(target, impersonator, reason, ttl)
}
