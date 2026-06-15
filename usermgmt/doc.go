// Package usermgmt provides passwordless user management using event-sourced CQRS.
//
// Users register with just an email address (no password). Authentication is done
// via WebAuthn (Passkeys / FIDO2). All user state changes are persisted as events
// to an event store, with read models rebuilt from projections.
//
// # Quick Start
//
//	svc, _ := usermgmt.NewService(usermgmt.ServiceConfig{
//	    WebAuthnConfig: &usermgmt.WebAuthnConfig{
//	        RPID:          "example.com",
//	        RPDisplayName: "My App",
//	        RPOrigins:     []string{"https://example.com"},
//	    },
//	})
//
// # Registration Flow
//
// 1. Register a user account (email only):
//
//	resp, _ := svc.Register(ctx, usermgmt.RegisterRequest{
//	    ID:    usermgmt.NewUserID(ulid),
//	    Email: "alice@example.com",
//	})
//
// 2. Begin WebAuthn credential registration (sends challenge to browser):
//
//	beginResp, _ := svc.BeginRegistration(ctx, resp.User.ID)
//
// 3. Browser creates credential via navigator.credentials.create()
//
// 4. Finish registration (verifies attestation, persists credential):
//
//	_ = svc.FinishRegistration(ctx, resp.User.ID, httpRequest, "My Passkey")
//
// # Login Flow
//
// 1. Begin login (sends challenge to browser):
//
//	beginResp, _ := svc.BeginLogin(ctx, "alice@example.com")
//
// 2. Browser asserts credential via navigator.credentials.get()
//
// 3. Finish login (verifies assertion, creates session):
//
//	loginResp, _ := svc.FinishLogin(ctx, userID, httpRequest)
//
// # Architecture
//
// All state changes go through the event store:
//
//	Service → CommandDispatcher → DeciderRepository.Execute
//	  → Load → Fold → Decide → Save → Publish
//	  → UserReadModel projection (query side)
//	  → CasbinProjection (authorization)
//
// Sessions, lockout, and WebAuthn challenge state are ephemeral (not event-sourced).
package usermgmt
