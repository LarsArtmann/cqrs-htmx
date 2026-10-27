package identitymodel

import (
	"crypto/sha256"
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
	if uid, err := ParseUserID(s); err == nil {
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
// A leading brand prefix ("StreamMarker:<ulid>", as emitted by branded-id String())
// is tolerated and stripped before the strict parse, mirroring go-crs-lite's
// ParseStreamID: the prefix carries no identity information. ULIDs never contain
// colons, so a colon is always a brand separator, never part of the identity.
func ParseUserID(s string) (UserID, error) {
	uid, err := id.ParseUserID(strippedBrandPrefix(s))
	if err != nil {
		return uid, errorfamily.Wrapf(err, event.Rejection, "usermgmt.userid.invalid", "invalid user id %q", s)
	}
	return uid, nil
}

// strippedBrandPrefix removes a leading "<Brand>:..." prefix if present.
func strippedBrandPrefix(s string) string {
	if _, after, ok := strings.Cut(s, ":"); ok {
		return after
	}
	return s
}

// MustParseUserID converts a ULID string to a UserID, panicking on invalid input.
// Like [ParseUserID], it tolerates (and strips) a leading brand prefix.
func MustParseUserID(s string) UserID {
	uid, err := ParseUserID(s)
	if err != nil {
		panic(
			fmt.Sprintf("identitymodel.MustParseUserID(%q): %v", s, err),
		)
	}
	return uid
}

// GenerateUserID creates a fresh random UserID (new ULID) for production use.
// The ULID's timestamp is preserved, so IDs minted later sort later across
// millisecond boundaries (ULID's core ordering guarantee). Do NOT route
// fresh ULIDs through [SyntheticUserID] — the SHA-256 derivation destroys
// the timestamp and with it the ordering.
func GenerateUserID() UserID {
	return brandid.NewID[id.UserMarker](ulid.Make())
}

type tenantBrand struct{}

func (tenantBrand) Name() string { return "Tenant" }

// TenantID is a branded type for tenant identifiers.
type TenantID = brandid.ID[tenantBrand, string]

// NewTenantID constructs a TenantID from a string value.
func NewTenantID(s string) TenantID { return brandid.NewID[tenantBrand](s) }

// ParseTenantID converts a string to a TenantID, returning an error for
// empty or whitespace-padded input. The brand display prefix ("Tenant:", as
// emitted by TenantID.String()) is tolerated and stripped, mirroring
// [ParseUserID]: the prefix carries no identity information. For arbitrary
// trusted input use [NewTenantID]; this is the strict path for RECONSTRUCTION
// (event folds) and external input.
func ParseTenantID(s string) (TenantID, error) {
	v, err := parseBrandedComponent(s, "Tenant")
	if err != nil {
		return TenantID{}, errorfamily.Wrapf(err, event.Rejection,
			"identitymodel.tenant_id.invalid", "invalid tenant id %q", s)
	}

	return NewTenantID(v), nil
}

// parseBrandedComponent strips a single exact "<Brand>:" display prefix and
// rejects empty or whitespace-padded values — the ambiguous forms that would
// otherwise silently become distinct identities.
func parseBrandedComponent(s, brand string) (string, error) {
	v := strings.TrimPrefix(s, brand+":")

	if v == "" || v != strings.TrimSpace(v) {
		return "", errorfamily.NewRejection("identitymodel.branded_id.invalid",
			"value is empty or whitespace-padded")
	}

	return v, nil
}

type botBrand struct{}

func (botBrand) Name() string { return "Bot" }

// BotID is a branded type for bot (machine actor) identifiers.
type BotID = brandid.ID[botBrand, string]

// NewBotID constructs a BotID from a string value.
func NewBotID(s string) BotID { return brandid.NewID[botBrand](s) }

// ParseBotID converts a string to a BotID, returning an error for empty or
// whitespace-padded input. The brand display prefix ("Bot:", as emitted by
// BotID.String()) is tolerated and stripped. For arbitrary trusted input use
// [NewBotID]; this is the strict path for RECONSTRUCTION and external input.
func ParseBotID(s string) (BotID, error) {
	v, err := parseBrandedComponent(s, "Bot")
	if err != nil {
		return BotID{}, errorfamily.Wrapf(err, event.Rejection,
			"identitymodel.bot_id.invalid", "invalid bot id %q", s)
	}

	return NewBotID(v), nil
}

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

	uid, err := ParseUserID(actorID.Raw())
	if err != nil {
		// A user-kind actor whose raw value is not a ULID is malformed;
		// persisting a hash of it would fabricate a different identity.
		var zero UserID
		return zero, false
	}

	return uid, true
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
