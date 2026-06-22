package usermgmt

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
)

// CreateTenantRequest holds the parameters for creating a new tenant.
type CreateTenantRequest struct {
	ID          TenantID `json:"id"`
	Name        string   `json:"name"`
	DisplayName string   `json:"display_name"`
}

// CreateTenant creates a new tenant and dispatches the TenantCreated event.
func (s *Service) CreateTenant(ctx context.Context, req CreateTenantRequest) (*Tenant, error) {
	if req.ID.IsZero() {
		return nil, event.NewRejection("usermgmt.tenant.id_required", "tenant ID is required")
	}
	if req.Name == "" {
		return nil, event.NewRejection("usermgmt.tenant.name_required", "tenant name is required")
	}

	aggID, err := aggIDFromTenant(req.ID)
	if err != nil {
		return nil, event.WrapInfrastructure(err, "usermgmt.tenant.id_conversion_failed", "convert tenant ID")
	}

	err = s.dispatcher.Dispatch(ctx, NewCreateTenantCmd(
		aggID, req.Name, req.DisplayName,
	))
	if err != nil {
		return nil, err //nolint:wrapcheck // decider returns typed domain errors
	}

	tenant, ok := s.tenantReadModel.FindByID(aggID)
	if !ok {
		return nil, event.NewTransient("internal", "tenant not in read model after create")
	}
	return tenant, nil
}

// SuspendTenant suspends a tenant by dispatching the TenantSuspended event.
func (s *Service) SuspendTenant(ctx context.Context, tenantID TenantID, reason string) error {
	aggID, err := aggIDFromTenant(tenantID)
	if err != nil {
		return event.WrapInfrastructure(err, "usermgmt.tenant.id_conversion_failed", "convert tenant ID")
	}
	return s.dispatcher.Dispatch( //nolint:wrapcheck // decider returns typed domain errors
		ctx,
		NewSuspendTenantCmd(aggID, reason),
	)
}

// ReactivateTenant restores a suspended tenant.
func (s *Service) ReactivateTenant(ctx context.Context, tenantID TenantID) error {
	aggID, err := aggIDFromTenant(tenantID)
	if err != nil {
		return event.WrapInfrastructure(err, "usermgmt.tenant.id_conversion_failed", "convert tenant ID")
	}
	return s.dispatcher.Dispatch( //nolint:wrapcheck // decider returns typed domain errors
		ctx,
		NewReactivateTenantCmd(aggID),
	)
}

// DeleteTenant permanently deletes a tenant and cleans up all memberships
// associated with it. The CasbinProjection handles policy removal automatically
// via the TenantDeleted event.
func (s *Service) DeleteTenant(ctx context.Context, tenantID TenantID, reason string) error {
	aggID, err := aggIDFromTenant(tenantID)
	if err != nil {
		return event.WrapInfrastructure(err, "usermgmt.tenant.id_conversion_failed", "convert tenant ID")
	}
	if err := s.dispatcher.Dispatch(
		ctx,
		NewDeleteTenantCmd(aggID, reason),
	); err != nil {
		return err //nolint:wrapcheck // decider returns typed domain errors
	}
	// Best-effort membership cleanup: remove all members from the deleted tenant.
	s.revokeMembershipsForTenantBestEffort(ctx, tenantID)
	return nil
}

// revokeMembershipsForTenantBestEffort removes all memberships for a tenant.
// Errors are logged but not returned — the tenant is already deleted.
func (s *Service) revokeMembershipsForTenantBestEffort(ctx context.Context, tenantID TenantID) {
	memberships := s.membershipReadModel.FindByTenant(tenantID.Get())
	for _, mem := range memberships {
		removalCmd := NewRemoveMemberCmd(mem.ActorID, tenantID)
		if err := s.dispatcher.Dispatch(ctx, removalCmd); err != nil {
			s.logger.Warn(
				"usermgmt: failed to remove membership on tenant deletion",
				"tenant_id", tenantID.Get(),
				"actor_id", mem.ActorID.String(),
				"error", err,
			)
		}
	}
}

// GetTenant retrieves a tenant by ID from the read model.
func (s *Service) GetTenant(_ context.Context, id TenantID) (*Tenant, error) {
	aggID, err := aggIDFromTenant(id)
	if err != nil {
		return nil, event.WrapInfrastructure(err, "usermgmt.tenant.id_conversion_failed", "convert tenant ID")
	}
	tenant, ok := s.tenantReadModel.FindByID(aggID)
	if !ok {
		return nil, event.NewRejection("usermgmt.tenant.not_found", "tenant not found")
	}
	return tenant, nil
}

// AllTenants returns all active tenants.
func (s *Service) AllTenants() []*Tenant {
	return s.tenantReadModel.All()
}

func aggIDFromTenant(tenantID TenantID) (id.AggregateID, error) {
	return aggIDFromBranded(tenantID.Get(), "usermgmt.invalid_tenant_id")
}
