package usermgmt

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/id/v3"
)

func TestService_CreateTenant_Success(t *testing.T) {
	t.Parallel()

	svc := newTestService(t)
	t.Cleanup(svc.Stop)
	ctx := context.Background()

	tenantID := NewTenantID(id.NewAggregateID().String())
	tenant, err := svc.CreateTenant(ctx, CreateTenantRequest{
		ID:          tenantID,
		Name:        "acme-corp",
		DisplayName: "Acme Corporation",
	})
	if err != nil {
		t.Fatalf("CreateTenant: %v", err)
	}
	if tenant == nil {
		t.Fatal("CreateTenant returned nil tenant")
	}
	if tenant.Name != "acme-corp" {
		t.Errorf("Name = %q, want %q", tenant.Name, "acme-corp")
	}
	if tenant.DisplayName != "Acme Corporation" {
		t.Errorf("DisplayName = %q, want %q", tenant.DisplayName, "Acme Corporation")
	}
	if tenant.Suspended || tenant.Deleted {
		t.Error("new tenant should not be suspended or deleted")
	}

	// Verify via read model (read-your-writes)
	fetched, err := svc.GetTenant(ctx, tenantID)
	if err != nil {
		t.Fatalf("GetTenant after create: %v", err)
	}
	if fetched.Name != tenant.Name {
		t.Errorf("GetTenant Name = %q, want %q", fetched.Name, tenant.Name)
	}
}

func TestService_CreateTenant_EmptyID(t *testing.T) {
	t.Parallel()

	svc := newTestService(t)
	t.Cleanup(svc.Stop)

	_, err := svc.CreateTenant(context.Background(), CreateTenantRequest{
		Name:        "no-id",
		DisplayName: "No ID",
	})
	if err == nil {
		t.Fatal("expected error for empty tenant ID, got nil")
	}
}

func TestService_SuspendAndReactivateTenant(t *testing.T) {
	t.Parallel()

	svc := newTestService(t)
	t.Cleanup(svc.Stop)
	ctx := context.Background()

	tenantID := NewTenantID(id.NewAggregateID().String())
	_, err := svc.CreateTenant(ctx, CreateTenantRequest{
		ID:   tenantID,
		Name: "suspend-test",
	})
	if err != nil {
		t.Fatalf("CreateTenant: %v", err)
	}

	if err := svc.SuspendTenant(ctx, tenantID, "policy violation"); err != nil {
		t.Fatalf("SuspendTenant: %v", err)
	}

	suspended, err := svc.GetTenant(ctx, tenantID)
	if err != nil {
		t.Fatalf("GetTenant after suspend: %v", err)
	}
	if !suspended.Suspended {
		t.Error("tenant should be suspended")
	}

	if err := svc.ReactivateTenant(ctx, tenantID); err != nil {
		t.Fatalf("ReactivateTenant: %v", err)
	}

	reactivated, err := svc.GetTenant(ctx, tenantID)
	if err != nil {
		t.Fatalf("GetTenant after reactivate: %v", err)
	}
	if reactivated.Suspended {
		t.Error("tenant should not be suspended after reactivation")
	}
}

func TestService_DeleteTenant(t *testing.T) {
	t.Parallel()

	svc := newTestService(t)
	t.Cleanup(svc.Stop)
	ctx := context.Background()

	tenantID := NewTenantID(id.NewAggregateID().String())
	_, err := svc.CreateTenant(ctx, CreateTenantRequest{
		ID:   tenantID,
		Name: "delete-test",
	})
	if err != nil {
		t.Fatalf("CreateTenant: %v", err)
	}

	if err := svc.DeleteTenant(ctx, tenantID, "cleanup"); err != nil {
		t.Fatalf("DeleteTenant: %v", err)
	}

	// Deleted tenants are hard-removed from the read model (tombstone is in
	// the event journal, not the API-facing read model).
	_, err = svc.GetTenant(ctx, tenantID)
	if err == nil {
		t.Fatal("expected error getting deleted tenant, got nil")
	}
}

func TestService_AllTenants(t *testing.T) {
	t.Parallel()

	svc := newTestService(t)
	t.Cleanup(svc.Stop)
	ctx := context.Background()

	_, err := svc.CreateTenant(ctx, CreateTenantRequest{
		ID:   NewTenantID(id.NewAggregateID().String()),
		Name: "t1",
	})
	if err != nil {
		t.Fatalf("CreateTenant t1: %v", err)
	}
	_, err = svc.CreateTenant(ctx, CreateTenantRequest{
		ID:   NewTenantID(id.NewAggregateID().String()),
		Name: "t2",
	})
	if err != nil {
		t.Fatalf("CreateTenant t2: %v", err)
	}

	all := svc.AllTenants()
	if len(all) < 2 {
		t.Errorf("AllTenants returned %d, want at least 2", len(all))
	}
}
