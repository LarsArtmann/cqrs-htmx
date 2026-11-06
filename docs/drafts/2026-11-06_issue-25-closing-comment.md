XS scope shipped. `SessionStarted`/`SessionEnded` are emitted from the one chokepoint (`NewService` wraps the configured `SessionStore`; the interface is unchanged), so register, WebAuthn/OAuth2 login, impersonation begin/end, logout, lazy expiry and user-deletion revocation all journal. Impersonation `By`/`Reason`/`At` ride the payload; the token is never journaled, `SessionTokenHash` is the correlation key.

The verification criterion holds: `TestSessionAudit_ImpersonationForensicsSurviveSessionDeletion` deletes the session rows first and still answers who impersonated whom, when, and why from the journal alone. Shape and limits in ADR 0060, sweeper gap included: https://github.com/LarsArtmann/cqrs-htmx/blob/master/docs/adr/0060-session-lifecycle-audit-events.md

Unreleased on master until the next train — identity-model has to be tagged before usermgmt can consume the events. Full `sessionlifecycle` module stays demand-gated.
