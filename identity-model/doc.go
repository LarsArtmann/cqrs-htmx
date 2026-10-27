// Package identitymodel contains the pure domain model types for event-sourced
// identity management: Users, Tenants/Orgs, Sessions, Auth, and RBAC.
//
// This module has no HTTP or SQL dependencies — it defines the WHAT (types,
// events, commands, states, and the Casbin-backed Authz enforcement engine),
// not the HOW (storage, transport). Casbin IS a first-class dependency here:
// the authorization model, default policies, and role hierarchy are domain
// concepts and live in this package; only the enforcer's POLICY STORAGE is
// injected by consumers.
//
// The module is designed to be imported by implementation packages (like
// usermgmt) which provide the infrastructure: SQL event stores, HTTP handlers,
// Casbin authorization, read-model projections, etc.
//
// # Fold aliasing invariant
//
// Fold payload slices alias the raw event bytes. Folded state must never
// retain a payload slice directly: every slice taken from a payload is
// defensively copied at the fold site (slices.Clone).
package identitymodel
