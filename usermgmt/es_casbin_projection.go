package usermgmt

import (
	"context"
	"errors"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/codec/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2"
)

// CasbinProjection derives Casbin policies from User events.
// It subscribes to UserRegistered, RolesUpdated, and UserDeleted events
// and maintains the group policy set so that authorization checks reflect
// the current event-derived state.
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

func (p *CasbinProjection) Name() string  { return "casbin-projection" }

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
		for _, role := range d.Roles {
			if err := p.authz.AddGroupPolicy(GroupPolicy{
				Subject: subject, Role: role, Domain: domain,
			}); err != nil {
				return fmt.Errorf("add group policy on roles update: %w", err)
			}
		}

	case eventUserDeleted:
		if err := p.authz.RemoveAllRolesForUser(subject); err != nil {
			return fmt.Errorf("delete user from casbin: %w", err)
		}

	default:
	}

	return nil
}

var _ event.Projection = (*CasbinProjection)(nil)
