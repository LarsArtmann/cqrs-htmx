package usermgmt

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
)

// CasbinProjection derives Casbin policies from User events.
// It subscribes to UserRegistered, RolesUpdated, UserDeleted, CredentialAdded,
// CredentialRemoved, ExternalAccountLinked, and ExternalAccountUnlinked events
// and maintains the group policy set so that authorization checks reflect the
// current event-derived state.
//
// Credential and external account events are subscribed for projection ordering
// guarantees (they arrive after any preceding policy-changing events in the same
// stream). The projection is a no-op for these events since they do not affect
// Casbin policies.
type CasbinProjection struct {
	authz *Authz
}

// NewCasbinProjection creates a CasbinProjection that wraps the given Authz.
func NewCasbinProjection(authz *Authz) (*CasbinProjection, error) {
	if authz == nil {
		return nil, event.NewInfrastructure("usermgmt.casbin_projection.nil_authz", "NewCasbinProjection: authz is nil")
	}
	return &CasbinProjection{authz: authz}, nil
}

func (p *CasbinProjection) Name() string { return "casbin-projection" }

func (p *CasbinProjection) EventTypes() []event.Type {
	return []event.Type{
		eventUserRegistered,
		eventRolesUpdated,
		eventUserDeleted,
		eventCredentialAdded,
		eventCredentialRemoved,
		eventExternalAccountLinked,
		eventExternalAccountUnlinked,
		eventMemberAdded,
		eventMemberRolesChanged,
		eventMemberRemoved,
		eventTenantDeleted,
		eventBotDeleted,
	}
}

func (p *CasbinProjection) Handle(_ context.Context, evt event.Event) error {
	subject := evt.AggregateID().String()

	switch evt.Type() {
	case eventUserRegistered:
		d, err := unmarshalPayload[UserRegisteredPayload](evt)
		if err != nil {
			return event.WrapCorruption(
				err,
				"usermgmt.casbin_projection.decode_failed",
				"decode UserRegistered in casbin projection",
			)
		}
		return p.addRolesFor(subject, subject, d.Roles, "on register")

	case eventRolesUpdated:
		d, err := unmarshalPayload[RolesUpdatedPayload](evt)
		if err != nil {
			return event.WrapCorruption(
				err,
				"usermgmt.casbin_projection.decode_failed",
				"decode RolesUpdated in casbin projection",
			)
		}
		domain := d.Domain
		if domain == "" {
			domain = subject
		}
		if err := p.authz.RemoveAllRolesInDomain(subject, domain); err != nil {
			return event.Wrapf(
				err,
				event.Infrastructure,
				"usermgmt.casbin_projection.remove_roles_failed",
				"remove old roles for %s in %s",
				subject, domain,
			)
		}
		return p.addRolesFor(subject, domain, d.Roles, "on roles update")

	case eventUserDeleted:
		if err := p.authz.RemoveAllRolesForUser(subject); err != nil {
			return event.WrapInfrastructure(err, "usermgmt.casbin_projection.delete_failed", "delete user from casbin")
		}

	case eventCredentialAdded, eventCredentialRemoved,
		eventExternalAccountLinked, eventExternalAccountUnlinked:
		// Credentials and external accounts don't affect Casbin policies
		// — subscribed for projection ordering.

	case eventMemberAdded, eventMemberRolesChanged, eventMemberRemoved:
		return p.handleMembershipEvent(evt)

	case eventTenantDeleted:
		// Remove all Casbin policies in the tenant's domain (the tenant's aggregate ID).
		// This cascades role cleanup for all members of the deleted tenant.
		if err := p.authz.RemoveAllRolesForUser(subject); err != nil {
			return event.WrapInfrastructure(
				err, "usermgmt.casbin_projection.tenant_delete_failed",
				"remove all policies for deleted tenant "+subject,
			)
		}

	case eventBotDeleted:
		// Remove all Casbin policies for the deleted bot.
		if err := p.authz.RemoveAllRolesForUser(subject); err != nil {
			return event.WrapInfrastructure(
				err, "usermgmt.casbin_projection.bot_delete_failed",
				"remove all policies for deleted bot "+subject,
			)
		}

	default:
	}

	return nil
}

// addRolesFor grants each of the given roles to subject under domain.
// errContext is included in the wrapped error message (e.g. "on register").
func (p *CasbinProjection) addRolesFor(subject, domain string, roles []Role, errContext string) error {
	for _, role := range roles {
		if err := p.authz.AddGroupPolicy(GroupPolicy{
			Subject: subject, Role: role, Domain: domain,
		}); err != nil {
			return event.Wrapf(
				err,
				event.Infrastructure,
				"usermgmt.casbin_projection.add_policy_failed",
				"add group policy %s",
				errContext,
			)
		}
	}
	return nil
}

var _ event.Projection = (*CasbinProjection)(nil)

// handleMembershipEvent processes MemberAdded, MemberRolesChanged, and
// MemberRemoved events. Extracted from Handle to reduce cognitive complexity.
func (p *CasbinProjection) handleMembershipEvent(evt event.Event) error {
	switch evt.Type() {
	case eventMemberAdded:
		d, err := unmarshalPayload[MemberAddedPayload](evt)
		if err != nil {
			return event.WrapCorruption(
				err,
				"usermgmt.casbin_projection.decode_failed",
				"decode MemberAdded in casbin projection",
			)
		}
		return p.addRolesFor(d.ActorID, d.TenantID, d.Roles, "on member added")

	case eventMemberRolesChanged:
		d, err := unmarshalPayload[MemberRolesChangedPayload](evt)
		if err != nil {
			return event.WrapCorruption(
				err,
				"usermgmt.casbin_projection.decode_failed",
				"decode MemberRolesChanged in casbin projection",
			)
		}
		if err := p.removeAllRolesInDomain(d.ActorID, d.TenantID, "on member roles changed"); err != nil {
			return err
		}
		return p.addRolesFor(d.ActorID, d.TenantID, d.Roles, "on member roles changed")

	case eventMemberRemoved:
		d, err := unmarshalPayload[MemberRemovedPayload](evt)
		if err != nil {
			return event.WrapCorruption(
				err,
				"usermgmt.casbin_projection.decode_failed",
				"decode MemberRemoved in casbin projection",
			)
		}
		return p.removeAllRolesInDomain(d.ActorID, d.TenantID, "on member removed")

	default:
		return nil
	}
}

// removeAllRolesInDomain removes all Casbin group policies for a subject
// within a specific domain. Delegates to Authz.RemoveAllRolesInDomain.
// Used by MemberRolesChanged and MemberRemoved.
func (p *CasbinProjection) removeAllRolesInDomain(subject, domain, errContext string) error {
	if err := p.authz.RemoveAllRolesInDomain(subject, domain); err != nil {
		return event.Wrapf(
			err, event.Infrastructure,
			"usermgmt.casbin_projection.remove_roles_failed",
			"remove all roles for %s in %s (%s)",
			subject, domain, errContext,
		)
	}
	return nil
}
