# Status — The M2M/AI-agent answer, audited: no durable grant path, a cleanup that removes the wrong rows, and a setter that does not exist

> **ANNOTATED 2026-11-07** — ~~this report~~ **SUPERSEDED** by [`2026-11-06_21-28_m2m-w0-execution-truth-attribution-landed_status.md`](../2026-11-06_21-28_m2m-w0-execution-truth-attribution-landed_status.md), which M2M audit findings + the two identity traps were executed and closed by the M2M W0 execution report (truth attribution landed); open rows live in TODO_LIST. Strike depth:
> supersession verdict (round-18 precedent for series-internal reports); nothing here is a
> unique open item. **ARCHIVED 2026-11-07 (round-23 htmx-session (2026-11-07)).**

**Date:** 2026-11-06, 11:05–11:45 CET
**Repo:** `/home/lars/projects/cqrs-htmx`, branch `master`, shared working tree. One uncommitted modification exists (`scripts/selftests/test-check-conflict-markers.sh`) that this session did not author and deliberately did not touch.
**Scope of this session:** a single in-conversation question — "What does usermgmt offer for machine-to-machine communications or AI Agents?" No code was written, no test was run, no gate was run, nothing was tagged or committed. This report is the retrospective audit of that answer plus the verification pass I owed it.
**One-sentence verdict:** the inventory was right in outline and materially wrong in two places — a bot can never be given authority through the durable (event-sourced) path that this repo forbids bypassing, and the delete-cleanup that supposedly removes a bot's policies cannot remove the exact rows my own recommendation produces — while the answer also missed two decisive negatives for the AI-agent question (bots cannot subscribe to any SSE feed, and machine access is absent from the audit journal) and advertised a setter in a doc comment that no code implements.

---

## Session shape

The whole session is one exchange plus the reads behind it:

| Pass | What | Volume |
| --- | --- | --- |
| 1 | Greped for M2M vocabulary across `usermgmt/`, read `api_token_middleware.go`, `service_bot.go` | ~2 tool calls |
| 2 | Read the bot read model, the token crypto, the fold and event payloads, the authz engine subject handling, `DispatchSubject` | ~6 tool calls |
| 3 | Skimmed module README, skill reference, `FEATURES.md` rows, `adminui`/`setup` call sites | ~5 tool calls |
| Answer | One written reply: what exists, what is missing, one recommendation | ~20 lines |
| Post-answer audit (this report) | Verified the four claims I had inferred, and two more I had not thought to check at all | ~9 tool calls |

Total: reading only. Zero lines of production code, zero tests, zero gate runs before the answer was written. That is the first process defect and it is recorded in §b1.

---

## a) FULLY DONE

### a1. The inventory itself, grounded in reads rather than in `FEATURES.md`

Every row below was confirmed by opening the file, and every file:line is what the answer cited.

| Surface | Location | What it actually is |
| --- | --- | --- |
| Bot lifecycle | `usermgmt/service_bot.go:34` `RegisterBot`, `usermgmt/service_bot.go:75` `DeleteBot`, `usermgmt/service_bot.go:88` `GetBot` | 2 events (`BotRegistered`, `BotDeleted`), a decider, an in-memory read model plus a SQL read model |
| Token minting | `identity-model/crypto.go:15` `GenerateToken` | 32 random bytes, base64url, returned once; only `HMAC-SHA256(token, pepper)` persisted (`identity-model/crypto.go:24`) |
| Token resolution | `usermgmt/service_bot.go:100` `ResolveBotByToken` | read-model lookup by token hash; refuses when no pepper is configured |
| HTTP auth for machines | `usermgmt/api_token_middleware.go:37` `NewAPITokenMiddleware`, `:66` `RequireBot` | Bearer to `*Bot` in context, pass-through by default, 401 gate on demand |
| Machine actor kind | `identity-model/id.go:164-225` | `ActorBot` documented as "machine actor authenticated via API token", plus `ActorSystem`, `ActorService`, `ActorIDFromBot`, `ParseActorID` |
| Dispatch-layer subject | `authz_dispatch.go:612` `DispatchSubject` | users are spelled as raw user IDs, machine actors keep a kind prefix so namespaces cannot collide |
| Policy cleanup on delete | `usermgmt/es_casbin_projection.go:119-128` | bot delete branch — see §d1, this is where the answer inherited a false claim |

### a2. The gaps named at the time that still hold

Four claims were made from direct reads and survive the audit:

1. **Scopes are stored and never enforced.** `Scopes` appears only in `RegisterBotRequest`, the event payload, the command accessor, the read model, and the systemadapter declarations. There is no scope check anywhere in the route layer, the dispatch layer, or the middleware.
2. **One token per bot, no expiry, no rotation, no last-used, no revocation-by-token.** The `Bot` struct (`usermgmt/es_bot_readmodel.go:14-21`) carries exactly `ID`, `Name`, `OwnerID`, `TokenHash`, `Scopes`, `Deleted`; revocation is bot deletion.
3. **No bot routes and no bot panel.** A repo-wide search for `NewAPITokenMiddleware` and `RequireBot` outside tests returns only their own definitions — neither `setup` nor `adminui` mounts them, so bot auth is middleware the consumer wires by hand.
4. **No client-credentials flow.** The OAuth2 module is user login and OIDC; impersonation is user-to-user only.

### a3. The one shape in the answer that is correct and worth keeping

Mounting the token middleware and bridging the bot into the actor chain is the right composition, and the actor kind guard makes it safe: `ActorIDAsUserID` (`identity-model/id.go:205`) refuses to mint a `UserID` from a machine actor, and `RolesForActor` returns nil roles for bot, system, and service actors (`identity-model/authz_engine_test.go:467-480`), so a bot cannot be silently promoted into a human role.

---

## b) PARTIALLY DONE

### b1. The answer shipped before verification, not after

The reporting standard this repo applies to code (verify the claim before writing it) was not applied to prose. Four statements were inferred from adjacent evidence and written as fact; all four turned out to be right or nearly right, which is luck, not method. Two further checks that mattered were never conceived of at all (§d2, §d3).

### b2. The policy-row recipe was asserted from inference

The answer published `p, bot:<id>, *, command/*, execute, allow` as the gating row. The prefix was inferred from `ParseActorID`'s documentation ("kind:raw"); it was verified only afterwards, in `go-cqrs-lite/id` (`actor_id.go:156-161`, `kind.String() + ":" + raw`). Correct — but it should have been checked before publication, because a policy row with the wrong subject spelling denies everything and looks like a policy bug.

### b3. The attribution gap was named, its consequence was not

The answer said a bot-dispatched command "lands un-attributed". The precise consequence is worse and more useful: `bridgeSessionIdentity` (`usermgmt/audit_context.go:73`) only lifts `UserFromContext`, and the token middleware stores the bot without setting an actor, so the command still dispatches successfully and the emitted event carries a zero actor. The failure mode is not "no audit" but "a successful, silently anonymous audit entry" — the worst of the three possibilities, because nothing alerts.

### b4. Persistence nuance omitted

The answer said a SQL read model exists. It did not say that token authentication survives a restart only because hydration rebuilds the `byTokenHash` index (`usermgmt/sql_hydrate.go:280-311`) — the same hydrate-or-lose class that the project already documents for TOTP secrets. A consumer who skips that step gets a working login path and dead tokens.

### b5. Ownership named as data, not as an absent guard

`RegisterBotRequest.OwnerID` was described as "owner = a `UserID`". It was not checked whether the library enforces that relationship anywhere; it does not. `RegisterBot` accepts any owner ID for any caller, and `DeleteBot(ctx, botID, reason)` takes a bare bot ID with no caller-ownership predicate — a contrast with the WebAuthn enrollment path, which is library-enforced (ADR-0055). The service API trusts its caller; that is a legitimate library posture, but it needed saying, because an AI agent issuing bot-management calls is exactly the caller that must not be trusted.

---

## c) NOT STARTED

Checks that a complete answer required and that this session did not run before answering (all of them were run afterwards, results in §d):

1. Exact spelling of a machine-actor policy subject (`PrefixedString`).
2. Whether any bot can be granted authority through an event (`BotRegistered` handling in the Casbin projection).
3. What `RemoveAllRolesForUser` actually removes (grouping rows versus direct policy rows).
4. Whether the bot-delete branch's subject is kind-prefixed or raw.
5. Whether a bot can subscribe to the domain-event SSE feeds.
6. Whether bot token use produces any audit event.
7. Whether `Service.SetTokenPepper` exists (it is named in a doc comment).
8. Whether an exported list-bots API exists (`FindByOwner` is used only by cascade delete).
9. Whether an end-to-end test proves a bot-dispatched command carries a bot actor.
10. Whether `VerifyToken` (constant-time compare) is used on the authentication path.
11. Whether `RequireBot` speaks the project's error vocabulary (it does not — plain `http.Error`).
12. Whether bot policy rows survive journal replay (the durability question that follows from §d2).

---

## d) TOTALLY FUCKED UP

### d1. The claim "CasbinProjection handles bot deletion cleanup" is wrong twice

The answer repeated it from the `FEATURES.md` row. Reading the code (`usermgmt/es_casbin_projection.go:61-128`):

```go
subject := evt.StreamID().Get()   // raw stream id, no kind prefix

case eventBotDeleted:
    // Remove all Casbin policies for the deleted bot.
    if err := p.authz.RemoveAllRolesForUser(subject); err != nil { ... }
```

Two independent defects:

- **Wrong subject spelling.** The cleanup targets the raw stream ID, while a dispatch authorizes a machine actor by its kind-prefixed form (`authz_dispatch.go:612`). Any row written against the spelling the dispatcher actually checks is invisible to this cleanup.
- **Wrong row type.** `RemoveAllRolesForUser` walks domains and roles for a subject and removes grouping policy. Direct `p` rows — the exact shape a bot needs, since machine actors get no roles — are untouched.

So the comment "Remove all Casbin policies for the deleted bot" is false on both counts, and the practical outcome is dead allow rows that accumulate forever while the deleted bot's token is gone. A reader who trusts the comment will believe revocation is complete.

### d2. There is no durable grant path for a bot — which makes the answer's own recommendation unachievable in the sanctioned way

The answer said bot authority is expressible as a policy row and did not ask who writes it. Verification:

- The Casbin projection subscribes to `eventBotDeleted` only; `BotRegistered` has no branch, so bot scopes never become policies.
- `Service.GrantRoles` (`usermgmt/service_grants.go:25`) routes through membership commands, and `RolesForActor` returns nil for non-user actors by design.
- The only remaining door is `Authz.AddPolicy` / `AddGroupPolicy`, and the project documents exactly that as the ephemeral trap (gotcha 45: a grant written into the enforcer is not durable).

For human identities the library forbids the ephemeral path and provides the durable one. For bots it provides neither, so the answer's recipe — declare posture rows for the agent — can only be satisfied by the mechanism the same repository calls a bug. This is the single most consequential thing the session missed, and it directly contradictions the guidance given.

### d3. The doc comment advertises a method that does not exist

`usermgmt/service_bot.go:26-29`:

```go
// TokenPepper is the server-side secret used for HMAC-SHA256 token hashing.
// It must be set before registering or authenticating bots. Set via
// ServiceConfig.TokenPepper or by calling Service.SetTokenPepper.
type TokenPepper = []byte
```

A repo-wide search for `SetTokenPepper` returns that comment and nothing else. The setter does not exist; `tokenPepper` is written exactly once, in `service_factory.go:249`, from the config. A consumer following the comment hits a compile error, which is the mild outcome — the class is the one this repo treats as a defect: the name tells a story the code does not implement.

### d4. Missed negative: a bot cannot consume any realtime feed

The answer described the transport surface as though a machine client could read events. It cannot through the composition root: `setup/sse.go:97` and `setup/sse.go:109` wrap both the domain-event feed and the DataStar feed in `b.SessionMiddleware()(RequireSession(...))`, so a Bearer-only bot gets 401 on every stream. For the AI-agent question this is a first-class fact — the machine path is request/response only, with no subscription — and the answer omitted it entirely, in both the "what exists" and the "what is missing" halves.

### d5. Missed negative: machine access leaves no trace in the audit journal

The answer praised the actor chain and did not check whether token use is journalled. It is not: `BotTokenUsed`, `BotAuthenticationFailed`, `BotAuth` — a repo-wide search returns nothing. The event catalog has 5 auth-audit events, all human-session shaped (`LoginSucceeded`, `LoginFailed`, `RegistrationClosed`, `SessionStarted`, `SessionEnded`). A failed bearer attempt is invisible; a successful one is visible only if it happened to dispatch a command and only if the consumer bridged the actor (§b3). "Which agent used this token, when, and how often" is not answerable from the journal — for an M2M surface, that is the answer to the question that was asked.

### d6. Two smaller errors of the same class

- `RequireBot` rejects with `http.Error` plain text (`usermgmt/api_token_middleware.go:68`), while every other surface in the repo emits an RFC 9457 document with a stable `code`. A bot client therefore has to parse prose to distinguish 401 reasons, on the one surface whose consumer is, by definition, a program.
- The answer stated the read model "indexes by token hash" as a steady-state property without noting that the index is memory-only and rebuilt by hydrate, which is the difference between "works" and "works after a restart" (§b4).

---

## e) WHAT WE SHOULD IMPROVE

### e1. Fix the two defects, not just document them

`es_casbin_projection.go` needs a bot branch that (a) writes scopes or postures as policies on `BotRegistered` using the same prefixed subject the dispatcher checks, and (b) removes those rows on `BotDeleted` with a subject-aware removal that clears direct policy rows too. Doing only (b) without (a) is pointless; doing (a) without (b) leaks. They are one change and it should be gated by a test that registers, grants, authorizes, deletes, and re-authorizes.

### e2. Derive authority from events, or say plainly that bots are not authorizable

The choice in §d2 is architectural, and it has three answers: derive from `Scopes` (the field already round-trips through the event), derive from owner roles (a bot acts as its owner — a delegation model, and the one most AI-agent designs expect), or declare bots out of scope for authorization and let consumers own it. Right now the code implies the third while the middleware implies the second and the `Scopes` field implies the first. That is a split brain with a security shape.

### e3. Close the honesty gaps that cost a reader a compile error or a false belief

`SetTokenPepper` in the doc comment (§d3), the "remove all Casbin policies" comment (§d1), and the `FEATURES.md` row that repeated it — all three are two-line fixes and all three were live at session start.

### e4. Make machine identity observable

Without a bot auth-audit event and without automatic actor bridging, the most useful question about an agent deployment is unanswerable. One event on failure and one on success, plus the actor bridge for a bot in context, would make the whole surface legible in the journal the repo already ships.

### e5. Speak one error vocabulary on the machine surface

`RequireBot` should return the same problem document, with a stable code, that every other rejection returns. Machine consumers are the audience least able to read prose.

### e6. Process: verify before writing, not after

Every defect above was visible in the same files the answer cited. Two of them were one grep away. The cost of a wrong answer here is not a red build, it is a consumer building agent authorization on a mechanism the library cannot durably support. The cheap fix is to treat a claim about a library's capability the way a test treats a behavior: prove it, then state it.

---

## f) Up to 50 things we should get done next

Ordered roughly by the ratio of risk removed to effort spent. None of these were implemented this session.

~~1. Add a `BotRegistered` branch to `CasbinProjection` that derives policy rows from `Scopes`, using the dispatcher's prefixed subject.~~ resolved — routed: TODO_LIST.md "M2M/AI-agent hardening" row, finding 1 — durable grant path (plan T8/W1, blocked on owner decisions D1/D2)
~~2. Fix `eventBotDeleted` cleanup to remove direct policy rows, not only grouping rows, and to target the subject spelling the dispatcher checks.~~ resolved — routed: TODO_LIST.md "M2M/AI-agent hardening" row, finding 2 — subject-correct delete cleanup (plan T10/W1)
~~3. Prove both with one end-to-end test: register, authorize, delete, re-authorize, assert deny.~~ resolved — routed: TODO_LIST.md "M2M owner decisions" row blocks W1 — the end-to-end test is plan T11/A11 behind T8/T10
~~4. Add an audit event for a successful bot authentication (who, when, which token fingerprint).~~ resolved — routed: TODO_LIST.md "M2M/AI-agent hardening" row, finding 5 — audit-journal silence (plan A7/W1, granularity is owner decision D4)
~~5. Add an audit event for a failed bot authentication (unknown token, deleted bot).~~ resolved — routed: TODO_LIST.md "M2M/AI-agent hardening" row, finding 5 — audit-journal silence (plan A7/W1, granularity is owner decision D4)
6. ~~Bridge a bot into the actor chain inside `NewAPITokenMiddleware` (set `cqrshtmx.WithActorID` from `ActorIDFromBot`), so bot-dispatched commands are attributed without consumer code.~~ done at `4084bf49`
~~7. Decide and document the bot authority model: scopes, owner delegation, or consumer-owned.~~ resolved — routed: TODO_LIST.md "M2M owner decisions" row — this is decision D2 (authority source), which blocks W1/W2
8. ~~Delete the false `SetTokenPepper` sentence from `usermgmt/service_bot.go:28`.~~ done at `d877e64e`
9. ~~Rewrite the bot-delete comment in `usermgmt/es_casbin_projection.go:119` to state what is actually removed.~~ done at `d877e64e`
10. ~~Correct the `FEATURES.md` "CasbinProjection" row that claims full bot deletion cleanup.~~ done at `d877e64e`
11. ~~Make `RequireBot` emit the shared problem document with a stable code.~~ done at `4084bf49`
~~12. Add a scope-check middleware or handler option, so `Scopes` stops being decorative.~~ resolved — routed: TODO_LIST.md "M2M owner decisions" row (D2) blocks W1 — scope enforcement rides the authority-source answer
~~13. Give the bot a token prefix or checksum, so a leaked token is identifiable in logs without revealing it.~~ resolved — routed: TODO_LIST.md "M2M owner decisions" row blocks W1 — token prefix + checksum is plan A5/T6
~~14. Add an optional token expiry to the bot aggregate and enforce it in resolution.~~ resolved — routed: TODO_LIST.md "M2M owner decisions" row blocks W1 — token expiry is plan A6/T7
~~15. Add token rotation as a service method that emits its own event.~~ resolved — routed: TODO_LIST.md "M2M owner decisions" row blocks W2 — rotation is plan A26
16. Add a `LastUsedAt` projection field so idle agents are visible.
~~17. Export `Service.ListBots(ctx, ownerID)` instead of leaving `FindByOwner` internal to cascade delete.~~ resolved — routed: TODO_LIST.md "M2M owner decisions" row blocks W1 — the ListBots export is plan A14
18. Add an ownership predicate to bot management methods, or document that the caller owns that check.
19. Verify and, if needed, make constant-time comparison part of the resolve path, or document why a map lookup is acceptable.
~~20. Document the hydrate requirement for token resolution in the usermgmt README bot section (which does not exist yet).~~ resolved — routed: TODO_LIST.md "M2M/AI-agent hardening" row, finding 7 — hydration-dependent resolution (plan T12/W1), the doc rides it
21. Write that bot section: register, mount middleware, bridge actor, authorize, revoke.
22. Add a worked example under `examples/` showing a bot calling a command endpoint.
~~23. State in the README that SSE feeds are session-gated and bots cannot subscribe, with the request/response alternative.~~ resolved — routed: TODO_LIST.md "M2M/AI-agent hardening" row, finding 4 — SSE session-gating (plan T15/W2)
~~24. Add a bot-failure test asserting no token material appears in logs.~~ resolved — routed: TODO_LIST.md "M2M/AI-agent hardening" row, finding 5 — the no-secret leg of plan A7/W1 (granularity is owner decision D4)
25. Consider per-bot rate limiting reusing `httputil` limiters keyed by bot ID.
26. Add a builder/helper that returns a fully wired bot middleware stack so consumers stop hand-ordering it.
27. Add a gate rule that a documented setter must exist (the `SetTokenPepper` class is mechanically detectable).
28. Add a gate or test that every documented policy-subject form matches `DispatchSubject` output.
29. Audit other projections for the same raw-subject-versus-prefixed-subject mismatch.
30. Audit other `RemoveAll*` calls for the direct-row blind spot.
31. Decide whether deleted bots should be tombstoned instead of removed, so audit references stay resolvable.
32. Check whether `Bot.Deleted` is ever observable, given the read model deletes the entry.
~~33. Add bot-shaped entries to the event catalog docs describing when a bot token event will appear.~~ resolved — routed: TODO_LIST.md "M2M/AI-agent hardening" row, finding 5 — catalog registration is a leg of plan A7/W1 (granularity is owner decision D4)
34. Publish the bot surface in the module README table alongside tenants and members.
35. Add a `check-skill-claims` style verification for the bot rows in `FEATURES.md`.
~~36. Confirm bot policies survive a projection rebuild, given they would be event-derived after item 1.~~ resolved — routed: TODO_LIST.md "M2M/AI-agent hardening" row, finding 1 — rebuild survival is proven together with the T8/W1 derivation
~~37. Add a restart test: register bot, restart with hydrate, authenticate, assert success.~~ resolved — routed: TODO_LIST.md "M2M/AI-agent hardening" row, finding 7 — the hydration restart test is plan T12/W1
~~38. Add a rotation test asserting the old token stops resolving.~~ resolved — routed: TODO_LIST.md "M2M owner decisions" row blocks W2 — the rotation test rides plan A26
~~39. Add a deletion test asserting the token stops resolving and the policies are gone.~~ resolved — routed: TODO_LIST.md "M2M/AI-agent hardening" row, finding 2 — the deletion test proves the T10/W1 cleanup
40. Add a cross-tenant test for a bot used in a scoped domain.
41. Review whether `ActorService` and `ActorSystem` deserve the same treatment as `ActorBot` before any of this is built once.
42. Decide whether agent-supplied scopes should be reject-not-ignore when unsupported.
43. Add a validation that scopes are from a closed set, once scopes mean something.
44. Consider exposing a `PermissionChecker`-shaped bot authority so `Decide` answers for agents too.
~~45. Wire the bot subject into `authztest` suites so agent postures are provable like route postures.~~ resolved — routed: TODO_LIST.md "M2M owner decisions" row blocks W1 — the authztest principal matrix is plan A13/T13
~~46. Record the bot authority decision in an ADR before code, since the shape is hard to reverse.~~ resolved — routed: TODO_LIST.md "M2M owner decisions" row — D2 gates ADRs A15/A16 before any W1 code
47. Add a doc line stating what a bot cannot do today, so consumers do not infer capability from the presence of `Scopes`.
~~48. Delete or implement `authztest` coverage for the machine actor kind, whichever is honest.~~ resolved — routed: TODO_LIST.md "M2M owner decisions" row blocks W1 — authztest machine-actor coverage is the A13/T13 matrix
49. Sweep the guides for statements about machine clients that predate the SSE gating.
~~50. After items 1 to 3, re-run this audit and check whether the answer changed.~~ resolved — routed: TODO_LIST.md "M2M/AI-agent hardening" row Verify field — re-run this audit after W1 lands

---

## g) Three questions only you can answer — tracked canonically in the TODO_LIST.md "M2M owner decisions" row since 2026-11-06

1. **Is bot-token M2M a supported product surface, or a vestige?** The `FEATURES.md` marks every bot row green and functional, but nothing in the authorization path treats a bot as grantable. The answer decides whether §f items 1 to 3 are a fix or a removal — and a removal with a deprecation note is a legitimate, cheaper answer that would need your sign-off because it is a published surface.
2. **Should a bot's authority come from its own scopes, or from its owner's roles?** Scopes-fit-agents (narrow, explicit, per-agent) and delegation (a bot acts as the human who registered it, with audit carried on both) are both defensible and produce different data models: the first needs scope-to-policy derivation, the second needs an owner-resolution step at dispatch time plus a delegation record for the audit trail. I cannot pick this from the code, because the code currently implies all three options at once.
3. **Does machine access need to be in the audit journal, and in which stream?** A `BotTokenUsed`-style event per request could be the difference between an auditable agent fleet and an invisible one, but at agent request rates it is also a journal-volume decision, and volume belongs to the owner of the event store, not to me. The alternative is a coarse daily summary or a metrics-only posture, and that choice shapes items 4 and 5.

---

## Verification notes

Everything asserted above is reproducible with the following, run from the repo root:

| Claim | Command |
| --- | --- |
| No scope enforcement exists | `rg -n 'Scopes' --glob '*.go' -g '!*_test.go'` — only request, event, command, read model, systemadapter |
| No `SetTokenPepper` | `rg -n 'SetTokenPepper' --glob '*.go'` — one doc comment, no definition |
| Bot delete uses a raw subject and grouping-only removal | `sed -n '61,128p' usermgmt/es_casbin_projection.go` |
| No bot branch on register | same switch: no `eventBotRegistered` case |
| Machine subject is kind-prefixed | `authz_dispatch.go:612-632`, `id@.../actor_id.go:156-161` |
| SSE is session-gated | `rg -n 'RequireSession' setup/sse.go` — lines 97 and 109 |
| No bot auth-audit event | `rg -n 'TokenUsed\|BotAuth\|AuthenticationFailed' --glob '*.go'` — no output |
| Nothing mounts the token middleware | `rg -n 'NewAPITokenMiddleware\|RequireBot' --glob '*.go' -g '!*_test.go'` — definitions only |

No tests, builds, linters, or gates were executed this session; nothing in the working tree was modified apart from this report and the two budget baselines noted below.

**Baselines:** this report grows the live `docs/status` tail by one and the docs corpus by its line count, so `docs/status/TAIL_BASELINE` and `docs/CORPUS_BASELINE` are bumped in the same change, following the ratchet rule that a deliberate increase travels with the artifact that causes it.
