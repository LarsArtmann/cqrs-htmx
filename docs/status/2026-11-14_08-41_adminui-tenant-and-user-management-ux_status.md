# Admin UI/UX wave: Tenant integration and User management (status report)

**Date:** 2026-11-14 08:41 CET · **Session:** one continuous UX/implementation session on the adminui module (tenant integration + user management), no plan of record, no sibling coordination · **Tree at end:** master clean at the daemon's `51c49f15`; adminui 158 tests green (workspace AND `GOWORK=off`), lint 0 issues, route-posture gate green (adminui 4 manifest tests), CSS bundle rebuilt (+1 class token) and both CSS gates green · **Not run this session:** `nix run .#check-modules`, `.#coverage-gate`, `.#check-cqrs-lint`, `.#e2e*` (environmental blocks recorded in §d).

> **Scope note.** The user asked: "How can we improve the Admin UI/UX especially for Tenant Integration and User Management?" — then: break it down, execute and verify one step at a time. This report covers ONLY that work and what was noticed while doing it. Nothing here claims progress on the backlog rows this session did not touch.

## a) FULLY DONE (verified, with receipts)

1. **Member rows now name people, not actor tokens.** Before: the tenant roster rendered `user:01HZ…` as the primary label, so an admin could not tell who a member was. Now `memberRow` (`adminui/models.go`) carries the resolved directory identity (`Email`, `Name`, `IsUser`) plus `AddedAt`; `toMemberRows` resolves through `identitymodel.ActorIDAsUserID` → `ReadModel().FindByUserID`; the row renders avatar, email (linked to the user page in super-admin mode), display name, role badges, the join time, and the raw prefixed id only for non-user actors (bot/system/service). Tenant-admin mode deliberately renders no user links because that mode serves no user route. Receipt: `adminui/tenant_ux_test.go` (`TestTenantDetail_MembersRenderAsPeople`, `TestTenantAdminMembers_NoUserLinks`).

2. **Add-member is now a picker, not a typing test.** The email input carries a `<datalist>` fed from the directory (zero JS, CSP-safe), minus people who are already members (a duplicate add is refused by the decider, so suggesting one is a dead end). The role select gained a disabled "Choose role…" prompt plus `Required`, so a distracted Enter can no longer grant the first role in the list. Receipts: `TestMembersCard_AddMemberAutocomplete`, `TestMembersCard_ExistingMembersNotSuggested`.

3. **Unknown-email handling is honest and inline.** Before: a toast said "No user with that email" and the typed address was lost. Now `doAddMember` redirects with `member_error=unknown&member_email=…`, the card renders a warning alert with the address, preserves the typed value in the form, and (super-admin only) offers a "Create the user" button. The query values are templ-escaped and an unknown `member_error` code renders nothing. Receipts: `TestPanel_TenantAdminAddMemberUnknownEmail` (rewritten to assert the banner + preserved value), `TestFlow_AddMemberUnknownEmail` (rewritten to assert the carried query).

4. **Tenant detail leads with the tenant's identity.** New "Tenant" card: copyable tenant id (reused affordance, generalized from the user page as `copyableID`), immutable slug, display name, live status badge, roster size. Receipt: `TestTenantDetail_MetaCard`.

5. **Tenant roster search + per-tenant member counts.** The tenants list gained the same live HTMX search as users (slug or display name, 350 ms debounce, spinner, honest empty state, pagination that preserves `q`, fragment-only HTMX response) and a Members column ("1 member"/"N members"). Receipts: `TestTenantsIndex_SearchAndMemberCounts` (search, miss state, partial contract, counts).

6. **User directory gained the access picture.** New Access column: platform (wildcard-domain) role badges plus a tenant-membership count; a fifth filter chip ("Administrators") narrows to platform-role holders; search now also matches the user id (the handle an operator actually pastes from a log). The count only includes domains that resolve to a real tenant — registration grants default roles in a domain named after the user's own stream, so a naive domain count would report a phantom tenant for every account (found live while testing: a fresh user reported two domains). Receipts: `adminui/users_access_test.go` (access column, dash fallback, chip, chip+search composition, id search).

7. **User detail now manages tenant access from the user side.** New "Tenant access" card: memberships resolved to tenant identity (display name + slug, linked to the tenant page) with role badges; a grant form (tenant + role) offered only for tenants the user does NOT belong to; a per-row role control; a revoke action; plus a "Tenants" stat card. All three writes dispatch the SAME durable membership commands the tenant roster uses (`AddMember` / `UpdateMemberRoles` / `RemoveMember`), and unknown roles/tenants are refused before any dispatch. Routes added: `POST /users/{id}/tenants`, `POST /users/{id}/tenants/{tenant}`, `POST /users/{id}/tenants/{tenant}/delete`, each with mode + write + Why, and the manifest golden updated. Receipts: `adminui/user_tenant_access_test.go` (render, empty state, grant, refused duplicate, role replacement, bad-input refusal ×4, revoke), `TestRoutes_ManifestGolden`, `scripts/checks/check-route-postures.sh` (adminui 4/4).

8. **The tenant creation form is labeled.** Both fields carried only placeholders; they now have labels and help text (the slug is immutable and URL-safe, the display name optional).

9. **Docs and receipts.** `adminui/CHANGELOG.md` `[Unreleased]` (Added / Changed / Fixed, with the API + route vocabulary), `adminui/README.md` (Users/Tenants/Members bullets rewritten to the shipped behavior), `FEATURES.md` (User Management + Tenant Management rows). CSS bundle rebuilt with the canonical recipe: exactly ONE class token added (`max-w-[220px]`), zero removed; `check-css-bundles.sh` green (87,917 bytes, canaries), class-set gate green in fixture mode (1,076 tokens exact).

10. **Live smoke test against the seeded demo** (`examples/admin-demo` on `:18930`, dev-login, then real HTML inspected): tenant page shows the meta card, person rows (`alice@acme.dev`, `bob@acme.dev`), "just now" join times, a datalist of non-members, and the "Choose role…" prompt; users page shows the Access column with a `super_admin` badge; user detail shows the Tenant access card and grant form. Consumer builds green: `setup`, `examples/admin-demo`, `examples/setup-demo`, `integration_test`.

## b) PARTIALLY DONE

1. **Role editing is single-role replace.** `UpdateMemberRoles` accepts a slice and memberships can hold several roles, but both role controls (tenant row and user row) are a single `<select>` + "Set", which flattens a multi-role membership. The badges show the truth, so the UI does not lie, but it cannot EXPRESS a multi-role edit. A checkbox set per row is the fix; not attempted.
2. **Tenant roster counts are derived per row.** `TenantMembers` scans memberships per tenant, so the list is O(page × memberships). Page-bounded at 50, documented in code, acceptable now; a batched count query is the real fix and needs an upstream read-model API.
3. **The user-row tenant count is N+1-ish.** For each visible user x each policy domain, one `GetTenant` lookup. Fine at 50 rows; a batched membership query would remove it.
4. **Autocomplete is capped.** The datalist carries at most `MaxListRows` (200) directory emails, sorted alphabetically. Larger directories get a truncated suggestion set (free-text entry still works); "type-ahead search" is the real answer.
5. **No e2e coverage for the new surfaces.** The Playwright battery was NOT run (no Chromium on this host, §d) and no specs were added. The adminui behavior spec asserts on row counts and the confirm modal, which still hold, but nothing pins the new tenant-access card, the datalist, or the inline error banner at browser level.
6. **Coverage not measured.** The `coverage-gate` was not run; the new code is covered by 20+ new unit tests but the module percentage is unknown (FEATURES records 70.1% against a 66% threshold from an older run).
7. **No dark-mode / a11y verification of the new cards.** The panel is class-driven dark mode; the new Tailwind classes are existing utilities (only `max-w-[220px]` was new) but nothing was visually checked in dark mode, and no axe probe ran.
8. **The tenant-access grant select has a dead-end edge.** If a user belongs to every live tenant, the select renders only its disabled prompt. There is a hint line, but the form should be hidden and replaced with an explicit "member of every tenant" state.

## c) NOT STARTED

1. **Impersonation UI.** `usermgmt.Service.BeginImpersonation`/`EndImpersonation` exist with real guards (super_admin caller, no self-impersonation, mandatory reason, audited session origin) and no adminui surface. It is the largest user-management capability still missing, and the most security-sensitive.
2. **Session introspection / "sign out everywhere".** No public Service API lists sessions; `revokeSessionsBestEffort` is unexported. The UI cannot show active sessions or force-revoke them.
3. **Bot / API-token management.** `RegisterBot`, `GetBot`, `DeleteBot`, `ResolveBotByToken` exist; the panel has no bots section. M2M tokens are currently unmanageable from the UI.
4. **Import / export.** `ImportUsersFromCSV`, `ImportUsersFromJSON`, `ExportUsersToJSON`, `ExportUsersToCSV` exist; no panel surface.
5. **Tenant rename / display-name edit.** No Service API; the slug is immutable by design, but the display name ought to be editable.
6. **Member attribution ("added by").** Membership carries `AddedAt` but no actor; the roster cannot say who granted access.
7. **Sortable columns, bulk actions, saved views, export of the current filter.**
8. **Last-login / risk hints on the directory** (would need session or audit-derived data).
9. **e2e specs for the new flows**, and an adminui screenshot refresh.
10. **The adminui CHANGELOG receipt gap** between `v4.12.4` and `v4.16.0` (TODO_LIST row 50) — untouched.

## d) TOTALLY FUCKED UP

Nothing in the delivered code is broken. Three honest admissions:

1. **I shipped a lie in UI copy for ~10 minutes.** The first version of the tenant-access card said "Granting a tenant the user already belongs to replaces their roles there", taken from the `AddMember` doc comment in `usermgmt/service_membership.go` ("Re-adding the same actor replaces its role set"). The decider actually REFUSES a duplicate (`already in this tenant`), which the new test caught immediately. Fixed properly: a separate update route + command for role changes, and the grant select excludes existing memberships. **The upstream doc comment is wrong and should be corrected** — it will mislead the next consumer the same way.
2. **The canonical CSS builder cannot run on this host.** `nix run .#build-adminui-css` fails before Tailwind starts: the flake's Go override builds `go-1.27.3` from a fixed-output derivation whose hash is the placeholder `sha256-AAAA…`, so the toolchain build aborts with a hash mismatch. I reproduced the recipe by hand with the same Tailwind version the flake pins (4.3.3), then proved the delta is exactly one added class token and zero removals, and ran both CSS gates (form + class-set fixture mode). **The repo-level flake defect is not mine and is still open**; until it is fixed, `check-css-bundle-classes.sh` (which rebuilds through the flake app) cannot prove anything on a host like this one.
3. **The demo server I used for the smoke test was started and later killed**; its log lives in `/tmp/admin-demo.log`. No stray process is expected, but the `e2e-kill-strays.sh` ritual was not run.

## e) WHAT WE SHOULD IMPROVE (process and quality)

1. **Read the decider, not the doc comment.** The one real defect of this session came from trusting a Service doc comment about idempotency. For any write, the decider's reject paths are the contract.
2. **The member/tenant surfaces were built bottom-up (list first, identity second).** Names are the primary key of an admin UI; resolving actor → person should have been step zero, not the realization after reading the template.
3. **No e2e lane means UI changes are only unit-proven.** The repo has a strong browser battery; adding three routes and a card without a browser assertion is a coverage hole I knowingly accepted because the environment has no Chromium.
4. **The auto-commit daemon split this work into four commits with no narrative.** The CHANGELOG is the only coherent receipt. A session-long change on a UX surface deserves a single human-written commit message; the daemon makes that impossible unless I commit myself at phase boundaries (gotcha 4 says exactly this — I did not).
5. **Baseline bookkeeping was deferred to the end.** The report, the corpus re-true and the tail bump must land together; discovering the tail is already at 12 required an extra check late in the session.
6. **Two cross-module API needs were worked around instead of decided.** Phantom-domain filtering and per-row tenant lookups exist because no batched membership query does; the workaround is documented but it is still a workaround.
7. **The `Set` control in a row re-dispatches even when the role is unchanged.** Harmless (idempotent update), but it writes an event for a no-op.
8. **No visual diff.** The new cards were verified by HTML string inspection, not screenshots; spacing/alignment issues would not be caught.

## f) WHAT WE SHOULD GET DONE NEXT (ranked, 50)

**A. Close the loop on this wave (1-8)**
1. Add a Playwright spec for the tenant-access card (grant, role change, revoke) once a Chromium is available.
2. Add a browser spec for the add-member datalist + inline unknown-email banner.
3. Run the full `.#e2e` battery; fix any selector/row-count fallout from the new columns.
4. Dark-mode + axe probe of the tenant detail, user detail and tenant list.
5. Fix the flake's `go-1.27.3` source hash so `build-adminui-css` and the canonical class-set gate run again.
6. Run `nix run .#check-modules` and `.#coverage-gate`; record the adminui percentage.
7. Run `.#check-cqrs-lint` and `.#check-docs-gates-battery`; annotate/resolve findings.
8. Correct the `AddMember` doc comment upstream-side (or in usermgmt here, with a CHANGELOG note).

**B. User management completion (9-20)**
9. Multi-role membership editing (checkbox set) on both tenant and user surfaces.
10. Impersonation: opt-in `Config.Impersonation` (cookie name, super_admin gate, mandatory reason, persistent banner, end action).
11. Session introspection + "sign out everywhere" (needs a public Service API).
12. Bots / API-token section (list, create, rotate, revoke).
13. Import/export entry points (CSV/JSON) with a dry-run summary.
14. Tenant display-name editing.
15. Member attribution ("added by", from membership events).
16. Last-login column (audit-derived) on the directory.
17. "Member of every tenant" state instead of a disabled-only select.
18. Bulk user actions (delete, role grant) with a single confirm summary.
19. Sortable/pinned columns on users and tenants.
20. Export the current filtered list.

**C. Tenant integration depth (21-32)**
21. Tenant switcher in the top bar (super-admin jumps between tenants without going through the list).
22. Tenant detail: recent tenant-scoped activity preview (reuse the audit labels).
23. Tenant detail: per-tenant role distribution summary.
24. Tenant list: suspended/deleted chips + filter.
25. Tenant list: sort by member count / name.
26. Tenant detail: "add member" that can also create the user in place (super-admin).
27. Guard rails on last-owner removal inside a tenant.
28. Show whether a tenant's members can reach any panel surface (authz dry check).
29. Tenant-delete preflight: name the memberships and sessions that will be revoked.
30. Tenant suspension: show how many users are currently affected.
31. Cross-link tenant ↔ user pages in both directions (tenant row → user, user row → tenant is done).
32. A per-tenant audit filter shortcut from the tenant page.

**D. Directory quality (33-42)**
33. Type-ahead member search endpoint (replace the 200-email datalist cap).
34. Server-side dynamic filter/query API to retire the in-memory filter + phantom-domain workaround.
35. Batched membership counts (remove both N+1s).
36. Empty/edge-state audit across all new cards.
37. Convert the new write handlers to the typed/templ partial path where it removes duplication.
38. A11y: announce the inline unknown-email alert and the tenant-access success (aria-live).
39. Add `data-sync-target` markers to the new cards so SSE refreshes can target them.
40. Empty-state CTA on the users list ("create the first user").
41. Reduce duplication between the two member-row renderers (tenant roster vs user tenant access) if a shared primitive emerges.
42. Verify the new routes are covered by `authztest` route-posture proof (non-admin caller gets 401/403).

**E. Process / docs (43-50)**
43. Add the tenant-access routes to the adminui README route table (if one exists) and the guide.
44. Add a `docs/gotchas.md` entry for the "Service doc comment vs decider reject path" trap (one real miss this session).
45. Reconstruct the adminui CHANGELOG gap `v4.12.4` → `v4.16.0` (TODO row 50).
46. Archive settled status reports and lower `TAIL_BASELINE` (the tail is at the 12 ceiling before this report).
47. Re-run `check-docs-corpus-budget` after this report and ledger the delta.
48. Add a `check-adminui-surface` gate (route table ↔ rendered nav ↔ templates) if the surface keeps growing.
49. Record the flake Go-hash breakage in the TODO_LIST as an owned red with a reproduction.
50. A short demo-video/screenshot refresh for `examples/admin-demo` covering the new cards, so the README's showcase stays true.

## g) QUESTIONS I CANNOT ANSWER MYSELF (max 3)

1. **Impersonation: ship it or not?** The service supports it with real guards, and it is the highest-leverage missing user-management capability — but it is also a "act as this person" button. Do you want it in the admin panel behind an explicit opt-in `Config` (cookie name + super_admin + mandatory reason + banner), or is impersonation permanently out of scope for this library?
2. **May I add cross-module APIs?** Two fixes want usermgmt-side additions (`MembershipCountByActor`, `Tenant.CreatedAt`, session listing). That means a usermgmt tag + an adminui pin bump (the gotcha-30 root-tag-gap dance) instead of staying on published API. Should the admin panel keep working strictly against published API and accept the N+1/workaround, or is a small usermgmt API addition the right call?
3. **What is the next wave's priority?** The three unbuilt surfaces are (a) bots/API tokens, (b) sessions + impersonation, (c) import/export; and there is also (d) hardening this wave with e2e + coverage. Which one do you want next?

## h) VERIFICATION RECEIPTS (this session, on this host)

| Check | Command | Result |
| --- | --- | --- |
| adminui tests (workspace) | `GOEXPERIMENT=jsonv2 go test ./... -count=1` | PASS, 158 `--- PASS` lines |
| adminui tests (hermetic, CI shape) | `GOWORK=off GOEXPERIMENT=jsonv2 go test ./... -count=1` | PASS |
| setup tests | `GOEXPERIMENT=jsonv2 go test ./... -count=1` | PASS |
| lint | `golangci-lint run` (adminui) | 0 issues (exhaustruct deprecation warning is pre-existing) |
| route postures | `bash scripts/checks/check-route-postures.sh` | all modules green, adminui 4 manifest tests |
| CSS form gate | `bash scripts/checks/check-css-bundles.sh` | PASS (2 bundles canonical) |
| CSS class-set gate | `check-css-bundle-classes.sh` in fixture mode | exact sets (1,076 / 1,057 tokens) |
| CSS delta | class token diff old vs new | +`max-w-[220px]`, 0 removed |
| consumer builds | `setup`, `examples/admin-demo`, `examples/setup-demo`, `integration_test` | all `go build ./...` green |
| live smoke | `examples/admin-demo` on `:18930` + dev-login | tenant meta + person rows + datalist + access column + tenant-access card all render as intended |
| NOT RUN | `.#check-modules`, `.#coverage-gate`, `.#check-cqrs-lint`, `.#e2e*`, `.#build-adminui-css` | blocked: flake Go hash placeholder, no Chromium |

## i) FILES TOUCHED (all committed by the daemon; `git log 75e8e3af..51c49f15`)

`adminui/models.go`, `adminui/handler_members.go`, `adminui/handler_tenants.go`, `adminui/handler_users.go`, `adminui/routes.go`, `adminui/util.go`, `adminui/members.templ`, `adminui/tenants.templ`, `adminui/users.templ`, `adminui/components.templ` (+ regenerated `*_templ.go`), `adminui/README.md`, `adminui/CHANGELOG.md`, `adminui/assets/admin-tw.css`, `FEATURES.md`, and the new tests `adminui/tenant_ux_test.go`, `adminui/users_access_test.go`, `adminui/user_tenant_access_test.go`, plus edited `adminui/coverage_gaps_test.go`, `adminui/flows_test.go`, `adminui/users_filter_test.go`, `adminui/routes_manifest_test.go`.
