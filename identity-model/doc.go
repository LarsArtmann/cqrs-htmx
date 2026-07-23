// Package identitymodel contains the pure domain model types for event-sourced
// identity management: Users, Tenants/Orgs, Sessions, Auth, and RBAC.
//
// This module is dependency-free from infrastructure concerns — no HTTP
// frameworks, no SQL drivers, no Casbin. It defines the WHAT (types, events,
// commands, states), not the HOW (storage, transport, enforcement).
//
// The module is designed to be imported by implementation packages (like
// usermgmt) which provide the infrastructure: SQL event stores, HTTP handlers,
// Casbin authorization, read-model projections, etc.
package identitymodel
