package identitymodel

import (
	"bytes"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
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
func (s UserState) Exists() bool {
	return s.Email != ""
}

// MembershipState is the aggregate state for a Membership (actor+tenant pair).
type MembershipState struct {
	ActorID  ActorID
	TenantID TenantID
	Roles    []Role
	Removed  bool
}

// Exists reports whether the membership has been added (has at least one event).
func (s MembershipState) Exists() bool {
	return !s.ActorID.IsZero() && !s.Removed
}

// HasRole reports whether the membership grants the given role.
func (s MembershipState) HasRole(role Role) bool {
	for _, r := range s.Roles {
		if r == role {
			return true
		}
	}
	return false
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

// Exists reports whether the bot has been registered (has at least one event).
func (s BotState) Exists() bool {
	return s.Name != "" && !s.Deleted
}

// FoldUser applies an event to the current UserState, returning the new state.
func FoldUser(state UserState, evt event.Event) (UserState, error) {
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
		p, err := UnmarshalPayload[CredentialRemovedPayload](evt)
		if err != nil {
			return state, err
		}
		filtered := make([]WebAuthnCredential, 0, len(next.Credentials))
		for _, c := range next.Credentials {
			if !bytes.Equal(c.ID, p.ID) {
				filtered = append(filtered, c)
			}
		}
		next.Credentials = filtered

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
		next.TOTPSecret = p.Secret

	case EventTOTPDisabled:
		next.TOTPEnabled = false
		next.TOTPSecret = nil

	case EventExternalAccountLinked:
		p, err := UnmarshalPayload[ExternalAccountLinkedPayload](evt)
		if err != nil {
			return state, err
		}
		next.ExternalAccounts = append(next.ExternalAccounts, ExternalAccount{
			ExternalAccountCore: ExternalAccountCore{
				Provider:    p.Provider,
				Subject:     p.Subject,
				Email:       p.Email,
				DisplayName: p.DisplayName,
			},
			LinkedAt: evt.OccurredAt(),
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
func FoldMembership(state MembershipState, evt event.Event) (MembershipState, error) {
	next := state

	switch evt.Type() {
	case EventMemberAdded:
		p, err := UnmarshalPayload[MemberAddedPayload](evt)
		if err != nil {
			return state, err
		}
		roles := make([]Role, len(p.Roles))
		copy(roles, p.Roles)
		kind, err := ActorKindFromString(p.ActorKind)
		if err != nil {
			return state, err
		}
		next.ActorID = NewActorID(kind, p.ActorID)
		next.TenantID = NewTenantID(p.TenantID)
		next.Roles = roles
		next.Removed = false

	case EventMemberRolesChanged:
		p, err := UnmarshalPayload[MemberRolesChangedPayload](evt)
		if err != nil {
			return state, err
		}
		roles := make([]Role, len(p.Roles))
		copy(roles, p.Roles)
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
func FoldTenant(state TenantState, evt event.Event) (TenantState, error) {
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
func FoldBot(state BotState, evt event.Event) (BotState, error) {
	next := state

	switch evt.Type() {
	case EventBotRegistered:
		p, err := UnmarshalPayload[BotRegisteredPayload](evt)
		if err != nil {
			return state, err
		}
		scopes := make([]string, len(p.Scopes))
		copy(scopes, p.Scopes)
		next.Name = p.Name
		next.OwnerID = p.OwnerID
		next.TokenHash = p.TokenHash
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
		return ActorUser, nil
	case ActorKindBotStr:
		return ActorBot, nil
	default:
		return ActorUser, errorfamily.NewRejection(
			"usermgmt.membership.unknown_actor_kind",
			"unknown actor kind: "+s,
		)
	}
}
