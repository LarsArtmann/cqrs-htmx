# Supervised enrollment — the short-TTL one-time token that rescues locked-out owners

**Date:** 2026-11-24 · **Status:** PROPOSAL (T01/F008 design note). Nothing here is approved; the mechanism needs a new usermgmt API and is therefore **D2-gated** — it feeds the T02 packet ([`2026-11-24_adminui-d1-d3-and-upstream-asks.md`](2026-11-24_adminui-d1-d3-and-upstream-asks.md)), where it is the self-service alternative to Ask C (admin TOTP bypass) for the *legitimate-owner* case, and the landing spot for audited idea 8 (admin-initiated passkey enrollment).

## Problem

T01 shipped the honest lockout surface: the user-detail page shows a "Cannot sign in" badge when a user holds no WebAuthn credential, no linked external account, and no TOTP. But the badge only *names* the emergency — it cannot *resolve* it. Every rescue is an enrollment act, and every enrollment act is `PostureAuthenticated` in the decider's posture table: `CmdAddCredential` ("enrolling a credential requires an authenticated owner", `usermgmt/command_postures.go:52`) and `CmdLinkExternalAccount` (`usermgmt/command_postures.go:60`). A locked-out user has **no session by definition**, so they can never pass the posture themselves, and the admin cannot pass it for them (the panel cannot run WebAuthn ceremonies or OAuth2 handshakes on someone's behalf — `usermgmt/webauthn_service.go:94/:132` bind the ceremony to the user's own authenticator).

## Proposed mechanism

1. **Mint.** A super-admin mints an *enrollment token* for the user: `EnrollToken(ctx, adminID, userID, ttl) (token string, err error)` — opaque, single-use, user-bound, short TTL (default 15 min), one outstanding token per user (minting replaces the previous). Issuance journals an audit entry naming admin, target, and TTL; the panel reveals the link **once** (the bots' show-once contract).
2. **Consume.** The user opens the link out-of-band (their own device, no session needed). The token satisfies owner-match for **exactly one ceremony**: the passkey `FinishRegistration` dance, or (extension) the OAuth2 link handshake. It authorizes ONE command — it never mints a session, never reads anything, and dies on use or TTL.
3. **Audit.** Consumption journals credential name + provider; expiry and reuse attempts journal refusals. The T01 badge copy stays truthful afterwards: a consumed enrollment clears the lockout badge on the next render (pinned by `adminui/user_auth_state_test.go`).

## Precedent in-tree

`VerificationTokenStore` (`identity-model/interfaces.go:98` — `Save(userID, email, ttl) (string, error)` / `Consume(token) (UserID, error)`) is exactly this shape already: opaque token, TTL-bound, single-consume, user-bound. An `EnrollmentTokenStore` mirrors it; the Service wiring (store + eviction sweeper + config knob) copies the verification-token plumbing.

## What it is NOT

- **Not a session and not a login.** Nothing grants access to the account; only an *additive* enrollment is authorized.
- **Not an MFA bypass.** Unlike Ask C (`AdminDisableTOTP`), nothing is removed or weakened — a lost-device user keeps their TOTP and gains a new passkey. This is why the two asks are complements, not alternatives.
- **Not a lockout bypass for attackers.** The guard is the *admin's* super-admin gate on minting, not the token: an attacker with neither a session nor an admin cannot mint, and a minted token is useless without the link.

## Alternatives considered

- **Admin TOTP bypass (Ask C):** needed anyway for the *hostile/lost-factor* case (TOTP stripping victim), but it is destructive and needs heavier guardrails. Supervised enrollment covers the benign majority (fresh account, lost passkey, new device) without removing any factor.
- **Temporary password:** N/A — usermgmt has no password factor at all (passkeys / OAuth2 / TOTP only).
- **Do nothing:** rescue = re-register + re-grant roles; loses the user's stream, memberships history, and audit continuity.

## Open questions for the owner

TTL default (15 min?) · who may mint (super-admin only, or tenant admins for their members?) · whether the token also covers the OAuth2 link handshake in v1 · mint rate limiting · audit vocabulary entry (`EnrollmentTokenIssued`/`Consumed`/`Expired`) · whether the panel exposes a "supervised enrollment" action next to the T01 lockout badge (needs no new panel API beyond the minting endpoint once D2 lands).
