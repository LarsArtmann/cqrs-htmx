package usermgmt

import (
	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
)

// The store interfaces are defined once in identity-model (the domain source of
// truth) and re-exported here as type aliases so consumers that import usermgmt
// see the same concrete types. See AGENTS.md "identity-model is the domain
// source of truth".

// WebAuthnSessionStore manages WebAuthn challenge session data as opaque bytes.
// The provider serializes *webauthn.SessionData to []byte; the store is
// transport-agnostic and does not import go-webauthn.
// The default in-memory implementation is suitable for single-process deployments.
// Implement this interface with Redis or SQL for multi-instance deployments.
// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
type WebAuthnSessionStore = identitymodel.WebAuthnSessionStore

// VerificationTokenStore manages email verification tokens.
// The default in-memory implementation is suitable for single-process deployments.
// Implement this interface with Redis or SQL for multi-instance deployments.
// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
type VerificationTokenStore = identitymodel.VerificationTokenStore

// LockoutStore tracks failed authentication attempts and enforces temporary account lockout.
// The default in-memory implementation is suitable for single-process deployments.
// Implement this interface with Redis or SQL for distributed lockout across instances.
// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
type LockoutStore = identitymodel.LockoutStore

// PendingTOTPStore manages pending TOTP setup secrets during the enable-TOTP ceremony.
// The default in-memory implementation is suitable for single-process deployments.
// Implement this interface with Redis or SQL for multi-instance deployments.
// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
type PendingTOTPStore = identitymodel.PendingTOTPStore
