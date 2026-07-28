package usermgmt

import (
	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
)

// All identity types now live in identity-model. These aliases re-export them
// from usermgmt for backward compatibility. Methods on ActorID and ActorKind
// are inherited automatically through the type aliases.

type (
	UserID    = identitymodel.UserID
	TenantID  = identitymodel.TenantID
	BotID     = identitymodel.BotID
	ActorKind = identitymodel.ActorKind
	ActorID   = identitymodel.ActorID
)

const (
	ActorUser = identitymodel.ActorUser
	ActorBot  = identitymodel.ActorBot
)

// Deprecated: NewUserID silently hashes non-ULID strings, masking invalid input.
// Use [ParseUserID] for strict ULID validation or [SyntheticUserID] for explicit hashing.
func NewUserID(s string) UserID { return identitymodel.NewUserID(s) } //nolint:staticcheck // backward-compat shim, intentionally delegates
func SyntheticUserID(s string) UserID      { return identitymodel.SyntheticUserID(s) }
func ParseUserID(s string) (UserID, error) { return identitymodel.ParseUserID(s) }
func MustParseUserID(s string) UserID      { return identitymodel.MustParseUserID(s) }
func GenerateUserID() UserID               { return identitymodel.GenerateUserID() }
func NewTenantID(s string) TenantID        { return identitymodel.NewTenantID(s) }
func NewBotID(s string) BotID              { return identitymodel.NewBotID(s) }
func NewActorID(kind ActorKind, raw string) ActorID {
	return identitymodel.NewActorID(kind, raw)
}
func ActorIDFromUser(uid UserID) ActorID { return identitymodel.ActorIDFromUser(uid) }
func ActorIDFromBot(bid BotID) ActorID   { return identitymodel.ActorIDFromBot(bid) }
func ParseActorID(s string) ActorID      { return identitymodel.ParseActorID(s) }
