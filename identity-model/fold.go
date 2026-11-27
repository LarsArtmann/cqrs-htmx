package identitymodel

import (
	"bytes"
	"slices"
	"strings"

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
	// MergedInto is non-zero only on a merged-away user and names the
	// survivor. Deleted is then true as well, so every existing liveness gate
	// refuses the stream without knowing about merges.
	MergedInto UserID
	// MergedUserIDs lists the duplicate users that were merged into this one.
	MergedUserIDs []UserID
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

// HasCredential reports whether the user holds a credential with the given ID.
func (s UserState) HasCredential(credentialID []byte) bool {
	return slices.ContainsFunc(s.Credentials, func(cred WebAuthnCredential) bool {
		return bytes.Equal(cred.ID, credentialID)
	})
}

// IsMerged reports whether this user was merged into another one, in which
// case [UserState.MergedInto] names the survivor.
func (s UserState) IsMerged() bool {
	return !s.MergedInto.IsZero()
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
	// MergedInto is non-zero only on a merged-away tenant and names the
	// survivor. Deleted is then true as well, so every existing liveness gate
	// refuses the stream without knowing about merges.
	MergedInto TenantID
	// MergedTenantIDs lists the tenants that were merged into this one.
	MergedTenantIDs []TenantID
}

// Exists reports whether the tenant has been created (has at least one event).
func (s TenantState) Exists() bool {
	return s.Name != "" && !s.Deleted
}

// IsMerged reports whether this tenant was merged into another one, in which
// case [TenantState.MergedInto] names the survivor.
func (s TenantState) IsMerged() bool {
	return !s.MergedInto.IsZero()
}

// IsActive reports whether the tenant exists and is not suspended or deleted.
func (s TenantState) IsActive() bool {
	return s.Exists() && !s.Suspended
}

// NormalizeTenantName folds a tenant name to its comparison form: trimmed and
// lowercased. Two tenants whose names differ only in case or surrounding
// whitespace are the same name to a human, which is exactly how duplicate
// tenants come to exist (issue #47); the normalized form is what the
// opt-in uniqueness check and the duplicates report compare.
func NormalizeTenantName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

// IsValid enforces struct-level invariants.
func (s TenantState) IsValid() bool {
	if s.Deleted && s.Suspended {
		return false
	}
	if s.IsMerged() && !s.Deleted {
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
//
// The switch is dispatch only: one row per event, handed to the helper that
// owns that transition (the apply* helpers at the bottom of this file). A
// helper answers a payload that does not decode with the state it was given,
// and FoldUser answers every error with the caller's original state. An event
// type outside the User stream's vocabulary is a rejection, never a no-op.
// cqrs-lint:ignore(B005) canonical fold pattern: switch is the standard ES fold
func FoldUser(
	state UserState,
	evt event.Event,
) (UserState, error) {
	next := state

	var err error

	switch evt.Type() {
	// Who the user is: the identity fields, from registration to the tombstone.
	case EventUserRegistered:
		next, err = applyUserRegistered(next, evt)
	case EventEmailChanged:
		next, err = applyEmailChanged(next, evt)
	case EventEmailVerified:
		next, err = applyEmailVerified(next, evt)
	case EventDisplayNameChanged:
		next, err = applyDisplayNameChanged(next, evt)
	case EventUserDeleted:
		next, err = applyUserDeleted(next, evt)

	// Merge of duplicate identities: the survivor's half and the loser's.
	case EventUserMerged:
		next, err = applyUserMerged(next, evt)
	case EventUserMergedInto:
		next, err = applyUserMergedInto(next, evt)

	// Legacy (see constants.go): no longer emitted, decoded for backward compat.
	// Its payload is decoded for the error alone; it never changed UserState.
	case EventRolesUpdated:
		_, err = UnmarshalPayload[RolesUpdatedPayload](evt)

	// Credentials: the user's WebAuthn authenticators.
	case EventCredentialAdded:
		next, err = applyCredentialAdded(next, evt)
	case EventCredentialRemoved:
		next, err = applyCredentialRemoved(next, evt)
	case EventCredentialUsed:
		next, err = applyCredentialUsed(next, evt)

	// Sign-in factors beyond credentials: TOTP and linked external identities.
	case EventTOTPEnabled:
		next, err = applyTOTPEnabled(next, evt)
	case EventTOTPDisabled:
		next.TOTPEnabled = false
		next.TOTPSecret = nil
	case EventExternalAccountLinked:
		next, err = applyExternalAccountLinked(next, evt)
	case EventExternalAccountUnlinked:
		next, err = applyExternalAccountUnlinked(next, evt)

	default:
		return state, errorfamily.NewRejection(
			"usermgmt.user.unknown_event",
			"FoldUser received unknown event type: "+string(evt.Type()),
		)
	}

	if err != nil {
		return state, err
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
		// A fresh incarnation inherits nothing from the previous one: a
		// recreated tenant must not keep the old reasons (IsValid would
		// reject them) nor the tenants the old incarnation absorbed.
		next = TenantState{Name: p.Name, DisplayName: p.DisplayName}

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

	case EventTenantRenamed:
		p, err := UnmarshalPayload[TenantRenamedPayload](evt)
		if err != nil {
			return state, err
		}
		next.DisplayName = p.DisplayName

	case EventTenantMerged:
		p, err := UnmarshalPayload[TenantMergedPayload](evt)
		if err != nil {
			return state, err
		}
		for _, loser := range p.LoserIDs {
			next.MergedTenantIDs = withAppendedUnique(next.MergedTenantIDs, loser)
		}

	case EventTenantMergedInto:
		p, err := UnmarshalPayload[TenantMergedIntoPayload](evt)
		if err != nil {
			return state, err
		}
		next.Deleted = true
		next.Suspended = false
		next.SuspendReason = ""
		next.DeleteReason = p.Reason
		next.MergedInto = p.SurvivorID

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

	case EventBotTransferred:
		p, err := UnmarshalPayload[BotTransferredPayload](evt)
		if err != nil {
			return state, err
		}
		next.OwnerID = p.OwnerID

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

// withAppendedUnique returns ids plus value when value is absent. The result
// is a fresh slice: fold functions start from a shallow copy of the caller's
// state, so an in-place append could write through to the state the caller
// still holds.
func withAppendedUnique[T comparable](ids []T, value T) []T {
	if slices.Contains(ids, value) {
		return ids
	}
	return append(slices.Clone(ids), value)
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

// applyCredentialAdded records a WebAuthn authenticator with the sign counter
// the event carried. Its later life belongs to applyCredentialUsed and
// applyCredentialRemoved.
func applyCredentialAdded(state UserState, evt event.Event) (UserState, error) {
	p, err := UnmarshalPayload[CredentialAddedPayload](evt)
	if err != nil {
		return state, err
	}

	state.Credentials = append(state.Credentials, NewCredentialFromPayload(p, evt.OccurredAt()))

	return state, nil
}

// applyUserRegistered starts a fresh incarnation of the stream. The whole
// state is replaced, so nothing from a previous incarnation (credentials,
// sign-in factors, a tombstone, merge bookkeeping) survives it.
func applyUserRegistered(state UserState, evt event.Event) (UserState, error) {
	p, err := UnmarshalPayload[UserRegisteredPayload](evt)
	if err != nil {
		return state, err
	}

	return UserState{Email: p.Email, DisplayName: p.DisplayName}, nil
}

// applyEmailChanged moves the account to a new address and drops what was
// attached to the old one: the new address starts out unverified and TOTP is
// switched off, secret included.
func applyEmailChanged(state UserState, evt event.Event) (UserState, error) {
	p, err := UnmarshalPayload[EmailChangedPayload](evt)
	if err != nil {
		return state, err
	}

	state.Email = p.Email
	state.EmailVerified = false
	state.TOTPEnabled = false
	state.TOTPSecret = nil

	return state, nil
}

// applyEmailVerified marks the address as verified. The payload is decoded
// for its error alone; the transition is carried by the event type.
func applyEmailVerified(state UserState, evt event.Event) (UserState, error) {
	_, err := UnmarshalPayload[EmailVerifiedPayload](evt)
	if err != nil {
		return state, err
	}

	state.EmailVerified = true

	return state, nil
}

// applyDisplayNameChanged sets the name other people see. The login identity
// (the email) is untouched.
func applyDisplayNameChanged(state UserState, evt event.Event) (UserState, error) {
	p, err := UnmarshalPayload[DisplayNameChangedPayload](evt)
	if err != nil {
		return state, err
	}

	state.DisplayName = p.DisplayName

	return state, nil
}

// applyUserDeleted raises the tombstone. The stream keeps its history
// (Exists stays true); IsActive is what turns false.
func applyUserDeleted(state UserState, evt event.Event) (UserState, error) {
	p, err := UnmarshalPayload[UserDeletedPayload](evt)
	if err != nil {
		return state, err
	}

	state.Deleted = true
	state.DeleteReason = p.Reason

	return state, nil
}

// applyUserMerged runs on the SURVIVOR of a merge: each duplicate folded into
// this user is recorded once in MergedUserIDs.
func applyUserMerged(state UserState, evt event.Event) (UserState, error) {
	p, err := UnmarshalPayload[UserMergedPayload](evt)
	if err != nil {
		return state, err
	}

	for _, loser := range p.LoserIDs {
		state.MergedUserIDs = withAppendedUnique(state.MergedUserIDs, loser)
	}

	return state, nil
}

// applyUserMergedInto is the terminal event of a merged-away duplicate. It is
// a tombstone as well (Deleted=true), so every liveness gate that predates
// merges refuses the stream; MergedInto names the survivor to follow.
func applyUserMergedInto(state UserState, evt event.Event) (UserState, error) {
	p, err := UnmarshalPayload[UserMergedIntoPayload](evt)
	if err != nil {
		return state, err
	}

	state.Deleted = true
	state.MergedInto = p.SurvivorID
	state.DeleteReason = p.Reason

	return state, nil
}

// applyTOTPEnabled switches the second factor on and adopts the secret the
// event carries. The secret is cloned: the payload's slice belongs to the
// decoded event, and the state must not alias it.
func applyTOTPEnabled(state UserState, evt event.Event) (UserState, error) {
	p, err := UnmarshalPayload[TOTPEnabledPayload](evt)
	if err != nil {
		return state, err
	}

	state.TOTPEnabled = true
	state.TOTPSecret = slices.Clone(p.Secret)

	return state, nil
}

// applyExternalAccountLinked adds a linked third-party identity (provider +
// subject). LinkedAt is the event's own timestamp, not a payload field.
func applyExternalAccountLinked(state UserState, evt event.Event) (UserState, error) {
	p, err := UnmarshalPayload[ExternalAccountLinkedPayload](evt)
	if err != nil {
		return state, err
	}

	state.ExternalAccounts = append(state.ExternalAccounts, ExternalAccount{
		Provider:    p.Provider,
		Subject:     p.Subject,
		Email:       p.Email,
		DisplayName: p.DisplayName,
		LinkedAt:    evt.OccurredAt(),
	})

	return state, nil
}

// applyExternalAccountUnlinked drops every linked identity matching the
// event's provider + subject pair; a pair that was never linked is not an
// error. The survivors move to a fresh slice so the fold never writes through
// to the caller's state.
func applyExternalAccountUnlinked(state UserState, evt event.Event) (UserState, error) {
	p, err := UnmarshalPayload[ExternalAccountUnlinkedPayload](evt)
	if err != nil {
		return state, err
	}

	filtered := make([]ExternalAccount, 0, len(state.ExternalAccounts))
	for _, ea := range state.ExternalAccounts {
		if ea.Provider != p.Provider || ea.Subject != p.Subject {
			filtered = append(filtered, ea)
		}
	}

	state.ExternalAccounts = filtered

	return state, nil
}
