package usermgmt

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
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

// DeleteTenant permanently deletes a tenant.
func (s *Service) DeleteTenant(ctx context.Context, tenantID TenantID, reason string) error {
	aggID, err := aggIDFromTenant(tenantID)
	if err != nil {
		return event.WrapInfrastructure(err, "usermgmt.tenant.id_conversion_failed", "convert tenant ID")
	}
	return s.dispatcher.Dispatch( //nolint:wrapcheck // decider returns typed domain errors
		ctx,
		NewDeleteTenantCmd(aggID, reason),
	)
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
