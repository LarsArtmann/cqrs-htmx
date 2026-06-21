package usermgmt

import (
	"fmt"

	brandid "github.com/larsartmann/go-branded-id"
)

type userBrand struct{}

func (userBrand) Name() string { return "User" }

// UserID is a branded type for user identifiers backed by ULID strings.
// Use NewUserID to construct instances from string values.
type UserID = brandid.ID[userBrand, string]

// NewUserID constructs a UserID from a string value.
// In tests, prefer passing a known ULID string for determinism.
func NewUserID(s string) UserID { return brandid.NewID[userBrand](s) }

type tenantBrand struct{}

func (tenantBrand) Name() string { return "Tenant" }

// TenantID is a branded type for tenant identifiers.
// It replaces the free-form domain string used in Casbin policies.
type TenantID = brandid.ID[tenantBrand, string]

// NewTenantID constructs a TenantID from a string value.
func NewTenantID(s string) TenantID { return brandid.NewID[tenantBrand](s) }

type botBrand struct{}

func (botBrand) Name() string { return "Bot" }

// BotID is a branded type for bot (machine actor) identifiers.
type BotID = brandid.ID[botBrand, string]

// NewBotID constructs a BotID from a string value.
func NewBotID(s string) BotID { return brandid.NewID[botBrand](s) }

// ActorKind discriminates between human and machine actors.
type ActorKind int

const (
	// ActorUser represents a human actor authenticated via WebAuthn or OAuth2.
	ActorUser ActorKind = iota
	// ActorBot represents a machine actor authenticated via API token.
	ActorBot
)

// String returns the lowercase kind name used in prefixed identifiers.
func (k ActorKind) String() string {
	switch k {
	case ActorUser:
		return "user"
	case ActorBot:
		return "bot"
	default:
		return "unknown"
	}
}

// ActorID is a kind-discriminated identifier for any actor (user or bot).
// It is the identity type used across sessions, context propagation, and
// event metadata. Construct via ActorIDFromUser or ActorIDFromBot.
type ActorID struct {
	kind ActorKind
	raw  string
}

// NewActorID creates an ActorID from a kind and raw string value.
func NewActorID(kind ActorKind, raw string) ActorID {
	return ActorID{kind: kind, raw: raw}
}

// ActorIDFromUser creates an ActorID from a UserID.
func ActorIDFromUser(uid UserID) ActorID {
	return ActorID{kind: ActorUser, raw: uid.Get()}
}

// ActorIDFromBot creates an ActorID from a BotID.
func ActorIDFromBot(bid BotID) ActorID {
	return ActorID{kind: ActorBot, raw: bid.Get()}
}

// Kind returns the actor kind (user or bot).
func (a ActorID) Kind() ActorKind { return a.kind }

// String returns the raw identifier string without kind prefix.
func (a ActorID) String() string { return a.raw }

// IsZero reports whether the ActorID is uninitialized.
func (a ActorID) IsZero() bool { return a.raw == "" }

// PrefixedString returns the identifier with kind prefix (e.g. "user:01JX...").
func (a ActorID) PrefixedString() string {
	return fmt.Sprintf("%s:%s", a.kind, a.raw)
}
