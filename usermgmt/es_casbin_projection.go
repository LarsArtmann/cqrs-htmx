package usermgmt

import (
	"context"
	"errors"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
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
		return nil, errors.New("NewCasbinProjection: authz is nil")
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
	}
}

func (p *CasbinProjection) Handle(_ context.Context, evt event.Event) error {
	subject := evt.AggregateID().String()

	switch evt.Type() {
	case eventUserRegistered:
		d, err := unmarshalPayload[UserRegisteredPayload](evt)
		if err != nil {
			return fmt.Errorf("decode UserRegistered in casbin projection: %w", err)
		}
		return p.addRolesFor(subject, subject, d.Roles, "on register")

	case eventRolesUpdated:
		d, err := unmarshalPayload[RolesUpdatedPayload](evt)
		if err != nil {
			return fmt.Errorf("decode RolesUpdated in casbin projection: %w", err)
		}
		domain := d.Domain
		if domain == "" {
			domain = subject
		}
		currentRoles, err := p.authz.RolesForUser(NewUserID(subject), domain)
		if err != nil {
			return fmt.Errorf("get current roles for %s: %w", subject, err)
		}
		for _, r := range currentRoles {
			if err := p.authz.RemoveGroupPolicy(GroupPolicy{
				Subject: subject, Role: r, Domain: domain,
			}); err != nil {
				return fmt.Errorf("remove old role %s for %s: %w", r, subject, err)
			}
		}
		return p.addRolesFor(subject, domain, d.Roles, "on roles update")

	case eventUserDeleted:
		if err := p.authz.RemoveAllRolesForUser(subject); err != nil {
			return fmt.Errorf("delete user from casbin: %w", err)
		}

	case eventCredentialAdded, eventCredentialRemoved,
		eventExternalAccountLinked, eventExternalAccountUnlinked:
		// Credentials and external accounts don't affect Casbin policies
		// — subscribed for projection ordering.

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
			return fmt.Errorf("add group policy %s: %w", errContext, err)
		}
	}
	return nil
}

var _ event.Projection = (*CasbinProjection)(nil)
