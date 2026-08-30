# Runbook: Next Family Train — EXECUTED 2026-08-30

> **STATUS: EXECUTED (2026-08-30, user-approved).** Deltas vs the prepared plan:
> - §1b/§1d were already done before execution (dashboardui/v4.8.1 and datastar/v4.8.1 found pushed).
> - §1a/§1c executed: `usermgmt/v4.8.1`, `usermgmt/totp/v4.8.1`, `usermgmt/oauth2/v4.8.1` cut+pushed (signed).
> - The planned setup/dashboardui v4.8.1 pair was UNUSABLE: the pushed `dashboardui/v4.8.1` was cut at a pre-SSEMaxReplay commit, and `setup/v4.8.1`/`v4.8.2` shipped with a require on that stale tag (v4.8.2 additionally tagged before the go.mod fix was committed — the tag points at commits, never the working tree). Superseded by `dashboardui/v4.8.2` + `setup/v4.8.3`; consider retracting setup/v4.8.1+v4.8.2 in the next release.
> - §2 executed (5 replaces stripped), §3 executed (admin-demo totp v4.7.0→v4.8.1), plus family-wide usermgmt→v4.8.1 and dashboardui→v4.8.2 require alignment.
> - §4 result: isolation/test/lint green, release-train exit 0 (exemptions 3→1); strict drift reduced to the 5 go-cqrs-lite sibling axes confined to systemadapter + examples/system-demo (upstream-blocked).

This document contains the exact, ordered commands for the next coordinated
family train, so the release itself is mechanical. Nothing here has been
executed.

## 0. Preconditions (verified 2026-08-29)

| Fact | Value |
| --- | --- |
| Remote max `usermgmt/v4` | v4.8.0 (v4.8.1 NOT pushed) |
| Remote max `dashboardui/v4` | v4.8.0 (v4.8.1 cut locally, NOT pushed) |
| `usermgmt/v4.8.0` has `Service.Journal()/EventBus()` | NO (verified via `git show`) |
| `dashboardui/v4.8.0` has `SSEMaxReplay` | NO (verified via `git show`) |
| Root `v4.8.1` (transport pkg) | published ✓ |
| `setup/go.mod` dev-replaces remaining | 2: `../usermgmt`, `../dashboardui` (root replace stripped 2026-08-29, hermetic build+vet+test green) |
| `examples/setup-demo/go.mod` dev-replaces | 3: `../../setup`, `../../usermgmt`, `../../dashboardui` (added 2026-08-29 for the ServiceConfig escape hatch) |

## 1. Tags that MUST be part of the train (release blockers)

These submodules carry unreleased API consumed by setup; without them the
setup dev-replaces cannot be stripped and `check-modules` strict version
drift stays red:

```sh
# 1a. usermgmt (Journal/EventBus accessors + everything on master)
git tag -a usermgmt/v4.8.1 -m "usermgmt v4.8.1: Journal()/EventBus() accessors" && git push origin usermgmt/v4.8.1

# 1b. dashboardui (SSEMaxReplay + dashboard master state)
git push origin dashboardui/v4.8.1   # tag already cut locally

# 1c. NEVER-PUBLISHED auth strategies — required by the train reminder in
#     TODO_LIST ("Decisions awaiting the user"):
git tag -a usermgmt/totp/v4.8.1 -m "totp v4.8.1" && git push origin usermgmt/totp/v4.8.1
git tag -a usermgmt/oauth2/v4.8.1 -m "oauth2 v4.8.1" && git push origin usermgmt/oauth2/v4.8.1

# 1d. Locally-cut datastar tag awaiting push (from the v4.8.0 train §4.8):
git push origin datastar/v4.8.1 2>/dev/null || true   # check tag exists first
```

## 2. After the tags are pushed — strip the dev-replaces

```sh
# setup/go.mod: remove BOTH replace blocks annotated "DEV-ONLY" and bump
# nothing (requires already at v4.8.1).
# examples/setup-demo/go.mod: remove all 3 DEV-ONLY replaces.
# Per-module hermetic verify recipe (vet is load-bearing — it compiles tests):
for m in setup examples/setup-demo; do
  (cd "$m" && GOWORK=off GOEXPERIMENT=jsonv2 go mod tidy && \
   GOWORK=off GOEXPERIMENT=jsonv2 go build ./... && \
   GOWORK=off GOEXPERIMENT=jsonv2 go vet ./...)
done
```

## 3. admin-demo totp pin — bump back to the train version

```sh
# examples/admin-demo/go.mod currently pins totp v4.7.0 (the published max).
# After 1c, align it with the family version:
cd examples/admin-demo
GOWORK=off GOEXPERIMENT=jsonv2 go mod edit -require=github.com/larsartmann/cqrs-htmx/usermgmt/totp/v4@v4.8.1
GOWORK=off GOEXPERIMENT=jsonv2 go mod tidy && GOWORK=off GOEXPERIMENT=jsonv2 go build ./...
```

## 4. Verification gates for the train commit

```sh
nix run .#check-modules        # drift stage must turn GREEN (splits dissolve)
nix run .#check-release-train  # 0 unpublished, replace-exempt count drops to 0
nix run .#build && nix run .#test && nix run .#lint
```

## 5. Go-appkit fold-in reminder (separate decision, ADR-001)

Not part of this train. Precondition: push the go-appkit wave cut at
`f938d65` (go-appkit TODO_LIST P1, user gate).

## 6. Tag protocol (MANDATORY — institutionalized after two poisoned tags)

The setup/v4.8.1 + v4.8.2 incident (2026-08-30): v4.8.1 required
dashboardui/v4.8.1, whose tag predated SSEMaxReplay (stale content);
v4.8.2 was tagged while the go.mod fix was still UNCOMMITTED (tags point
at commits, never the working tree). Both are published and poisoned;
they are superseded by v4.8.3 and carry retract directives on master
(take effect at the next setup tag). Ladder for a poisoned tag:

```sh
# 1. Commit the fix FIRST, then cut a superseding version (never force-move
#    a pushed tag — the module proxy caches version names forever).
scripts/verify-tag.sh <module-dir> <next-version> --push
# 2. Add retract directives to the module's go.mod on master (see
#    setup/go.mod for the worked example) so the NEXT release retires the
#    poisoned versions on pkg.go.dev / go get. Upstream precedent:
#    go-cqrs-lite retracted storage/v4.7.0.
```

ALWAYS tag through `scripts/verify-tag.sh <module-dir> <version> [--push]`
instead of raw `git tag`/`git push`. The script enforces: committed module
tree (tracked changes), no existing local/remote tag of that name, no
cqrs-htmx dev-replaces in the tagged go.mod, no larsartmann pseudo-version
requires, every internal require published on origin, and (on --push) a
post-push ls-remote verification. What it CANNOT check: whether a
dependency's existing tag carries the API you ship against — verify that
by hand with `git show <tag>:<path>` BEFORE tagging (the stale
dashboardui/v4.8.1 class).

Precondition discipline: re-derive every "current state" fact from
`git ls-remote` / `git show` at execution time. Two preconditions in the
2026-08-30 runbook were wrong within 24 hours of being written. Tables
above describe intent, not reality — verify, do not trust.

## 7. Post-execution stage floor (recorded 2026-08-30, post-sweep)

State after the 2026-08-30 lag sweep (366→0) and systemadapter
replace-strips — the first GREEN strict-drift since 2026-08-15:

| Stage | Status | Notes |
| --- | --- | --- |
| module-isolation | ✅ | 27 modules GOWORK=off |
| dep-budgets | ✅ | |
| go-toolchain | ✅ | 1.26.7 everywhere |
| version-drift --strict | ✅ | was RED on 5 go-cqrs-lite axes; upstream tagged (metaengine v4.12.0, sqliteengine v4.2.0, event v4.9.0, metadata v4.6.0, record v4.4.0) and requires aligned |
| release-train | ✅ | 0 unpublished / 1 replace-exempted (systemadapter projectionadapter, upstream v4.5.0 still untagged) / 0 lag (was 366) |
| replace-directives | ✅ | systemadapter keeps ONLY the projectionadapter replace |

Remaining replace inventory (all with removal conditions):
- systemadapter + examples/system-demo: `metaengine/projectionadapter/v4 =>
  local` — strip once projectionadapter v4.5.0+ is tagged (v4.4.1 is the
  current max; OccurredAt on EventWithID needs v4.5.0).
- setup: `go-appkit => ../../go-appkit` — spike-only (ADR-001), harmless to
  consumers, strip at the appkit fold-in decision.
