# ADR 0019: usermgmt God-Package Decomposition — Blocked by Method Receiver Constraint

**Date:** 2026-06-22
**Status:** Blocked (requires architectural redesign)

## Context

The `usermgmt` package has 152 files with 60 methods on `*Service`. The plan
(E10) calls for decomposing into 8 sub-packages (webauthn, oauth2, totp, store,
authz, membership, tenant, bot).

## Problem

Go requires methods to be defined in the **same package** as the receiver type.
All 60 methods are defined as `func (s *Service) ...` in the root `usermgmt`
package. Moving implementation files (e.g., `webauthn_service.go`) to a
`webauthn/` sub-package would change the package to `webauthn`, making
`func (s *Service) ...` illegal — `Service` is defined in `usermgmt`, not
`webauthn`.

## What's Needed

The decomposition requires **redesigning Service to use composition**:

1. Extract domain-specific service types:

   ```go
   // webauthn/service.go
   type Service struct {
       readModel  *usermgmt.UserReadModel
       dispatcher command.Dispatcher
       webauthn   *webauthn.WebAuthn
       // ...
   }
   func (s *Service) BeginRegistration(...) (*Response, error) { ... }
   ```

2. Root Service composes sub-services:

   ```go
   // usermgmt/service_core.go
   type Service struct {
       WebAuthn *webauthn.Service
       OAuth2   *oauth2.Service
       // ...
   }
   ```

3. Backward-compatible delegation (optional):
   ```go
   func (s *Service) BeginRegistration(...) (*Response, error) {
       return s.WebAuthn.BeginRegistration(...)
   }
   ```

This is a multi-day architectural refactoring, not a mechanical file move.

## Decision

**Defer E10.** The god-package is internal hygiene with no consumer-facing impact
(all consumers import from `usermgmt` root regardless). The P0–P2 boundary fixes
(E02–E09) are complete and provide the real ecosystem health improvements.

## Future Path

When undertaking E10:

1. Start with the most self-contained domain (TOTP — 6 methods, minimal deps)
2. Extract `totp.Service` with its own dependencies
3. Root `Service` embeds `*totp.Service` as `TOTP` field
4. Keep backward-compatible wrapper methods during transition
5. Repeat for webauthn, oauth2, authz, membership, tenant, bot
