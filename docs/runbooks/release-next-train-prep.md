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
