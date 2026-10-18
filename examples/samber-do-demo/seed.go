package main

import (
	"context"
	"log"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
)

// seed registers a demo user so the usermgmt.Service has data.
// Storage is in-memory, so every boot starts fresh.
func seed(ctx context.Context, svc *usermgmt.Service) {
	uid := identitymodel.SyntheticUserID("demo-user-001")
	_, err := svc.Register(ctx, usermgmt.RegisterRequest{
		ID:          uid,
		Email:       "demo@samber-do.dev",
		DisplayName: "Demo User",
	})
	if err != nil {
		log.Printf("seed register: %v", err)
	}
}
