# Adminui owner decisions D1–D3 + upstream asks (h)(1)–(5) — decision packet

**Date:** 2026-11-24 · **Status:** DRAFT — packet for the owner; nothing here is approved and NO ask is filed/sent.
**Provenance:** fine tasks F011–F016 of T02 in [`docs/planning/2026-11-24_06-31_SUPERB-adminui-next-wave-pareto-plan.md`](../planning/2026-11-24_06-31_SUPERB-adminui-next-wave-pareto-plan.md). Refreshes (does not replace) the pre-staged recommendations in [`docs/proposals/2026-11-14_adminui-next-wave-owner-decisions-d1-d3.md`](../proposals/2026-11-14_adminui-next-wave-owner-decisions-d1-d3.md).
**Related design note:** [`docs/decisions/2026-11-24_adminui-supervised-enrollment-note.md`](2026-11-24_adminui-supervised-enrollment-note.md) (sibling, T01/F008 — the supervised enrollment-link flow that feeds the MFA-bypass discussion below; linked by path even if it lands minutes later).

All file:line citations verified against the working tree on 2026-11-24. Every ask below is D2-gated: nothing is implemented against unpublished API.

## 1. Verbatim current rows (F011)

### D1–D3 — TODO_LIST.md, "adminui next-wave owner decisions D1–D3" row (P1)

> - [ ] **adminui next-wave owner decisions D1–D3 (canonical list — the 2026-11-14 SUPERB adminui plan §7 points here; blocks impersonation/bots/sessions/rename panel work, NOT the plan's own M02–M17).** **Recommendations pre-staged 2026-11-14: [`docs/proposals/2026-11-14_adminui-next-wave-owner-decisions-d1-d3.md`](../proposals/2026-11-14_adminui-next-wave-owner-decisions-d1-d3.md)** (impersonation GO behind opt-in config / batch all four usermgmt asks into ONE tag train / sessions+impersonation before bots) — one tick per row converts the follow-on waves to GO. Plan erratum recorded there: published usermgmt v4.20.0 already exposes `BeginImpersonation`/`EndImpersonation` + the ADR-0060 forensics, so D1 needs NO API change — it is a panel-surface + guardrail decision. (D1) Ship impersonation in the admin panel? (D2) May adminui require new usermgmt APIs (the four asks of upstream-asks row (h))? (D3) Next-wave priority: bots / sessions+impersonation / import-export depth / hardening. Verify: answers recorded here before any gated panel code lands.

### (h)(1)–(5) — TODO_LIST.md, "Upstream asks this repo owns or watches" row, item (h)

> (h) **Four usermgmt API asks from the adminui tenant/user-management wave (NEW 2026-11-14, plan `docs/planning/2026-11-14_08-45_SUPERB-adminui-tenant-user-management-next-wave-pareto-plan.md` §2.2/§7 D2; filing owner-gated): (1) bot listing** — `RegisterBot`/`GetBot`/`DeleteBot`/`ResolveBotByToken` exist (v4.20.0, verified in the module cache) but no enumerate/list, so the panel cannot show bots at all; (2) **batched membership counts** — per-tenant member counts require one query per tenant today (the tenants list resolves them N+1); (3) **tenant rename** — display name is written once at creation, no rename command; (4) **session listing/revocation** — sessions are ephemeral by ADR-0012 and the auditing journal records starts/ends, but there is no enumerate-or-revoke API for an admin "sign out everywhere". **NEW (5) tenant attribution in audit entries** (2026-11-14, M10 execution): the audit log records NO tenant lifecycle entries — verified empirically (a CreateTenant dispatch produces zero entries; commandAuditMiddleware covers user/session commands only) — and membership entries aggregate on membership streams with no tenant field, so the plan's per-tenant audit shortcut (T104) is unimplementable and was reverted rather than shipped as a dead-end link; tenant-attributed entries (or a tenant filter field on AuditEntry) unblock it. All four are D2-gated: yes = a usermgmt tag + adminui pin bump (gotcha-30 root-tag-gap dance); no = the panel stays on published API with the documented workarounds. Verify: each item links its upstream artifact.

## 2. Ask-by-ask: API sketch + verify-before-filing evidence (F012/F013)

### Ask A — `RevokeAllSessions(ctx, userID)` (sign-out-everywhere)

- **Sketch:** `func (s *Service) RevokeAllSessions(ctx context.Context, adminID, userID UserID, reason string) error` — super-admin-gated, non-empty reason, delegates to `s.sessions.DeleteByUserID(ctx, userID)` (the auditing wrapper journals one `SessionEnded` batch), then logs `logAuth`.
- **Exists:** the store capability is already published: `SessionStore.DeleteByUserID` in the interface (`usermgmt/store.go:102`), implemented in-memory (`usermgmt/store.go:148`) and SQL (`usermgmt/sql_session_store.go:367`), journaled by `auditingSessionStore.DeleteByUserID` (`usermgmt/session_audit.go:100`). Already exercised on the user-delete path (`usermgmt/service_register.go:173`).
- **Missing:** no Service-level method exposes it for admin use; the only caller is the delete cascade. The ask is a thin Service surface over an existing store method — the cheapest of the five API asks.

### Ask B — `SessionsForUser(ctx, userID)` (session enumeration)

- **Sketch:** `func (s *Service) SessionsForUser(ctx context.Context, userID UserID) ([]SessionView, error)` returning token-prefix (never the raw token), created/expires, origin kind (incl. impersonation origin + actor), for a read-only admin sessions list.
- **Exists:** `identitymodel.Session` carries everything a view needs (`identity-model/session.go:16-22`: Token, UserID, Origin `SessionOrigin`, CreatedAt, ExpiresAt); impersonation origin is a typed variant (`usermgmt/user.go:13-15`, `usermgmt/service_impersonation.go:115` type-asserts `session.Origin.(Impersonation)`).
- **Missing:** `SessionStore` (`usermgmt/store.go:97-103`) has exactly `Create`/`Find`/`Delete`/`DeleteByUserID` — no enumeration query in the interface, either impl, or the contract test (`usermgmt/session_store_contract_test.go`). New interface method + both implementations + a token-redaction rule are required.

### Ask C — Admin TOTP bypass (explicit reason + audit event)

- **Sketch:** `func (s *Service) AdminDisableTOTP(ctx context.Context, adminID, userID UserID, reason string) error` — super-admin-gated, non-empty reason mandatory, dispatches the existing `NewDisableTOTPCmd` after validation, emits a new audited event (e.g. `AdminTOTPDisabled`-class action in the audit vocabulary) naming admin, target, and reason. blast-radius copy in the panel (the T01 badge language).
- **Exists:** the disable command and event pipeline already exist — `DisableTOTP` dispatches `NewDisableTOTPCmd` (`usermgmt/totp.go:128`). The related design note's supervised enrollment-link flow (short-TTL one-time token) is the alternative self-service rescue path for the *legitimate-owner* case.
- **Missing:** the only disable path requires the USER'S OWN valid code: `DisableTOTP` (`usermgmt/totp.go:117`) calls `requireValidTOTP` (`usermgmt/totp.go:137-156`), which validates against the stored `user.TOTPSecret` (`usermgmt/totp.go:151`) — deliberately, per the doc comment: "prevent MFA stripping via session hijack". No admin actor, reason parameter, or bypass audit event exists. A new API is unavoidable; the guardrail design (who may bypass, what is audited) is the owner's call.

### Ask D — `LockoutStatus(userID)` (lockout introspection)

- **Sketch:** `type LockoutStatus struct { Locked bool; LockedUntil time.Time; FailedAttempts uint; MaxAttempts uint }`; `func (s *Service) LockoutStatus(ctx context.Context, userID UserID) (LockoutStatus, error)`.
- **Exists:** the lockout machinery is real but internal: `LockoutStore` interface is email-keyed with only `IsLocked`/`RecordFailure`/`Reset` (`identity-model/interfaces.go:106-109`); the default impl `AccountLockout` (`usermgmt/lockout.go:27-33`) is in-memory (lost on restart, per its own doc comment), with `IsLocked` (`lockout.go:65`), `RecordFailure` (`lockout.go:103`), `Reset` (`lockout.go:127`). The Service holds it privately (`usermgmt/service_core.go:40`, knob `ServiceConfig.Lockout` at `service_core.go:91`, wired `service_factory.go:160`). The aggregate state carries NO lockout fields at all (`identity-model/fold.go:13-29`, `UserState`).
- **Missing:** everything user-facing: no Service method, no userID keying (email-keyed), no remaining-attempts/locked-until introspection, no persistence. Correction to the task brief: identity-model has a lockout *interface*, not lockout *state fields* — the state structs (`UserState`/`TenantState`, `fold.go:13/:80`) carry none. This ask is the largest design surface of the five (interface extension vs event-sourced lockout is an owner-visible choice).

### Ask E — Bot listing (enumerate registered bots)

- **Sketch:** `func (s *Service) Bots(ctx context.Context) ([]*Bot, error)` (optionally `BotsForOwner(ctx, ownerID)`), read-model only, no token material.
- **Exists:** `RegisterBot`/`DeleteBot`/`GetBot`/`ResolveBotByToken` on Service (`usermgmt/service_bot.go:35/:76/:89/:103`). **Partial coverage found:** the read model already has `FindByOwner(ownerID) []*Bot` (`usermgmt/es_bot_readmodel.go:114-116`, used at `usermgmt/service_misc.go:133`) — owner-scoped listing exists at read-model level but is not exported as a Service method.
- **Missing:** no `All()`/global enumerate on `BotReadModel` (methods are `Name`/`EventTypes`/`Handle`/`FindByID`/`FindByTokenHash`/`FindByOwner`, `es_bot_readmodel.go:48-116`) and no Service surface for either scope. The TODO row's "no enumerate/list" is accurate at Service level; the packet should note the partial so the ask is scoped as "export FindByOwner + add All", not "invent listing".

### Ask F — `RenameTenant` (tenant display-name change)

- **Sketch:** `func (s *Service) RenameTenant(ctx context.Context, tenantID TenantID, displayName string) error` dispatching `identitymodel.NewRenameTenantCmd`.
- **Exists — far more than the TODO row claims:** the vocabulary is complete in identity-model: `RenameTenantCmd` + `NewRenameTenantCmd` (`identity-model/commands.go:377-390`), `CmdRenameTenant` (`identity-model/constants.go:77`), `EventTenantRenamed` + `TenantRenamedPayload` (`identity-model/constants.go:46`, `identity-model/events.go:155-159`), the fold branch (`identity-model/fold.go:386-387`), a posture row (`usermgmt/command_postures.go:88-89`), and an event-catalog entry (`usermgmt/es_event_catalog.go:218-222`).
- **Missing:** the usermgmt wiring only — no `decideRenameTenant` (the decider has create/suspend/reactivate/delete only, `usermgmt/es_tenant_decide.go:19/:57/:90/:121`), no dispatch registration (`RegisterTenantCommands` wires exactly four commands, `usermgmt/es_tenant_dispatch.go:21/:37/:53/:69`), no Service method (`usermgmt/service_tenant.go:18-151`). **This corrects the (h)(3) premise** ("no rename command"): the command exists; three glue points are missing. Cheapest ask after Ask A.

### Ask G — Batched member counts (avoid per-tenant `TenantMembers` calls)

- **Sketch:** `func (s *Service) TenantMemberCounts(ctx context.Context, tenantIDs []TenantID) (map[TenantID]int, error)` — one read-model pass over memberships, grouped by tenant; the tenants page calls it once per render with the visible tenant set.
- **Exists:** `TenantMembers(ctx, tenantID)` (`usermgmt/service_tenant.go:145-147`) delegates to `membershipReadModel.FindByTenant` — the only membership-read surface, per-tenant by construction; `AllTenants` (`service_tenant.go:138`) has no count companion.
- **Missing:** any batched/counting query. The N+1 on the tenants list (the (h)(2) claim) is structural: each count is one full membership scan. Purely additive read-model work; no new event vocabulary.

### Ask H — Tenant-attributed audit entries

- **Sketch:** two owner options: (a) widen `AuditEntry` with `TenantID TenantID` (omitzero) + extend `AuditLog` queries with a tenant filter (`EntriesForTenant`); (b) add tenant lifecycle actions to the audited event set so `CreateTenant`/`SuspendTenant`/`DeleteTenant` produce entries keyed on the tenant aggregate.
- **Exists:** `AuditEntry` (`usermgmt/audit_log.go:21-36`) fields are EventType/AggregateID/OccurredAt/ActorID/UserID/Email/Action/Detail — **no tenant field**; the action vocabulary (`audit_log.go:39-61`) is user/session-only; the audited set is exactly `allUserEventTypes + allAuthEventTypes` (`audit_log.go:145`) — tenant/member/bot events are excluded by construction; queries are `Entries` (`:210`), `EntriesFor(aggregateID)` (`:223`), `Recent` (`:266`) — no tenant dimension to filter on. `commandAuditMiddleware` (`usermgmt/audit_context.go:30-37`, wired `service_factory.go:253`) covers the dispatcher chain, but only user/auth events reach `AuditLog`.
- **Missing:** both the field and the event-set widening; membership entries carry the tenant only implicitly via the membership stream's aggregate ID (`MembershipState`, `identity-model/fold.go:60-65`). The (h)(5) premise verifies exactly as written.

## 3. Recommendations per decision (F014)

### D1 — Ship impersonation in the admin panel?

**Recommendation: GO, behind an opt-in config knob.** `BeginImpersonation`/`EndImpersonation` are published since usermgmt v4.20.0 (`usermgmt/service_impersonation.go:26/:109`) with the guards already in place (non-empty reason :29, no self-impersonation :36, super-admin role check :52-65, target-exists :75, full Origin forensics :85/:98-102). D1 is a panel-surface + guardrail decision only — no API change, no D2 dependency.
**Dismissed:** *block until new API* — false premise (the 11-14 erratum stands; verified above); *fork sessions outside usermgmt* — loses the typed `Impersonation` origin that `EndImpersonation` and any future sessions list rely on; *permanent block* — the API is already public; an opt-in panel does not expand exposure, it adds audit visibility over a capability that exists today.

### D2 — May adminui require new usermgmt APIs?

**Recommendation: YES — batch ALL asks into ONE tag train** (single gotcha-30 root-tag-gap dance), with the packet-corrected scope: A (thin Service surface over an existing store method), B (new store query + redaction), C (new bypass path + audit event — the guardrail-heavy one), D (largest design choice: interface extension vs event-sourced), E (export FindByOwner + read-model All), F (wire existing identity-model vocabulary — cheapest new capability), G (additive read-model query), H (audit schema + event-set widening, owner picks option (a)/(b)).
**Dismissed:** *per-ask trains* — N release dances for one consumer family; *adminui-local workarounds* (client-side count caches, scraping the audit window) — they diverge from the published contract and silently rot; *never-new-APIs posture* — contradicts the monorepo reality: usermgmt and identity-model are this repo's own modules; the "upstream ask" is a tag train, not a cross-repo filing.

### D3 — Next-wave priority?

**Recommendation: sessions+impersonation first, bots second** (the 11-14 ordering stands, now with verified grounding): impersonation needs zero API (D1), session admin needs the smallest delta (A+B over an existing store), and bots carry a SECOND unanswered owner gate — the M2M owner-decisions row (TODO_LIST "M2M owner decisions", D1–D4: bot support/deprecation, scopes-vs-delegation, delegator model, audit granularity) must tick before any bot panel work per that row's own "do NOT start W1" rule.
**Dismissed:** *bots first* — stacks two owner gates on one wave; *import-export depth first* — no gate, but lowest security value against the locked-out-user emergency T01 targets; *hardening-only wave* — the no-decision slice of hardening is already T01's honest auth-state surface, so a hardening-only wave duplicates shipped scope.

## 4. Boundaries

- Nothing in this packet is sent, filed, or implemented: all eight asks stay owner-gated (K-gates, plan §7); SEND happens only on the owner's tick of D2 plus per-ask approval.
- The (h) row's "Verify: each item links its upstream artifact" remains unsatisfied by design — this packet is the pre-filing evidence base, not the artifact.

**UI specs (T31/F221–F224):** [`2026-11-24_adminui-gated-surface-ui-specs.md`](2026-11-24_adminui-gated-surface-ui-specs.md) — wireframes + interaction contracts for the impersonation kit, bots pack, sessions/MFA-bypass/lockout, and tenant rename/counts/attribution surfaces, each consuming only the D1/D2/D3 decisions and asks A–H documented above (paper only; nothing gated is implemented).
