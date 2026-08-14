package usermgmt

import (
	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
)

// All identity types now live in identity-model. These aliases re-export them
// from usermgmt for backward compatibility. Methods on ActorID and ActorKind
// are inherited automatically through the type aliases.

type (
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	UserID = identitymodel.UserID
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	TenantID = identitymodel.TenantID
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	BotID = identitymodel.BotID
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	ActorKind = identitymodel.ActorKind
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	ActorID = identitymodel.ActorID
)

const (
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	ActorUser = identitymodel.ActorUser
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	ActorBot = identitymodel.ActorBot
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	ActorSystem = identitymodel.ActorSystem
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	ActorService = identitymodel.ActorService
)

// Deprecated: NewUserID silently hashes non-ULID strings, masking invalid input.
// Use [ParseUserID] for strict ULID validation or [SyntheticUserID] for explicit hashing.
// This is a re-export of identitymodel.NewUserID.
func NewUserID(s string) UserID { return identitymodel.NewUserID(s) }

// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
func SyntheticUserID(s string) UserID { return identitymodel.SyntheticUserID(s) }

// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
func ParseUserID(s string) (UserID, error) { return identitymodel.ParseUserID(s) }

// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
func MustParseUserID(s string) UserID { return identitymodel.MustParseUserID(s) }

// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
func GenerateUserID() UserID { return identitymodel.GenerateUserID() }

// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
func NewTenantID(s string) TenantID { return identitymodel.NewTenantID(s) }

// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
func NewBotID(s string) BotID { return identitymodel.NewBotID(s) }

// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
func NewActorID(kind ActorKind, raw string) ActorID {
	return identitymodel.NewActorID(kind, raw)
}

// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
func ActorIDFromUser(uid UserID) ActorID { return identitymodel.ActorIDFromUser(uid) }

// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
func ActorIDAsUserID(actorID ActorID) (UserID, bool) {
	return identitymodel.ActorIDAsUserID(actorID)
}

// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
func ActorIDFromBot(bid BotID) ActorID { return identitymodel.ActorIDFromBot(bid) }

// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
func ParseActorID(s string) (ActorID, error) { return identitymodel.ParseActorID(s) }
