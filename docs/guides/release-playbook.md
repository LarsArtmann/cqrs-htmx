# Release Playbook

**Audience:** whoever cuts the next tag or family train. This is the
operator-facing summary; the deep details live in the linked documents.

## 0. The three rules

1. **Never raw-`git tag`.** Tag ONLY through
   `scripts/verify-tag.sh <module-dir> <version> [--push]` — it refuses
   uncommitted module trees, existing tag names (the module proxy caches
   version names forever), local-path replaces, pseudo-version requires, and
   unpublished internal requires.
2. **Tags point at commits, never the working tree.** Commit the fix FIRST,
   then tag. Rehearse with `--dry-run` (runs ALL guards, creates nothing).
3. **Verify facts at execution time.** `git ls-remote` / `git show <tag>:` —
   precondition tables rot within 24 hours (the dashboardui/v4.8.1 class).

## 1. Pre-tag checklist (per module)

```sh
# Hermetic verification of EXACTLY what the tag will contain (vet compiles
# tests — go build does not):
cd <module-dir>
GOWORK=off GOEXPERIMENT=jsonv2 go mod tidy
GOWORK=off GOEXPERIMENT=jsonv2 go build ./...
GOWORK=off GOEXPERIMENT=jsonv2 go vet ./...
GOWORK=off GOEXPERIMENT=jsonv2 go test ./...

# Dependency-tag content check the script CANNOT do for you — confirm the
# dep's existing tag actually contains the API you ship against:
git show <dep-repo>/<dep-tag>:<path/to/file> | grep <the symbol>
```

Then repo-wide gates from the root: `nix run .#check-modules -- --report`,
`nix run .#check-release-train`, `nix run .#build`, `.#test`, `.#lint`.

## 2. Tagging

```sh
scripts/verify-tag.sh <module-dir> vX.Y.Z --dry-run   # rehearse
scripts/verify-tag.sh <module-dir> vX.Y.Z             # local tag
scripts/verify-tag.sh <module-dir> vX.Y.Z --push      # + ls-remote confirm
```

Self-test of the guard itself (CI runs it too):
`bash scripts/test-verify-tag.sh` — the fixture corpus includes the REAL
poisoned `setup/v4.8.1` go.mod as a regression anchor.

Version numbers: the cqrs-htmx family cuts ONE coordinated version per train
(buildflow's gomod-check enforces it). First-time modules join the current
family version (health/auditlog precedent: first tag = v4.8.0).

## 3. Family train order

Follow `docs/runbooks/release-next-train-prep.md` — the ordered,
copy-pasteable train script. Shape:

1. Cut+push upstream-blocked or unpublished dependency tags first.
2. Strip DEV-ONLY family replaces (with removal-condition comments) from the
   modules being tagged, re-verify hermetically per module.
3. Tag leaves first (auth strategies), then aggregates (usermgmt, adminui,
   dashboardui), then setup, then root — require lines must resolve from the
   proxy at every step.
4. `nix run .#check-release-train` must read `0 unpublished / 0 lag` after.

### 3a. Wave-ordered choreography (the `--no-verify` killer)

The v4.12.0 train (2026-09-22, 14 tags) proved the refinement: sequence the
tags so that **every commit's `require` lines point at ALREADY-PUBLISHED
tags** — never at a tag you are ABOUT to cut. Concretely:

- Each tagging commit bumps its module's dependents' requires in the SAME
  commit only when those tags are already pushed; otherwise the bump waits
  for the dependent's own commit.
- Wave order: (1) leaves with no family dependents, (2) identity-model,
  (3) root + usermgmt + strategies, (4) adminui/dashboardui/loginpage/
  datastar/health/auditlog/systemadapter, (5) setup, (6) examples last.
- Verify per wave: `scripts/verify-tag.sh` (committed-tree + no-phantom-
  require checks) + `bash scripts/check-release-train.sh` (strict default =
  CI flags since 2026-09-22) after each wave, BEFORE the next.

Why: the pre-commit release-train gate fails a commit whose requires point
at unpublished tags — that is the design. A wave-ordered train never trips
it, so tag day needs NO `git commit --no-verify` (v4.12.0: zero skips).
Out-of-order trains are the ones that need the documented
`--no-verify`-with-justification fallback (AGENTS.md gotcha 6).

## 4. Poisoned-tag recovery ladder

Published tags cannot be repaired in place (the proxy caches the name
forever; force-moving a tag makes it WORSE):

1. Commit the fix, cut a SUPERSEDING version via verify-tag.
2. Add `retract` directives to the module's go.mod on master so the NEXT
   release retires the poisoned versions on pkg.go.dev / `go get`
   (worked example: `setup/go.mod`; upstream precedent: go-cqrs-lite
   storage/v4 v4.7.1).
3. Publish the retraction: `docs/runbooks/release-next-train-prep.md` §6b
   has the complete, execution-ready recipe (pre-tag gates, CHANGELOG entry,
   proxy/pkg.go.dev verification).

## 5. Post-train hygiene

- `bash scripts/check-module-isolation.sh` — 27 modules hermetic.
- `nix run .#check-modules -- --report` — all stages red/green.
- Update the runbook's §7 stage floor table + CHANGELOG (append-only;
  TODO_LIST keeps only open `[ ]`/`[~]` items).
- pkg.go.dev spot-check the new tags (license + docs render — the LICENSE
  files land per-tag).

## 6. Daemon attribution (who authored which commit)

The auto-commit daemon absorbs dirty files every 30-60 s, so long sessions
land mostly as `chore: auto-commit N changed file(s) (heuristic)` commits.
When you need to know what a heuristic commit actually contains:

- `git show <sha> --stat` groups by file; combine with
  `git log --format='%h %ad %s' --date=iso -- <paths>` to reconstruct a
  task's real history.
- Deliberate narrative commits survive only when made AT green boundaries
  with named files; batched end-of-session commits get shredded.
- The dashboardui Run-2/Run-3 work (2026-09-17..19) is attributed this way:
  heuristic commits `490e802e` (go-datastar v0.6.0 sweep), `cb2d0033`
  (lint posture), `d2465ad8` (broadcast replace strip) carry reviewed,
  verified content despite their messages; the tagged releases
  (dashboardui/v4.10.0, v4.10.1) are the authoritative artifacts.

## See Also

- [release-next-train-prep runbook](../runbooks/release-next-train-prep.md) — the concrete train script + tag protocol (§6) + retraction recipe (§6b)
- [v5 removal inventory](v5-removal-inventory.md) — what the NEXT breaking cut must remove
- [leveraging-go-cqrs-lite](leveraging-go-cqrs-lite.md) — upstream capability map (release-adjacent constraints)
