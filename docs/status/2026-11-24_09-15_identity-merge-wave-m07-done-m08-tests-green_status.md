# Status: identity-merge wave — M07 closed, M08 tests green, cqrs-lint receipt = expected tag-gap red

- **Date:** 2026-11-24 09:15 CET (report #5 today)
- **Session arc:** 09:00–09:15. Owner resumed with "NOW GET SHIT DONE! The WHOLE TODO LIST!". Executed the defaults from the pause (no Q1–Q3 answers): lint fix → M07 tests → plan bookkeeping → M07 close battery → M08 production + tests. Interrupted for this report.
- **Tree at report:** daemon committed my work in `411b2379`, `6856e5c2`, `f9acdcf5`, `f0925914`; uncommitted in tree: `usermgmt/es_bot_readmodel.go` (identitymodel import), `usermgmt/es_projection_merge_test.go` (casbin test fixes, just verified green). Untracked `scripts/checks/check-adminui-surface.sh` is the SIBLING session's file — not touched.

## a) FULLY DONE (this session, all verified by command)

| Item | Evidence |
| --- | --- |
| 7 M07 lint warnings fixed (4 nilnil, 1 exhaustruct, 2 nolintlint) + 1 new golines they caused | `buildflow -s "golangci-lint [usermgmt]"` → exactly the 3 expected `unused` (tenant constants, clear at M11); `go build && go vet ./usermgmt/` OK |
| nilnil fixes typed, not suppressed | `storedUser` → `(user *User, live bool, err error)`; `liveAfterMerge` → `(view *UserView, live bool, err error)`; callers in `retireToSQL`/`FindByEmailSQL` updated (`sql_readmodel.go`) |
| exhaustruct fix | `newUserIndex` literal now lists `merges: make([]mergeRecord, 0)` (`sql_hydrate.go`) |
| **The 11 owed M07 test packages** — new `usermgmt/sql_readmodel_merge_test.go`, 12 tests, **all PASS first run** | `go test ./usermgmt/ -run 'TestSQLUserReadModel_Merge' -count=1 -v`: 12/12 PASS |
| Deterministic event times for hydrate tests | `makeUserEvent` gained variadic `event.Option`; `userStream.atTime()` pins occurred-at (per-stream, minute-spaced); no globals (gochecknoglobals clean) |
| Full usermgmt suite after M07 | `go test ./usermgmt/ -count=1`: green except the known foreign tripwire (`TestSnapshotShape_SQLStoreDropsStampToday`, flips at M23) |
| `-race` full usermgmt run | same single tripwire red; zero races |
| Plan bookkeeping (the missing package 13) | §4: D22 + D23 appended; new "Accepted divergences (R1–R4, with G9)" section; §7 header states 145 rows; §10: M07 DONE row with test evidence |
| **M08 production code** | `es_casbin_projection.go`: `UserMergedInto` → `RemoveAllRolesForUser`; `TenantMergedInto` → `removeAllRolesForDomain`; both added to `EventTypes()`. `es_bot_readmodel.go`: `handleBotTransferred` (owner moves, token/scopes stay, no-row = no-op) + handler-map entry. Per D6: `identitymodel.*Payload` referenced directly, no new aliases |
| Guard test widened | `TestReadModels_HandleEveryDeclaredEventType` now has a `bot` row (would have caught a missing BotTransferred handler) — PASS |
| **M08 tests** — new `usermgmt/es_projection_merge_test.go`, 6 tests, all PASS | `go test -run 'TestCasbinProjection_|TestBotReadModel|TestSQLBotReadModel|TestAuditLog_Labels|TestReadModels_HandleEvery'`: all PASS (casbin pair verified green in the final run before this report) |
| Audit labels (task 47) verified as already-landed | constants + `auditActions` rows + `allAuditedEventTypes` membership asserted by `TestAuditLog_LabelsMergeEvents` |
| cqrs-lint gate run + triaged | `nix run .#check-cqrs-lint`: usermgmt FAILs on `undefined: identitymodel.*` findings — **verified phantom class**: every named symbol exists locally (`MergeContribution` at identity-model/commands.go:204, M03 constants/events), `go build ./usermgmt/` green. Cause: per-module hermetic walk resolves identity-model at usermgmt's PUBLISHED go.mod tag, which predates this wave's symbols (the gotcha-30 root-tag-gap class). Heals when the identity-model train lands. Other 25 modules: no findings reported |

## b) PARTIALLY DONE

- **M08 close-out:** production + tests green; still owed: M08 row in plan §10, one `buildflow` lint pass over the two new/edited files, one full-suite re-run including the new M08 tests.
- **M05:** registrations/postures landed; bijection gate stays red until M11 (3 tenant commands) — by design.
- **M18:** setup `Server-Timing` — untouched this session; the sibling's test-first red (`TestRunWithAppkit_ServerTimingHeaderDefaultAndOptOut`, 2 of 3) still waits for my M18 change.

## c) NOT STARTED (this session)

M09–M25 entirely: service merge APIs (M09, then K1), tenant deciders/registrations/read models (M10–M12, then K2), adminui surfaces (M13–M15), systemadapter/gfl parity (M16–M17), issues #43/#45 (M18–M19), docs/counts/ADR/gotchas/ratchets/gates (M20–M23, then K3), e2e/push (M24–M25).

## d) TOTALLY FUCKED UP (root-caused, all fixed same session)

1. **Whole-file signature bug:** wrote `sql_readmodel_merge_test.go` calling `applyToSQL(t, evt)` — missing the model argument at **10 callsites** — because I wrote the helpers and the test bodies in different passes without re-checking my own signature. One compile round wasted. Fix: multiedit all sites.
2. **Test contradicting my own just-written D23:** the pointer-only-tombstone test retired the same ghost stream twice and expected the pointer to follow the SECOND retirement — the tombstone-first design keeps the FIRST. Caught by re-reading before running; fixed by splitting into two ghost streams (live-survivor pointer, unknown-survivor pointer).
3. **Casbin assertions written blind:** asserted exact group-policy row counts without inspecting row shapes — reality: every granted domain also carries mirrored role-hierarchy rows, and my test used the loser as its own member actor (same raw subject ⇒ an extra matching row). Debug print proved both; fixed with a separate member user + hierarchy-aware assertions. (I even had gotcha 45 in memory — should have printed rows FIRST.)
4. **One edit round on a shadowed variable:** renaming a stream to `member` collided with the `member` event variable — compile caught it. Fixed (`memberAdded`).
5. **Lint fix caused a new lint finding:** the re-typed `liveAfterMerge` signature exceeded golines 120 — one extra lint round. Should have counted columns before writing the one-liner.

## e) WHAT WE SHOULD IMPROVE

- **Compile per test function, not per file** — the 10-callsite class would have been a 1-site fix if I had built after the first test.
- **Inspect external-system shapes (casbin rows, store rows) before writing count assertions** — same lesson as gotcha 64 (fixtures mirror reality); a 5-second debug print beats a red test.
- **Never leave a background gate running past its moment:** the cqrs-lint job (started 09:08) was only collected at 09:14 because the report forced the check. A gate whose result is never read is not a gate.
- **Todos tool hygiene:** recreated the list but did not tick items as milestones closed — stale board mid-session.
- The two `edit` failures from gofmt-touches-file-after-read are the known one-editing-path rule; internalize: view after every bash-side format.

## f) NEXT (ordered, from the plan of record; up to 50)

1. M08 close-out: plan §10 M08 row, `buildflow` lint over usermgmt, full suite re-run.
2. M09 (tasks 49–59): `Service.MergeUsers` — validation; load survivor/losers with typed rejections (not found, merged, deleted); third-party external-account conflict naming both rows; build contributions from the read model; dispatch survivor command; membership move (role union, then loser removal); bot transfer per loser; revoke loser sessions; loser tombstone LAST; the issue #46 verify scenario test (9 rows → 1); idempotent re-run, conflict, membership/bot/session tests; **K1**.
3. M10 (60–65): `decideRenameTenant`, `decideMergeTenants`, `decideMergeTenantInto`, `decideCreateTenant` recreate-on-plain-tombstone vs conflict-on-merged; exported wrappers; tests.
4. M11: tenant command registrations + postures + read model handlers; **bijection gate goes green** (27/27); the 3 `unused` lint warnings clear.
5. M12: tenant read models (rename/merge into views + hydrate) + `Service.RenameTenant`/`MergeTenants`/`MergeTenantInto`; **K2**.
6. M13–M14: adminui users-merge + tenants rename/merge routes/templ (defaults: new files only, one-line hooks after fresh read, regen only own templates).
7. M15: adminui manifest/route-posture pins, duplicates sections on list pages.
8. M16: systemadapter TypeDecoder/DomainConfig for the 6 new event types + projections — clears the parity red.
9. M17: gfl-usermgmt bindings/artifacts regen.
10. M18: issue #43 — appkit Server-Timing knob + docs (makes the sibling's red test green).
11. M19: issue #45 — legalparty-model bill_to aggregate.
12. M20: domain-counts re-true (FEATURES.md 33/27).
13. M21–M22: ADR for the merge design; gotchas entries; tail/corpus baselines re-true.
14. M23: snapshot tripwire flip (upstream shipped StateShape round-trip; TODO row 90); **K3** = every gate green.
15. M24–M25: e2e browser battery, pre-push hook, push master, CI watch (push gated on Q1/Q3 defaults).
16. Standing debts noticed this session: cqrs-lint tag-gap red heals only at the identity-model release train; corpus/tail ratchets red until M21/M22 by design.

## g) QUESTIONS (cannot figure out myself; defaults apply until answered)

1. **(Q1, fifth time) Release train ownership:** who cuts the train for this wave — may I run `nix run .#wave-tag` for identity-model/usermgmt when M23 is green, or do I stop at pushed+CI-green and hand you a tag checklist? **Default:** stop at pushed+CI-green, tag nothing. Note: the cqrs-lint phantom-red in usermgmt only heals at that train, so the default leaves one gate visibly red after K3.
2. **(Q2, fourth time) adminui sharing:** your sibling session's wave keeps editing adminui (`handler_audit.go`, `models.go`, `config.go`, `audit_labels.go`, `audit_filters_test.go`, now `scripts/checks/check-adminui-surface.sh` untracked). M13–M15 need routes + templ regen there. Confirm the default — new files only, one-line hooks into `routes.go` after a fresh read, regenerate only templates I wrote (`nix run .#gen` regenerates ALL templates in the module) — or give me a coordination rule. **Default:** defaults as stated, diff-check every rebase.
3. **(Q3, fourth time) Push scope:** master is now 50+ commits ahead mixing BOTH waves' files in daemon commits, with three known-red tests by design (bijection until M11, parity until M16, domain counts until M20) plus the M23 tripwire. Push the whole master only when every gate is green (K3), or split per-wave (impossible: daemon commits mix files — only separation is at push time by pathspec, which gotcha 65 warns about)? **Default:** whole master, only when all gates green.

**WAITING FOR INSTRUCTIONS.**
