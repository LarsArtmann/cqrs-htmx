package usermgmt

import (
	"crypto/sha256"
	"encoding/json/v2"
	"fmt"
	"strings"

	brandid "github.com/larsartmann/go-branded-id"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/oklog/ulid/v2"
)

// UserID is a strongly-typed identifier for users, backed by ULID.
// This is an alias of id.UserID from go-cqrs-lite, ensuring type-level
// interoperability with the root cqrs-htmx module and event metadata.
// See ADR-0018 for the unification rationale.
type UserID = id.UserID

// NewUserID creates a UserID from a string. If the string is a valid ULID,
// it is parsed directly. Otherwise, a deterministic ULID is derived via
// SHA-256 hashing — this preserves backward compatibility for callers that
// pass short identifiers (e.g., "u1", "alice") while ensuring the resulting
// type is always a valid id.UserID.
//
// For production code that validates external input, use ParseUserID instead.
// SECURITY NOTE: the silent-hash fallback is a footgun — two callers passing
// "alice" get the SAME UserID, and any non-ULID string silently becomes a
// valid-looking ID. Use [ParseUserID] (strict) or [SyntheticUserID] (explicit
// hashing) in new code. NewUserID is retained for backward compatibility.
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
// SHA-256 hashing it into a ULID. The explicit name signals the hashing
// behavior — use this only when you intentionally want a stable synthetic ID
// derived from a non-ULID input (e.g. deriving a test ID from a username).
// Two calls with the same input return the same UserID. For validated ULID
// input use [ParseUserID]. NewUserID delegates here for non-ULID strings.
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
		panic(fmt.Sprintf("usermgmt.MustParseUserID(%q): %v", s, err))
	}
	return uid
}

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

// actorKindUserStr and actorKindBotStr are the string representations
// used in ActorKind.String(), ActorID prefixes, and event payloads.
const (
	actorKindUserStr = "user"
	actorKindBotStr  = "bot"
)

// String returns the lowercase kind name used in prefixed identifiers.
func (k ActorKind) String() string {
	switch k {
	case ActorUser:
		return actorKindUserStr
	case ActorBot:
		return actorKindBotStr
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
// Panics if kind is not ActorUser or ActorBot — use ActorIDFromUser or
// ActorIDFromBot for type-safe construction.
func NewActorID(kind ActorKind, raw string) ActorID {
	switch kind {
	case ActorUser, ActorBot:
	default:
		panic(fmt.Sprintf("NewActorID: invalid ActorKind %d", kind))
	}
	return ActorID{kind: kind, raw: raw}
}

// ActorIDFromUser creates an ActorID from a UserID.
func ActorIDFromUser(uid UserID) ActorID {
	return ActorID{kind: ActorUser, raw: uid.Get().String()}
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

// ParseActorID reconstructs an [ActorID] from a [ActorID.PrefixedString] value
// (e.g. "user:01JX...", "bot:name"). It is the inverse of PrefixedString, used
// to carry actor identity through URLs and forms. Values without a recognized
// prefix are treated as a user actor.
func ParseActorID(s string) ActorID {
	if after, ok := strings.CutPrefix(s, actorKindBotStr+":"); ok {
		return ActorID{kind: ActorBot, raw: after}
	}
	if _, after, ok := strings.Cut(s, ":"); ok {
		return ActorID{kind: ActorUser, raw: after}
	}
	return ActorID{kind: ActorUser, raw: s}
}

// MarshalJSON encodes ActorID as its prefixed string form (e.g. "user:01JX...").
func (a ActorID) MarshalJSON() ([]byte, error) {
	data, err := json.Marshal(a.PrefixedString())
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(
			err, "usermgmt.actor_id.marshal", "marshal ActorID")
	}
	return data, nil
}

// UnmarshalJSON decodes ActorID from its prefixed string form.
func (a *ActorID) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return errorfamily.WrapRejection(
			err, "usermgmt.actor_id.unmarshal", "unmarshal ActorID")
	}
	*a = ParseActorID(s)
	return nil
}
