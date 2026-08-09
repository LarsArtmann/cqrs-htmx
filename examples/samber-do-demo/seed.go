package main

import (
	"context"
	"log"

	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
)

// seed registers a demo user so the usermgmt.Service has data.
// Storage is in-memory, so every boot starts fresh.
func seed(ctx context.Context, svc *usermgmt.Service) {
	uid := usermgmt.SyntheticUserID("demo-user-001") //nolint:staticcheck // demo uses deprecated alias for simplicity
	_, err := svc.Register(ctx, usermgmt.RegisterRequest{
		ID:          uid,
		Email:       "demo@samber-do.dev",
		DisplayName: "Demo User",
	})
	if err != nil {
		log.Printf("seed register: %v", err)
	}
}
