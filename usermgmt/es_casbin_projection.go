package usermgmt

import (
	"context"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/codec/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2"
)

// CasbinProjection derives Casbin policies from User events.
// It subscribes to UserRegistered, RolesUpdated, and UserDeleted events
// and maintains the group policy set so that authorization checks reflect
// the current event-derived state.
type CasbinProjection struct {
	authz       *Authz
	newAuthz    func(...EnforcerConfig) (*Authz, error)
	hasOwnAuthz bool
}

// NewCasbinProjection creates a CasbinProjection that wraps the given Authz.
// The Authz instance is used directly — callers must ensure it's initialized.
func NewCasbinProjection(authz *Authz) (*CasbinProjection, error) {
	if authz == nil {
		return nil, fmt.Errorf("NewCasbinProjection: authz is nil")
	}
	return &CasbinProjection{authz: authz, hasOwnAuthz: false}, nil
}

// NewCasbinProjectionWithNewAuthz creates a CasbinProjection that creates its
// own Authz instance using the provided constructor. This is useful when the
// projection should own a dedicated enforcer built from events.
func NewCasbinProjectionWithNewAuthz(
	newAuthzFn func(...EnforcerConfig) (*Authz, error),
) (*CasbinProjection, error) {
	authz, err := newAuthzFn()
	if err != nil {
		return nil, fmt.Errorf("create authz for casbin projection: %w", err)
	}
	return &CasbinProjection{authz: authz, newAuthz: newAuthzFn, hasOwnAuthz: true}, nil
}

// Authz returns the underlying Authz instance for direct policy queries.
func (p *CasbinProjection) Authz() *Authz { return p.authz }

func (p *CasbinProjection) Name() string { return "casbin-projection" }

func (p *CasbinProjection) EventTypes() []event.Type {
	return []event.Type{eventUserRegistered, eventRolesUpdated, eventUserDeleted}
}

func (p *CasbinProjection) Handle(_ context.Context, evt event.Event) error {
	c := codec.JSONCodec{}
	subject := evt.AggregateID().String()

	switch evt.Type() {
	case eventUserRegistered:
		d, err := event.DecodePayload[UserRegisteredPayload](evt, c)
		if err != nil {
			return fmt.Errorf("decode UserRegistered in casbin projection: %w", err)
		}
		domain := subject
		for _, role := range d.Roles {
			if err := p.authz.AddGroupPolicy(GroupPolicy{
				Subject: subject, Role: role, Domain: domain,
			}); err != nil {
				return fmt.Errorf("add group policy on register: %w", err)
			}
		}

	case eventRolesUpdated:
		d, err := event.DecodePayload[RolesUpdatedPayload](evt, c)
		if err != nil {
			return fmt.Errorf("decode RolesUpdated in casbin projection: %w", err)
		}
		domain := d.Domain
		if domain == "" {
			domain = subject
		}
		roles, err := p.authz.enforcer.GetRolesForUser(subject, domain)
		if err != nil {
			return fmt.Errorf("get current roles for %s: %w", subject, err)
		}
		for _, r := range roles {
			if _, err := p.authz.enforcer.RemoveGroupingPolicy(subject, r, domain); err != nil {
				return fmt.Errorf("remove old role %s for %s: %w", r, subject, err)
			}
		}
		for _, role := range d.Roles {
			if err := p.authz.AddGroupPolicy(GroupPolicy{
				Subject: subject, Role: role, Domain: domain,
			}); err != nil {
				return fmt.Errorf("add group policy on roles update: %w", err)
			}
		}

	case eventUserDeleted:
		_, err := p.authz.enforcer.DeleteUser(subject)
		if err != nil {
			return fmt.Errorf("delete user from casbin: %w", err)
		}

	default:
	}

	return nil
}

var _ event.Projection = (*CasbinProjection)(nil)
