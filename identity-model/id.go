package identitymodel

import (
	"crypto/sha256"
	"fmt"

	brandid "github.com/larsartmann/go-branded-id"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/oklog/ulid/v2"
)

// UserID is a strongly-typed identifier for users, backed by ULID.
// This is an alias of id.UserID from go-cqrs-lite, ensuring type-level
// interoperability with the root cqrs-htmx module and event metadata.
type UserID = id.UserID

// NewUserID creates a UserID from a string. If the string is a valid ULID,
// it is parsed directly. Otherwise, a deterministic ULID is derived via
// SHA-256 hashing.
//
// Deprecated: NewUserID silently hashes non-ULID strings, which masks invalid
// input and produces colliding IDs for unrelated callers. Use [ParseUserID]
// for strict ULID validation, or [SyntheticUserID] when an explicit, stable
// hash of arbitrary input is the intent.
func NewUserID(s string) UserID {
	if s == "" {
		var zero UserID
		return zero
	}
	if uid, err := id.ParseUserID(s); err == nil {
		return uid
	}
	return SyntheticUserID(s)
}

// SyntheticUserID derives a deterministic UserID from an arbitrary string by
// SHA-256 hashing it into a ULID. Two calls with the same input return the
// same UserID. For validated ULID input use [ParseUserID].
func SyntheticUserID(s string) UserID {
	h := sha256.Sum256([]byte(s))
	var u ulid.ULID
	copy(u[:], h[:16])

	return brandid.NewID[id.UserMarker](u)
}

// ParseUserID converts a ULID string to a UserID, returning an error for invalid ULIDs.
func ParseUserID(s string) (UserID, error) {
	uid, err := id.ParseUserID(s)
	if err != nil {
		return uid, errorfamily.Wrapf(err, event.Rejection, "usermgmt.userid.invalid", "invalid user id %q", s)
	}
	return uid, nil
}

// MustParseUserID converts a ULID string to a UserID, panicking on invalid input.
func MustParseUserID(s string) UserID {
	uid, err := id.ParseUserID(s)
	if err != nil {
		panic(
			fmt.Sprintf("identitymodel.MustParseUserID(%q): %v", s, err),
		)
	}
	return uid
}

// GenerateUserID creates a fresh random UserID (new ULID) for production use.
func GenerateUserID() UserID {
	return SyntheticUserID(ulid.Make().String())
}

type tenantBrand struct{}

func (tenantBrand) Name() string { return "Tenant" }

// TenantID is a branded type for tenant identifiers.
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
// This is an alias of id.ActorKind from go-cqrs-lite, ensuring full
// interoperability with event/command metadata (ADR-0111).
type ActorKind = id.ActorKind

const (
	// ActorUser represents a human actor authenticated via WebAuthn or OAuth2.
	ActorUser = id.ActorUser
	// ActorBot represents a machine actor authenticated via API token.
	ActorBot = id.ActorBot
	// ActorSystem represents an internal system process (scheduler, GC, etc.).
	ActorSystem = id.ActorSystem
	// ActorService represents a named service producing records on behalf of users.
	ActorService = id.ActorService
)

const (
	ActorKindUserStr    = "user"
	ActorKindBotStr     = "bot"
	ActorKindSystemStr  = "system"
	ActorKindServiceStr = "service"
)

// ActorID is a kind-discriminated identifier for any actor (user, bot,
// system, or service). This is an alias of id.ActorID from go-cqrs-lite,
// ensuring full interoperability with event/command metadata (ADR-0111).
type ActorID = id.ActorID

// NewActorID creates an ActorID from a kind and raw string value.
func NewActorID(kind ActorKind, raw string) ActorID {
	return id.NewActorID(kind, raw)
}

// ActorIDFromUser creates an ActorID from a UserID.
func ActorIDFromUser(uid UserID) ActorID {
	return id.NewUserActor(uid)
}

// ActorIDAsUserID returns the UserID of a human-user actor and true. For any
// other actor kind (bot, system, service, unknown) it returns the zero UserID
// and false — callers must not persist a UserID derived from those kinds,
// which would create invalid references in audit trails, sessions, and roles.
//
// Use this instead of hand-rolling the kind guard:
//
//	if uid, ok := identitymodel.ActorIDAsUserID(actorID); ok {
//	    entry.UserID = uid
//	}
func ActorIDAsUserID(actorID ActorID) (UserID, bool) {
	if actorID.Kind() != ActorUser {
		var zero UserID
		return zero, false
	}
	//nolint:staticcheck // SA1019: preserves historical NewUserID semantics (parse-or-synthesize)
	return NewUserID(actorID.String()), true
}

// ActorIDFromBot creates an ActorID from a BotID.
func ActorIDFromBot(bid BotID) ActorID {
	return id.NewBotActor(bid.Get())
}

// ParseActorID reconstructs an ActorID from its prefixed string form
// ("kind:raw"). Returns an error for malformed input.
func ParseActorID(s string) (ActorID, error) {
	a, err := id.ParseActorID(s)
	if err != nil {
		return a, errorfamily.Wrapf(err, event.Rejection,
			"identitymodel.actor_id.parse", "parse ActorID %q", s)
	}
	return a, nil
}
