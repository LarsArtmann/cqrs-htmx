package identitymodel

import (
	"bytes"
	"slices"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// UserState is the aggregate state for the User, reconstructed by folding events.
type UserState struct {
	Email            string
	DisplayName      string
	Credentials      []WebAuthnCredential
	ExternalAccounts []ExternalAccount
	Deleted          bool
	DeleteReason     string
	EmailVerified    bool
	TOTPEnabled      bool
	TOTPSecret       []byte
}

// Exists reports whether the user has been registered (has at least one event).
// It stays true after UserDeleted: a deleted user still HAS history. Use
// [UserState.IsActive] for the write-path liveness gate instead.
func (s UserState) Exists() bool {
	return s.Email != ""
}

// IsActive reports whether the user is usable: registered and not deleted.
// Deciders gate commands on this (not on Exists) so tombstoned streams stop
// accepting writes. Additive (S38c): every existing Exists() semantic is
// unchanged.
func (s UserState) IsActive() bool {
	return s.Exists() && !s.Deleted
}

// MembershipState is the aggregate state for a Membership (actor+tenant pair).
type MembershipState struct {
	ActorID  ActorID
	TenantID TenantID
	Roles    []Role
	Removed  bool
}

// Exists reports whether the membership is currently in effect: MemberRemoved
// folds to Removed=true and Exists() goes false. Unlike UserState, no
// IsActive split is needed — Exists already carries the liveness semantics.
func (s MembershipState) Exists() bool {
	return !s.ActorID.IsZero() && !s.Removed
}

// HasRole reports whether the membership grants the given role.
func (s MembershipState) HasRole(role Role) bool {
	return slices.Contains(s.Roles, role)
}

// TenantState is the aggregate state for a Tenant, reconstructed by folding events.
type TenantState struct {
	Name          string
	DisplayName   string
	Suspended     bool
	SuspendReason string
	Deleted       bool
	DeleteReason  string
}

// Exists reports whether the tenant has been created (has at least one event).
func (s TenantState) Exists() bool {
	return s.Name != "" && !s.Deleted
}

// IsActive reports whether the tenant exists and is not suspended or deleted.
func (s TenantState) IsActive() bool {
	return s.Exists() && !s.Suspended
}

// IsValid enforces struct-level invariants.
func (s TenantState) IsValid() bool {
	if s.Deleted && s.Suspended {
		return false
	}
	if !s.Suspended && s.SuspendReason != "" {
		return false
	}
	if !s.Deleted && s.DeleteReason != "" {
		return false
	}
	return true
}

// BotState is the aggregate state for a Bot (machine actor).
type BotState struct {
	Name      string
	OwnerID   UserID
	TokenHash []byte
	Scopes    []string
	Deleted   bool
}

// Exists reports whether the bot has been registered and not deleted. The
// Deleted tombstone is already folded here, so no separate IsActive split is
// required; [BotState.IsActive] exists as the uniform-vocabulary alias.
func (s BotState) Exists() bool {
	return s.Name != "" && !s.Deleted
}

// IsActive reports whether the bot is usable. BotState.Exists already folds
// the Deleted tombstone, so this is the uniform alias letting callers write
// `state.IsActive()` across every aggregate (S38c).
func (s BotState) IsActive() bool {
	return s.Exists()
}

// FoldUser applies an event to the current UserState, returning the new state.
// cqrs-lint:ignore(B005) canonical fold pattern: switch is the standard ES fold
func FoldUser(
	state UserState,
	evt event.Event,
) (UserState, error) {
	next := state

	switch evt.Type() {
	case EventUserRegistered:
		p, err := UnmarshalPayload[UserRegisteredPayload](evt)
		if err != nil {
			return state, err
		}
		next = UserState{
			Email:       p.Email,
			DisplayName: p.DisplayName,
		}

	case EventRolesUpdated:
		_, err := UnmarshalPayload[RolesUpdatedPayload](evt)
		if err != nil {
			return state, err
		}

	case EventEmailChanged:
		p, err := UnmarshalPayload[EmailChangedPayload](evt)
		if err != nil {
			return state, err
		}
		next.Email = p.Email
		next.EmailVerified = false
		next.TOTPEnabled = false
		next.TOTPSecret = nil

	case EventDisplayNameChanged:
		p, err := UnmarshalPayload[DisplayNameChangedPayload](evt)
		if err != nil {
			return state, err
		}
		next.DisplayName = p.DisplayName

	case EventUserDeleted:
		p, err := UnmarshalPayload[UserDeletedPayload](evt)
		if err != nil {
			return state, err
		}
		next.Deleted = true
		next.DeleteReason = p.Reason

	case EventCredentialAdded:
		p, err := UnmarshalPayload[CredentialAddedPayload](evt)
		if err != nil {
			return state, err
		}
		next.Credentials = append(next.Credentials, NewCredentialFromPayload(p, evt.OccurredAt()))

	case EventCredentialRemoved:
		var err error
		next, err = applyCredentialRemoved(next, evt)
		if err != nil {
			return state, err
		}

	case EventCredentialUsed:
		var err error
		next, err = applyCredentialUsed(next, evt)
		if err != nil {
			return state, err
		}

	case EventEmailVerified:
		_, err := UnmarshalPayload[EmailVerifiedPayload](evt)
		if err != nil {
			return state, err
		}
		next.EmailVerified = true

	case EventTOTPEnabled:
		p, err := UnmarshalPayload[TOTPEnabledPayload](evt)
		if err != nil {
			return state, err
		}
		next.TOTPEnabled = true
		next.TOTPSecret = slices.Clone(p.Secret)

	case EventTOTPDisabled:
		next.TOTPEnabled = false
		next.TOTPSecret = nil

	case EventExternalAccountLinked:
		p, err := UnmarshalPayload[ExternalAccountLinkedPayload](evt)
		if err != nil {
			return state, err
		}
		next.ExternalAccounts = append(next.ExternalAccounts, ExternalAccount{
			Provider:    p.Provider,
			Subject:     p.Subject,
			Email:       p.Email,
			DisplayName: p.DisplayName,
			LinkedAt:    evt.OccurredAt(),
		})

	case EventExternalAccountUnlinked:
		p, err := UnmarshalPayload[ExternalAccountUnlinkedPayload](evt)
		if err != nil {
			return state, err
		}
		filtered := make([]ExternalAccount, 0, len(next.ExternalAccounts))
		for _, ea := range next.ExternalAccounts {
			if ea.Provider != p.Provider || ea.Subject != p.Subject {
				filtered = append(filtered, ea)
			}
		}
		next.ExternalAccounts = filtered

	default:
		return state, errorfamily.NewRejection(
			"usermgmt.user.unknown_event",
			"FoldUser received unknown event type: "+string(evt.Type()),
		)
	}

	return next, nil
}

// FoldMembership applies an event to the current MembershipState.
// cqrs-lint:ignore(B005) canonical fold pattern: switch is the standard ES fold
func FoldMembership(
	state MembershipState,
	evt event.Event,
) (MembershipState, error) {
	next := state

	switch evt.Type() {
	case EventMemberAdded:
		p, err := UnmarshalPayload[MemberAddedPayload](evt)
		if err != nil {
			return state, err
		}
		roles := slices.Clone(p.Roles)
		kind, err := ActorKindFromString(p.ActorKind)
		if err != nil {
			return state, err
		}
		tenantID, err := ParseTenantID(p.TenantID)
		if err != nil {
			return state, err
		}
		next.ActorID = NewActorID(kind, p.ActorID)
		next.TenantID = tenantID
		next.Roles = roles
		next.Removed = false

	case EventMemberRolesChanged:
		p, err := UnmarshalPayload[MemberRolesChangedPayload](evt)
		if err != nil {
			return state, err
		}
		roles := slices.Clone(p.Roles)
		next.Roles = roles

	case EventMemberRemoved:
		next.Roles = nil
		next.Removed = true

	default:
		return state, errorfamily.NewRejection(
			"usermgmt.membership.unknown_event",
			"FoldMembership received unknown event type: "+string(evt.Type()),
		)
	}

	return next, nil
}

// FoldTenant applies an event to the current TenantState.
// cqrs-lint:ignore(B005) canonical fold pattern: switch is the standard ES fold
func FoldTenant(
	state TenantState,
	evt event.Event,
) (TenantState, error) {
	next := state

	switch evt.Type() {
	case EventTenantCreated:
		p, err := UnmarshalPayload[TenantCreatedPayload](evt)
		if err != nil {
			return state, err
		}
		next.Name = p.Name
		next.DisplayName = p.DisplayName
		next.Suspended = false
		next.Deleted = false
		// Clear tombstone/suspension reasons too: a recreated tenant must not
		// inherit the previous incarnation's reasons (FoldUser resets the
		// whole state; TenantState.IsValid would reject the stale reasons).
		next.SuspendReason = ""
		next.DeleteReason = ""

	case EventTenantSuspended:
		p, err := UnmarshalPayload[TenantSuspendedPayload](evt)
		if err != nil {
			return state, err
		}
		next.Suspended = true
		next.SuspendReason = p.Reason

	case EventTenantReactivated:
		next.Suspended = false
		next.SuspendReason = ""

	case EventTenantDeleted:
		p, err := UnmarshalPayload[TenantDeletedPayload](evt)
		if err != nil {
			return state, err
		}
		next.Deleted = true
		next.Suspended = false
		next.SuspendReason = ""
		next.DeleteReason = p.Reason

	default:
		return state, errorfamily.NewRejection(
			"usermgmt.tenant.unknown_event",
			"FoldTenant received unknown event type: "+string(evt.Type()),
		)
	}

	return next, nil
}

// FoldBot applies an event to the current BotState.
// cqrs-lint:ignore(B005) canonical fold pattern: switch is the standard ES fold
func FoldBot(
	state BotState,
	evt event.Event,
) (BotState, error) {
	next := state

	switch evt.Type() {
	case EventBotRegistered:
		p, err := UnmarshalPayload[BotRegisteredPayload](evt)
		if err != nil {
			return state, err
		}
		scopes := slices.Clone(p.Scopes)
		next.Name = p.Name
		next.OwnerID = p.OwnerID
		next.TokenHash = slices.Clone(p.TokenHash)
		next.Scopes = scopes
		next.Deleted = false

	case EventBotDeleted:
		next.Deleted = true

	default:
		return state, errorfamily.NewRejection(
			"usermgmt.bot.unknown_event",
			"FoldBot received unknown event type: "+string(evt.Type()),
		)
	}

	return next, nil
}

func ActorKindFromString(s string) (ActorKind, error) {
	switch s {
	case ActorKindUserStr:
		return id.ActorUser, nil
	case ActorKindBotStr:
		return id.ActorBot, nil
	case ActorKindSystemStr:
		return id.ActorSystem, nil
	case ActorKindServiceStr:
		return id.ActorService, nil
	default:
		return id.ActorUnknown, errorfamily.NewRejection(
			"usermgmt.membership.unknown_actor_kind",
			"unknown actor kind: "+s,
		)
	}
}

// applyCredentialUsed applies a CredentialUsed event to the credential it
// names. It copies the credential slice before writing: FoldUser's `next :=
// state` is a shallow copy, so mutating an element in place would write
// through to the caller's state and make folding impure.
func applyCredentialUsed(state UserState, evt event.Event) (UserState, error) {
	p, err := UnmarshalPayload[CredentialUsedPayload](evt)
	if err != nil {
		return state, err
	}

	use := CredentialUse{
		CredentialID: p.CredentialID,
		SignCount:    p.SignCount,
		UserPresent:  p.UserPresent,
		UserVerified: p.UserVerified,
		BackupState:  p.BackupState,
	}

	credentials := make([]WebAuthnCredential, len(state.Credentials))
	copy(credentials, state.Credentials)

	for i := range credentials {
		if bytes.Equal(credentials[i].ID, p.CredentialID) {
			credentials[i] = credentials[i].ApplyUse(use)
			state.Credentials = credentials
			return state, nil
		}
	}

	return state, errorfamily.NewCorruption(
		"identity.credential_used.unknown_credential",
		"CredentialUsed event for a credential the stream never added",
	)
}

// applyCredentialRemoved drops the credential the event names, allocating a new
// slice so folding never writes through to the caller's state.
func applyCredentialRemoved(state UserState, evt event.Event) (UserState, error) {
	p, err := UnmarshalPayload[CredentialRemovedPayload](evt)
	if err != nil {
		return state, err
	}

	filtered := make([]WebAuthnCredential, 0, len(state.Credentials))

	for _, c := range state.Credentials {
		if !bytes.Equal(c.ID, p.ID) {
			filtered = append(filtered, c)
		}
	}

	state.Credentials = filtered

	return state, nil
}
