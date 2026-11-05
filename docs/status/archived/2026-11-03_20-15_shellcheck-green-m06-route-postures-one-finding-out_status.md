# Status: Shellcheck red fixed + CI green; M06 route postures built, one lint finding from committed

> **ANNOTATED 2026-11-04** — ~~this report~~ **SUPERSEDED** by [`2026-11-04_00-04_m06-landed-train-armed-blocked-on-concurrent-root-edit_status.md`](2026-11-04_00-04_m06-landed-train-armed-blocked-on-concurrent-root-edit_status.md), which both section-g questions answered by the owner GO; M06 finished, committed 119e0f31, and the thread moved to train assembly. Strike depth:
> supersession verdict (round-18 precedent for series-internal reports); nothing here is a
> unique open item. **ARCHIVED 2026-11-04 (round-23 (2026-11-04)).**

> Session 2026-11-03 ~18:56–20:15 (resumed from the 18-31 WAIT state; the
> owner's "keep going until done" answered the predecessor's §g questions).
> Thread: authorization level-up. Predecessor:
> [`2026-11-03_18-31_swept-gated-pushed-ci-answers-shellcheck_status.md`](2026-11-03_18-31_swept-gated-pushed-ci-answers-shellcheck_status.md).

## a) Fully done (committed and pushed unless noted)

1. **The CI Shellcheck red is fixed and CI is GREEN end-to-end.** Triaged run
   `41675571500` to three findings (no local aggregate could ever see them):
   the dead `--gate` flag on `check-readme-badges.sh` (SC2034) and two
   literal-backtick SC2016 sites (`check-permission-maturity.sh:48`,
   `test-check-permission-maturity.sh:111`). The `--gate` flag is now REAL
   (it removes the CI fetch-failure downgrade — hard exit 4 — with a new
   selftest case); the SC2016s carry documented disables in the established
   form. Commit `517de987`.
2. **The CI/local parity gap is closed permanently.** New
   `scripts/checks/check-shellcheck.sh` gate (same file set as CI's docker
   step; PATH → `SHELLCHECK_BIN` → `nix run nixpkgs#shellcheck` fallback) +
   4-case fixture selftest, wired into check-modules (BOTH stage lists,
   parity self-test green) + `pkgs.shellcheck` in the app's runtimeInputs and
   the devShell (silences BuildFlow's "not in project devShell" warning).
   Commits `517de987`, `4473ddb5`. The gate proved itself immediately: it
   caught three violations in files I authored minutes earlier.
3. **Pushed and verified to green.** `43525f22..4473ddb5` pushed via
   push-and-watch; run `41691587152` finished with every job ✓ (test,
   build, security, mod-tidy, all 11 lints, checks incl. "Shellcheck gate
   scripts", module-architecture 6m33s).
4. **AGENTS.md memory updated** (Gates row documents the shellcheck leg).

## b) Partially done — M06 route postures (UNCOMMITTED, ~95%)

Implemented in the working tree, all five modules' test suites green:

- **Root**: `route_posture.go` — `Posture` (`public`/`authenticated`/
  `role-gated`/**`ungated`** = "this layer adds no gate", the honest value
  for delegated layers) + `Valid()`/`Provable()` + `RoutePosture{Method,
  Pattern, Posture, Why}` + tests.
- **authztest**: aliases to the root vocabulary (API-stable — old spellings
  are the same values); `Verify` REFUSES `ungated` (fail-closed) and `probe`
  carries a defense-in-depth case; new rejection test.
- **adminui**: route registration is a declarative table now
  (`sharedRoutes`/`superAdminRoutes`/`tenantAdminRoutes`/`fallbackRoutes`);
  mux AND manifest generate from it (`Handler.Routes()`); mode-scoped rows
  absent in the other mode; a forgotten mode = served by NO mode (fail-closed
  against silent over-exposure); `Config.AuthorizerPosture` declares
  custom-authorizer truth (lying values rejected at `New`). Golden tests per
  mode + fallback-semantics regression probes (guarded 404 catch-all,
  methodless 405, exact-match root, public assets). Old imperative `routes()`
  and `guardPanel` deleted.
- **dashboardui**: `Route.Posture` + `Why`, derived per instance (probes →
  public; authorizer → role-gated; none → ungated; versionz per config).
- **loginpage**: `Handler.Routes()` single public row + test.
- **setup**: `Bundle.RoutePostures()` aggregate (one row per mounted layer,
  honoring Disable*/paths; dashboard refined to role-gated exactly when
  `DashboardAuthorizer` is set) + 4 tests.
- **Gate**: `check-route-postures.sh` (workspace mode ON PURPOSE — GOWORK=off
  compiles submodule deps against the last published root tag, which predates
  the new symbols) + 5-case offline selftest; wired into check-modules (both
  lists) and a new CI checks-job step; refuses the empty-match trap.
- **Docs**: `docs/guides/route-posture.md`; PERMISSIONS cqrs-htmx L3→L4 with
  baseline regenerated (ratchet green); CHANGELOG entries (root, adminui,
  dashboardui, loginpage); AGENTS.md authorization bullet extended;
  CORPUS_BASELINE 38908→38990.

**What remains before commit**: exactly ONE lint finding —
`adminui/routes.go superAdminRoutes` is 79>60 lines after golines re-wrapped
it (fix: split users vs tenants builders, keep each ≤~50 pre-wrap lines).
Then: re-lint adminui, `nix run .#check-modules`, `.#lint`, `.#test`,
commit, push-and-watch. Note: the daemon will likely auto-commit the tree
before resume — check `git status`/`git log` first and build on whatever
landed; do NOT redo work.

**Also not done from the todo**: `verify-remote-tags` for the six-tag wave
(ran out of session before M06 finished).

## c) Not started

- **Consumer waves** (each its own repo): browser-history (M09
  `reconcileSuperAdmins`→`GrantRoles`, delete the delete-loop, wire
  `authztest.Suite`, bump modules), InboxClean (delete `sseDeadlineFree`),
  SwettySwipperWeb (durable idempotent bootstrap), crm (seed admin or
  `DisableAdmin`). All confirmed present locally.
- **Family train**: root gained exported symbols (`Posture`, `RoutePosture`)
  that submodule go.mods cannot see hermetically until the next
  root+adminui+dashboardui+loginpage train.

## d) Totally fucked up

Nothing irrecoverable this session. No force-pushes, no reverts, no lost
work. Two self-inflicted-and-caught items: an impossible-condition guard
written then deleted (`Public && Ungated`) and a registration-mapping order
bug that would have double-registered `/{$}` (caught in self-review before
any run). **Concurrent session notice**: commit `63206fff` (go-cqrs-lite
utilization audit) plus staged `TODO_LIST.md` + modified `FEATURES.md` are
ANOTHER session's work, untouched by me; my report commit uses explicit
pathspecs so their staging is not swept in.

## e) What I could have done better

1. **funlen/golines ping-pong**: splitting a too-long function and then
   letting golines re-wrap it past the limit again wasted two lint cycles.
   Rule going forward: apply golines FIRST, then judge funlen; keep table
   builders ≤50 pre-wrap lines so wrapping cannot push them over.
2. **Golden-before-truth**: my adminui manifest golden asserted members rows
   in SUPER-admin mode — the implementation was right, my golden was a guess.
   Derive the golden from actual behavior, THEN eyeball it for wrongness.
3. **The GOWORK=off reflex**: I copy-pasted hermetic mode into the new gate
   without asking what it resolves; submodule deps lag the family train by
   design, so the gate must run workspace-mode. Ask "what does this mode
   compile against" before inheriting flags.

## f) Next (ordered)

1. Fix the last adminui finding (split `superAdminRoutes` into users/tenants
   builders), re-lint, re-run its tests.
2. Run `nix run .#check-modules` (86 stages incl. the two new legs),
   `.#lint`, `.#test`; fix anything red.
3. Commit M06 (explicit pathspecs; the concurrent session's
   TODO_LIST/FEATURES stay out), push-and-watch to green.
4. `nix run .#verify-remote-tags 'v4.15.0' 'identity-model/*' 'usermgmt/*'
   'dashboardui/*' 'adminui/*' 'setup/*'` (the deferred six-tag audit).
5. Family train decision (see g2): if GO, `train-preflight` → wave-tag the
   five changed modules.
6. Consumer wave 1 — browser-history: `reconcileSuperAdmins` →
  `Service.GrantRoles`, delete the delete-loop, wire `authztest.Suite`
   against its real mux, bump cqrs-htmx modules, its tests green, commit.
7. Consumer wave 2 — InboxClean: delete `sseDeadlineFree` (library clears
   the deadline now), tests green, commit.
8. Consumer wave 3 — SwettySwipperWeb: grant on every boot idempotently,
   drop the `Count()>0` guard. <!-- fresh:skip: routed — the authorization row's consumer-waves item owns this wave; this item's only distinctive anchor was the frequency-fragile token "grant" (substring of GrantRoles), which any new CHANGELOG line mentioning GrantRoles unbinds. Same routing is proven by the sibling items carrying the durable "bootstrapAdmin" anchor -->
9. Consumer wave 4 — crm: seed an admin via `GrantRoles` or `DisableAdmin`;
   update its PERMISSIONS row.
10. Update `docs/PERMISSIONS.md` inventory rows as each consumer lands.
11. Consider teaching `Posture`/manifests in `.agents/skills/cqrs-htmx/SKILL.md`
    (check-skill gates after).
12. e2e surface check for the adminui table refactor (e2e/server hits admin
    routes; `.#test` excludes e2e — run the e2e battery once before the
    train).

## g) Questions

1. **M06 commit on resume is one finding away** — finish + gates + commit +
   push as planned, or do you want to review the uncommitted tree first
   (noting the daemon may auto-commit it meanwhile)?
2. **Train timing**: the new root symbols are invisible to submodule
   hermetic builds until a family train ships (root + adminui + dashboardui +
   loginpage + setup). Cut that train right after M06 lands, or batch it
   with the consumer-wave bumps?
3. **Consumer repos**: confirm this thread may write/commit/push in
   browser-history, InboxClean, SwettySwipperWeb, and crm directly, and
   whether each repo's own CI must be green before moving to the next.
